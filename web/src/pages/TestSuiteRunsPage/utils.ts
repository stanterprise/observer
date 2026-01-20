import type { TestStatus } from "@/types/common";
import type { Test } from "@/types/testCase";
import type { TestRun } from "@/types/testRun";
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

export const getRunStatus = (run: TestRun): TestStatus => {
  // Prioritize error states
  if (run.statistics) {
    const notRunTests =
      run.totalTests! -
      (run.statistics.passed +
        run.statistics.failed +
        (run.statistics.broken || 0) +
        (run.statistics.timedout || 0) +
        (run.statistics.interrupted || 0) +
        (run.statistics.skipped || 0) +
        (run.statistics.unknown || 0));

    if (notRunTests > 0) {
      return "RUNNING";
    } else {
      if (run.statistics.failed > 0) return "FAILED";
      if (run.statistics.broken && run.statistics.broken > 0) return "BROKEN";
      if (run.statistics.timedout && run.statistics.timedout > 0)
        return "TIMEDOUT";
      if (run.statistics.interrupted && run.statistics.interrupted > 0)
        return "INTERRUPTED";

      // Then check for success or skip (all tests completed)
      if (run.statistics.passed === run.totalTests && run.totalTests! > 0)
        return "PASSED";
      if (run.statistics.skipped === run.totalTests && run.totalTests! > 0)
        return "SKIPPED";
      // Check for active running tests (tests in progress)
      if (run.statistics.running && run.statistics.running > 0)
        return "RUNNING";

      // Check for unknown status (actual UNKNOWN status from backend)
      if (run.statistics.unknown && run.statistics.unknown > 0)
        return "UNKNOWN";
    }
  }

  return "UNKNOWN";
};
