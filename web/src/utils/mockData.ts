import type { TagTerritoryTest } from "@/types/tagTerritory";

/**
 * Generate mock test data for the Tag Territory Map
 * This is useful for development and testing
 */

const TEST_TAGS = [
  "authentication",
  "api",
  "database",
  "frontend",
  "backend",
  "integration",
  "unit",
  "e2e",
  "smoke",
  "regression",
  "performance",
  "security",
  "user-management",
  "payment",
  "notifications",
  "search",
  "analytics",
  "reporting",
  "admin",
  "customer",
  "checkout",
  "cart",
  "product",
  "inventory",
  "orders",
  "shipping",
  "critical",
  "slow",
  "flaky",
  "ui",
];

const TEST_STATUSES = [
  "PASSED",
  "FAILED",
  "SKIPPED",
  "BROKEN",
  "TIMEDOUT",
] as const;

/**
 * Generate a random selection of tags for a test
 */
function selectRandomTags(count: number): string[] {
  const shuffled = [...TEST_TAGS].sort(() => Math.random() - 0.5);
  return shuffled.slice(0, count);
}

/**
 * Generate mock test data
 * @param count - Number of tests to generate
 */
export function generateMockTests(count: number): TagTerritoryTest[] {
  const tests: TagTerritoryTest[] = [];

  for (let i = 0; i < count; i++) {
    // Random number of tags (1-5)
    const tagCount = Math.floor(Math.random() * 5) + 1;
    const tags = selectRandomTags(tagCount);

    // Status distribution: 70% passed, 20% failed, 10% other
    let status: typeof TEST_STATUSES[number];
    const rand = Math.random();
    if (rand < 0.7) {
      status = "PASSED";
    } else if (rand < 0.9) {
      status = "FAILED";
    } else {
      status = TEST_STATUSES[Math.floor(Math.random() * TEST_STATUSES.length)];
    }

    // Duration: mostly 500-2000ms, some slow tests up to 10s
    const durationMs =
      Math.random() < 0.9
        ? Math.random() * 1500 + 500
        : Math.random() * 9000 + 1000;

    // Retries: most have 0, some have 1-3
    const retries = Math.random() < 0.9 ? 0 : Math.floor(Math.random() * 3) + 1;

    tests.push({
      id: `test-${i + 1}`,
      name: `Test Case ${i + 1}: ${tags[0]} functionality`,
      tags,
      status,
      durationMs,
      retries,
    });
  }

  return tests;
}

/**
 * Generate a realistic test run with common patterns
 */
export function generateRealisticTestRun(): TagTerritoryTest[] {
  const tests: TagTerritoryTest[] = [];

  // Create some test groups with common tag patterns
  const testGroups = [
    {
      prefix: "Auth",
      tags: ["authentication", "api", "backend", "critical"],
      count: 20,
    },
    { prefix: "UI", tags: ["frontend", "ui", "e2e"], count: 30 },
    {
      prefix: "API",
      tags: ["api", "backend", "integration", "regression"],
      count: 25,
    },
    {
      prefix: "Payment",
      tags: ["payment", "checkout", "critical", "api"],
      count: 15,
    },
    {
      prefix: "Search",
      tags: ["search", "performance", "database"],
      count: 12,
    },
    { prefix: "Admin", tags: ["admin", "ui", "frontend"], count: 18 },
    {
      prefix: "Notifications",
      tags: ["notifications", "backend", "integration"],
      count: 10,
    },
  ];

  let id = 1;
  for (const group of testGroups) {
    for (let i = 0; i < group.count; i++) {
      // Add some variation to tags
      const tags = [...group.tags];
      if (Math.random() < 0.3) {
        tags.push(TEST_TAGS[Math.floor(Math.random() * TEST_TAGS.length)]);
      }

      // Status distribution
      let status: typeof TEST_STATUSES[number];
      const rand = Math.random();
      if (rand < 0.75) {
        status = "PASSED";
      } else if (rand < 0.9) {
        status = "FAILED";
      } else {
        status =
          TEST_STATUSES[Math.floor(Math.random() * TEST_STATUSES.length)];
      }

      // Duration varies by group
      const baseDuration =
        group.prefix === "Performance" ? 5000 : group.prefix === "UI" ? 2000 : 1000;
      const durationMs = baseDuration + Math.random() * 1000;

      // Some tests have retries (more likely if failed)
      const retries =
        status === "FAILED" && Math.random() < 0.3
          ? Math.floor(Math.random() * 2) + 1
          : 0;

      tests.push({
        id: `test-${id}`,
        name: `${group.prefix} Test ${i + 1}`,
        tags,
        status,
        durationMs,
        retries,
      });

      id++;
    }
  }

  return tests;
}
