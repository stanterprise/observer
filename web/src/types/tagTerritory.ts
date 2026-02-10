import type { TestStatus } from "./common";

/**
 * Tag Territory Map Types
 * For visualizing tests and tags in a force-directed layout
 */

export interface TagTerritoryTest {
  id: string;
  name: string;
  tags: string[];
  status: TestStatus;
  durationMs: number;
  retries: number;
}

export interface TagInfo {
  name: string;
  count: number;
  impactScore: number;
  color: string;
}

export interface TestNode {
  id: string;
  x: number;
  y: number;
  vx: number;
  vy: number;
  mass: number;
  radius: number;
  test: TagTerritoryTest;
}

export interface TagNode {
  name: string;
  x: number;
  y: number;
  vx: number;
  vy: number;
  mass: number;
  radius: number;
  testIds: Set<string>;
  info: TagInfo;
}

export interface SimulationState {
  tests: TestNode[];
  tags: TagNode[];
  iteration: number;
  isStable: boolean;
}

export interface SimulationConfig {
  width: number;
  height: number;
  seed: number;
  maxIterations: number;
  stabilityThreshold: number;
  forces: {
    testTagAttraction: number;
    tagTagRepulsion: number;
    centering: number;
    separation: number;
  };
}

export interface TagTerritoryMapProps {
  tests: TagTerritoryTest[];
  maxVisibleTags?: number;
  width?: number;
  height?: number;
  renderRegions?: boolean;
  onTestHover?: (test: TagTerritoryTest | null) => void;
  onTagClick?: (tag: string) => void;
}
