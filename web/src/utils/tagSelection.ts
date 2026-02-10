import type { TagTerritoryTest, TagInfo } from "@/types/tagTerritory";
import type { TestStatus } from "@/types/common";

/**
 * Check if a test status represents a failure state
 */
export function isFailedStatus(status: TestStatus): boolean {
  return (
    status === "FAILED" ||
    status === "BROKEN" ||
    status === "TIMEDOUT"
  );
}

/**
 * Calculate impact score for a tag based on:
 * - Number of tests with the tag
 * - Extra weight for failed tests
 * - Extra weight for slow tests
 */
export function calculateTagImpact(
  tag: string,
  tests: TagTerritoryTest[],
): number {
  const testsWithTag = tests.filter((t) => t.tags.includes(tag));
  const count = testsWithTag.length;

  if (count === 0) return 0;

  // Base score from count
  let score = count;

  // Bonus for failed tests (2x weight)
  const failedCount = testsWithTag.filter((t) =>
    isFailedStatus(t.status),
  ).length;
  score += failedCount * 2;

  // Bonus for slow tests (tests > 5 seconds get 1x weight)
  const slowCount = testsWithTag.filter((t) => t.durationMs > 5000).length;
  score += slowCount;

  return score;
}

/**
 * Select top N tags by impact score
 */
export function selectTopTags(
  tests: TagTerritoryTest[],
  maxTags: number,
): TagInfo[] {
  // Collect all unique tags
  const tagCounts = new Map<string, number>();
  tests.forEach((test) => {
    test.tags.forEach((tag) => {
      tagCounts.set(tag, (tagCounts.get(tag) || 0) + 1);
    });
  });

  // Calculate impact scores
  const tagInfos: TagInfo[] = [];
  for (const [tag, count] of tagCounts) {
    const impactScore = calculateTagImpact(tag, tests);
    tagInfos.push({
      name: tag,
      count,
      impactScore,
      color: generateTagColor(tag),
    });
  }

  // Sort by impact score descending, then by count, then by name
  tagInfos.sort((a, b) => {
    if (b.impactScore !== a.impactScore) {
      return b.impactScore - a.impactScore;
    }
    if (b.count !== a.count) {
      return b.count - a.count;
    }
    return a.name.localeCompare(b.name);
  });

  return tagInfos.slice(0, maxTags);
}

/**
 * Generate a deterministic color for a tag based on its name
 */
export function generateTagColor(tag: string): string {
  // Simple hash to generate a hue value
  let hash = 0;
  for (let i = 0; i < tag.length; i++) {
    hash = tag.charCodeAt(i) + ((hash << 5) - hash);
  }

  // Use hue for color, with decent saturation and lightness
  const hue = Math.abs(hash) % 360;
  const saturation = 65 + (Math.abs(hash >> 8) % 20); // 65-85%
  const lightness = 50 + (Math.abs(hash >> 16) % 15); // 50-65%

  return `hsl(${hue}, ${saturation}%, ${lightness}%)`;
}

/**
 * Get status color using Tailwind color classes
 */
export function getStatusColor(status: TestStatus): string {
  switch (status) {
    case "PASSED":
      return "#10b981"; // green-500
    case "FAILED":
      return "#ef4444"; // red-500
    case "SKIPPED":
      return "#6b7280"; // gray-500
    case "BROKEN":
      return "#dc2626"; // red-600
    case "TIMEDOUT":
      return "#f97316"; // orange-500
    case "INTERRUPTED":
      return "#f59e0b"; // amber-500
    case "RUNNING":
      return "#3b82f6"; // blue-500
    default:
      return "#9ca3af"; // gray-400
  }
}

/**
 * Calculate node mass based on test properties
 * Failed/broken tests and longer-duration tests have more mass
 */
export function calculateTestMass(test: TagTerritoryTest): number {
  let mass = 1;

  // Increase mass for failed tests
  if (isFailedStatus(test.status)) {
    mass += 2;
  }

  // Increase mass for slow tests (logarithmic scale)
  if (test.durationMs > 1000) {
    mass += Math.log10(test.durationMs / 1000);
  }

  // Increase mass for retried tests
  if (test.retries > 0) {
    mass += test.retries * 0.5;
  }

  return mass;
}
