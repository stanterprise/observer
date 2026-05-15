import { apiUrl } from "@/lib/config";
import type { Test } from "@/types/testCase";
import type { TestRun } from "@/types/testRun";
import type { TestSuite } from "@/types/testSuite";

const inFlightRunDetailRequests = new Map<string, Promise<TestRun>>();

export async function fetchRunDetailData(id: string): Promise<TestRun> {
  const existingRequest = inFlightRunDetailRequests.get(id);
  if (existingRequest) {
    return existingRequest;
  }

  const request = fetch(apiUrl(`/runs/${id}`))
    .then(async (response) => {
      if (!response.ok) {
        throw new Error(`Failed to fetch run details: ${response.statusText}`);
      }

      return normalizeRunDetail(await response.json());
    })
    .finally(() => {
      inFlightRunDetailRequests.delete(id);
    });

  inFlightRunDetailRequests.set(id, request);
  return request;
}

export function normalizeRunDetail(data: TestRun): TestRun {
  const nestedSuites = data.suites ?? [];
  const suites = flattenSuites(nestedSuites);
  const tests = dedupeTests([
    ...(data.tests ?? []),
    ...flattenSuiteTests(nestedSuites),
  ]);

  return {
    ...data,
    suites,
    tests,
    statistics: computeStatistics(tests, data.id, data.name),
  };
}

function flattenSuiteTests(suites: TestSuite[]): Test[] {
  const flattened: Test[] = [];

  const visit = (suite: TestSuite) => {
    flattened.push(...(suite.tests ?? []));
    suite.suites?.forEach(visit);
  };

  suites.forEach(visit);
  return flattened;
}

function flattenSuites(suites: TestSuite[]): TestSuite[] {
  const flattened: TestSuite[] = [];

  const visit = (suite: TestSuite) => {
    flattened.push({
      ...suite,
      tests: [],
      suites: [],
    });
    suite.suites?.forEach(visit);
  };

  suites.forEach(visit);
  return flattened;
}

function dedupeTests(tests: Test[]): Test[] {
  const byKey = new Map<string, Test>();

  for (const test of tests) {
    byKey.set(`${test.id}:${test.suiteId ?? ""}`, test);
  }

  return Array.from(byKey.values());
}

function computeStatistics(tests: Test[], runId: string, name: string) {
  return {
    runId,
    name,
    total: tests.length,
    passed: tests.filter(
      (test) => test.status === "PASSED" || test.status === "FLAKY",
    ).length,
    flaky: tests.filter((test) => test.status === "FLAKY").length,
    failed: tests.filter((test) => test.status === "FAILED").length,
    skipped: tests.filter((test) => test.status === "SKIPPED").length,
    running: tests.filter((test) => test.status === "RUNNING").length,
    pending: tests.filter((test) => test.status === "PENDING").length,
    notRun: tests.filter((test) => test.status === "NOT_RUN").length,
    broken: tests.filter((test) => test.status === "BROKEN").length,
    timedout: tests.filter((test) => test.status === "TIMEDOUT").length,
    interrupted: tests.filter((test) => test.status === "INTERRUPTED").length,
    unknown: tests.filter((test) => test.status === "UNKNOWN").length,
    expected: tests.filter(
      (test) => test.status === "PASSED" && (test.attempts?.length ?? 0) === 1,
    ).length,
  };
}
