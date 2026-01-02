import type { TestSuite } from "@/types/testSuite";

export function assembleSuiteHierarchy(suites: TestSuite[]): TestSuite {
  const tempSuites = suites;
  const rootSuite = suites.find((suite) => !suite.parentSuiteId)!;

  return buildSuiteTree(rootSuite, tempSuites);
}

function buildSuiteTree(suite: TestSuite, allSuites: TestSuite[]): TestSuite {
  const children = allSuites.filter((s) => s.parentSuiteId === suite.id);

  return {
    ...suite,
    suites: children.map((child) => buildSuiteTree(child, allSuites)),
  };
}
