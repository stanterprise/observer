import type { Test } from "@/types/testCase";
import type { TestSuite } from "@/types/testSuite";

export function assembleSuiteHierarchy(
  suites: TestSuite[],
  tests: Test[]
): TestSuite {
  const tempSuites = suites;
  const rootSuite = suites.find((suite) => !suite.parentSuiteId)!;

  return buildSuiteTree(rootSuite, tempSuites, tests);
}

function buildSuiteTree(
  suite: TestSuite,
  allSuites: TestSuite[],
  allTests: Test[]
): TestSuite {
  const children = allSuites.filter((s) => s.parentSuiteId === suite.id);
  const tests = allTests.filter((t) => t.suiteId === suite.id);

  return {
    ...suite,
    suites: children.map((child) => buildSuiteTree(child, allSuites, allTests)),
    tests: tests,
  };
}
