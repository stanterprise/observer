import type { Test } from "@/types/testCase";
import type { TestSuite } from "@/types/testSuite";

export interface TestCase {
  id: string;
  runId: string;
  title: string;
  status: string;
  duration?: number;
  retryCount?: number;
  createdAt: string;
  updatedAt: string;
}

export interface RunStatistics {
  total: number;
  passed: number;
  failed: number;
  skipped: number;
  running?: number;
  broken?: number;
  timedout?: number;
  interrupted?: number;
  unknown?: number;
}

export interface RunDetail {
  name: string;
  id: string;
  tests: Test[]; // Note: lowercase 'tests' in response
  suites?: TestSuite[];
  statistics: RunStatistics;
  totalSteps: number;
}
