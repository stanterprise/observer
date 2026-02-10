/**
 * Web Worker for Tag Territory Map physics simulation
 * Runs force-based layout in a separate thread to keep UI responsive
 */

import { SeededRandom } from "../utils/seededRandom";
import type {
  TestNode,
  TagNode,
  SimulationState,
  SimulationConfig,
  TagTerritoryTest,
  TagInfo,
} from "../types/tagTerritory";
import { calculateTestMass } from "../utils/tagSelection";
import { buildTagSimilarityMatrix, getTagSimilarity } from "../utils/tagSimilarity";

interface WorkerMessage {
  type: "init" | "step" | "stop";
  config?: SimulationConfig;
  tests?: TagTerritoryTest[];
  tags?: TagInfo[];
}

interface WorkerResponse {
  type: "state" | "complete" | "error";
  state?: SimulationState;
  error?: string;
}

let state: SimulationState | null = null;
let config: SimulationConfig | null = null;
let random: SeededRandom | null = null;
let animationId: number | null = null;
let tagSimilarityMatrix: Map<string, number> | null = null;

/**
 * Initialize the simulation with tests and tags
 */
function initialize(
  cfg: SimulationConfig,
  tests: TagTerritoryTest[],
  tags: TagInfo[],
): SimulationState {
  config = cfg;
  random = new SeededRandom(cfg.seed);

  const centerX = cfg.width / 2;
  const centerY = cfg.height / 2;

  // Create test nodes
  const testNodes: TestNode[] = tests.map((test) => ({
    id: test.id,
    x: centerX + random!.range(-100, 100),
    y: centerY + random!.range(-100, 100),
    vx: 0,
    vy: 0,
    mass: calculateTestMass(test),
    radius: 4 + Math.min(test.tags.length, 5), // Radius based on tag count
    test,
  }));

  // Build tag -> test mapping
  const tagToTests = new Map<string, Set<string>>();
  tags.forEach((tag) => {
    const testIds = new Set<string>();
    tests.forEach((test) => {
      if (test.tags.includes(tag.name)) {
        testIds.add(test.id);
      }
    });
    tagToTests.set(tag.name, testIds);
  });

  // Build similarity matrix
  tagSimilarityMatrix = buildTagSimilarityMatrix(tagToTests);

  // Create tag nodes (initial position is centroid of their tests)
  const tagNodes: TagNode[] = tags.map((tag) => {
    const testIds = tagToTests.get(tag.name) || new Set();
    const relatedTests = testNodes.filter((t) => testIds.has(t.id));

    let x = centerX;
    let y = centerY;

    if (relatedTests.length > 0) {
      x = relatedTests.reduce((sum, t) => sum + t.x, 0) / relatedTests.length;
      y = relatedTests.reduce((sum, t) => sum + t.y, 0) / relatedTests.length;
    }

    // Add some random offset to prevent initial overlap
    x += random!.range(-50, 50);
    y += random!.range(-50, 50);

    return {
      name: tag.name,
      x,
      y,
      vx: 0,
      vy: 0,
      mass: Math.sqrt(testIds.size) + 2, // Mass proportional to test count
      radius: 20 + Math.min(testIds.size * 2, 40), // Visual radius
      testIds,
      info: tag,
    };
  });

  return {
    tests: testNodes,
    tags: tagNodes,
    iteration: 0,
    isStable: false,
  };
}

/**
 * Calculate forces between all nodes
 */
function calculateForces(state: SimulationState, cfg: SimulationConfig): void {
  const { tests, tags } = state;

  // Reset forces
  tests.forEach((t) => {
    t.vx *= 0.85; // Damping
    t.vy *= 0.85;
  });
  tags.forEach((t) => {
    t.vx *= 0.85; // Damping
    t.vy *= 0.85;
  });

  // 1. Test -> Tag attraction
  tests.forEach((test) => {
    test.test.tags.forEach((tagName) => {
      const tag = tags.find((t) => t.name === tagName);
      if (!tag) return;

      const dx = tag.x - test.x;
      const dy = tag.y - test.y;
      const distance = Math.sqrt(dx * dx + dy * dy) + 0.1;

      const force = cfg.forces.testTagAttraction * (distance / 100);
      const fx = (dx / distance) * force;
      const fy = (dy / distance) * force;

      test.vx += fx / test.mass;
      test.vy += fy / test.mass;
      tag.vx -= fx / tag.mass;
      tag.vy -= fy / tag.mass;
    });
  });

  // 2. Tag <-> Tag repulsion (inversely proportional to similarity)
  for (let i = 0; i < tags.length; i++) {
    for (let j = i + 1; j < tags.length; j++) {
      const tag1 = tags[i];
      const tag2 = tags[j];

      const dx = tag2.x - tag1.x;
      const dy = tag2.y - tag1.y;
      const distance = Math.sqrt(dx * dx + dy * dy) + 0.1;

      // Get similarity (higher similarity = less repulsion)
      const similarity = getTagSimilarity(
        tagSimilarityMatrix!,
        tag1.name,
        tag2.name,
      );
      const repulsionFactor = 1 - similarity * 0.7; // 0.3 to 1.0 range

      const force =
        (cfg.forces.tagTagRepulsion * repulsionFactor) / (distance * distance);
      const fx = (dx / distance) * force;
      const fy = (dy / distance) * force;

      tag1.vx -= fx / tag1.mass;
      tag1.vy -= fy / tag1.mass;
      tag2.vx += fx / tag2.mass;
      tag2.vy += fy / tag2.mass;
    }
  }

  // 3. Separation force (prevent overlap)
  const allNodes = [...tests, ...tags];
  for (let i = 0; i < allNodes.length; i++) {
    for (let j = i + 1; j < allNodes.length; j++) {
      const node1 = allNodes[i];
      const node2 = allNodes[j];

      const dx = node2.x - node1.x;
      const dy = node2.y - node1.y;
      const distance = Math.sqrt(dx * dx + dy * dy) + 0.1;
      const minDistance = node1.radius + node2.radius;

      if (distance < minDistance) {
        const force = cfg.forces.separation * (minDistance - distance);
        const fx = (dx / distance) * force;
        const fy = (dy / distance) * force;

        node1.vx -= fx / node1.mass;
        node1.vy -= fy / node1.mass;
        node2.vx += fx / node2.mass;
        node2.vy += fy / node2.mass;
      }
    }
  }

  // 4. Centering force (gentle pull toward center)
  const centerX = cfg.width / 2;
  const centerY = cfg.height / 2;

  tests.forEach((test) => {
    const dx = centerX - test.x;
    const dy = centerY - test.y;
    test.vx += dx * cfg.forces.centering;
    test.vy += dy * cfg.forces.centering;
  });

  tags.forEach((tag) => {
    const dx = centerX - tag.x;
    const dy = centerY - tag.y;
    tag.vx += dx * cfg.forces.centering;
    tag.vy += dy * cfg.forces.centering;
  });
}

/**
 * Update node positions based on velocities
 */
function updatePositions(state: SimulationState): number {
  let maxVelocity = 0;

  state.tests.forEach((test) => {
    test.x += test.vx;
    test.y += test.vy;

    const velocity = Math.sqrt(test.vx * test.vx + test.vy * test.vy);
    maxVelocity = Math.max(maxVelocity, velocity);
  });

  state.tags.forEach((tag) => {
    tag.x += tag.vx;
    tag.y += tag.vy;

    const velocity = Math.sqrt(tag.vx * tag.vx + tag.vy * tag.vy);
    maxVelocity = Math.max(maxVelocity, velocity);
  });

  return maxVelocity;
}

/**
 * Run one simulation step
 */
function step(): void {
  if (!state || !config) return;

  calculateForces(state, config);
  const maxVelocity = updatePositions(state);

  state.iteration++;

  // Check for stability
  if (maxVelocity < config.stabilityThreshold) {
    state.isStable = true;
  }

  // Send state back to main thread
  const response: WorkerResponse = {
    type: state.isStable || state.iteration >= config.maxIterations ? "complete" : "state",
    state: state,
  };

  self.postMessage(response);

  // Continue simulation if not stable
  if (!state.isStable && state.iteration < config.maxIterations) {
    animationId = self.setTimeout(step, 16) as unknown as number; // ~60fps
  }
}

/**
 * Handle messages from main thread
 */
self.onmessage = (e: MessageEvent<WorkerMessage>) => {
  const { type, config: cfg, tests, tags } = e.data;

  try {
    switch (type) {
      case "init":
        if (cfg && tests && tags) {
          state = initialize(cfg, tests, tags);
          // Send initial state immediately
          const response: WorkerResponse = {
            type: "state",
            state: state,
          };
          self.postMessage(response);
        }
        break;

      case "step":
        if (!state) {
          const response: WorkerResponse = {
            type: "error",
            error: "Simulation not initialized",
          };
          self.postMessage(response);
        } else {
          step();
        }
        break;

      case "stop":
        if (animationId !== null) {
          self.clearTimeout(animationId);
          animationId = null;
        }
        break;
    }
  } catch (error) {
    const response: WorkerResponse = {
      type: "error",
      error: error instanceof Error ? error.message : "Unknown error",
    };
    self.postMessage(response);
  }
};
