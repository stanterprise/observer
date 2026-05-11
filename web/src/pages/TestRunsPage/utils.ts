import type { TestStatus } from "@/types/common";
import type { Test } from "@/types/testCase";
import type { TestSuite } from "@/types/testSuite";

export function assembleSuiteHierarchy(
  suites: TestSuite[],
  tests: Test[],
): TestSuite {
  // Deduplicate suites by ID - if there are multiple suites with the same ID,
  // merge their testCaseIds arrays to avoid duplicate rendering
  const suiteMap = new Map<string, TestSuite>();

  for (const suite of suites) {
    if (suiteMap.has(suite.id)) {
      // Merge testCaseIds if the suite already exists
      const existing = suiteMap.get(suite.id)!;
      if (suite.testCaseIds && existing.testCaseIds) {
        // Deduplicate testCaseIds
        const mergedIds = new Set([
          ...existing.testCaseIds,
          ...suite.testCaseIds,
        ]);
        existing.testCaseIds = Array.from(mergedIds);
      } else if (suite.testCaseIds && !existing.testCaseIds) {
        // If existing has no testCaseIds yet, copy from the duplicate suite
        existing.testCaseIds = suite.testCaseIds;
      }
    } else {
      suiteMap.set(suite.id, { ...suite });
    }
  }

  const dedupedSuites = Array.from(suiteMap.values());

  // Log deduplication stats
  if (suites.length !== dedupedSuites.length) {
    console.log(
      `[assembleSuiteHierarchy] Deduplicated ${suites.length} suites to ${dedupedSuites.length}`,
    );
  }

  const rootSuite = dedupedSuites.find((suite) => !suite.parentSuiteId)!;

  return buildSuiteTree(rootSuite, dedupedSuites, tests);
}

function buildSuiteTree(
  suite: TestSuite,
  allSuites: TestSuite[],
  allTests: Test[],
): TestSuite {
  const children = allSuites.filter((s) => s.parentSuiteId === suite.id);
  const tests = allTests.filter((t) => t.suiteId === suite.id);

  return {
    ...suite,
    suites: children.map((child) => buildSuiteTree(child, allSuites, allTests)),
    tests: tests,
  };
}

export const getRunCompletionStatus = (
  status: TestStatus,
): TestStatus | "COMPLETED" => {
  switch (status) {
    case "PASSED":
      return "COMPLETED";
    case "FAILED":
      return "COMPLETED";
    case "SKIPPED":
      return "COMPLETED";
    case "BROKEN":
      return "COMPLETED";
    case "TIMEDOUT":
      return "COMPLETED";
    case "INTERRUPTED":
      return "INTERRUPTED";
    case "RUNNING":
      return "RUNNING";
    case "PENDING":
      return "PENDING";
    default:
      return "UNKNOWN";
  }
};
