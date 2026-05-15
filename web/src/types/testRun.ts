import type { Test } from "./testCase";
import type { TestSuite } from "./testSuite";

export interface TestRun {
  id: string;
  name: string;
  description?: string;
  status?: string;
  metadata?: Record<string, unknown>;
  duration?: number; // Duration in nanoseconds
  totalTests?: number;
  initiatedBy?: string;
  projectName?: string;
  startTime?: string;
  endTime?: string;
  createdAt: string;
  updatedAt: string;
  statistics?: RunStatistics;
  executions?: RunExecution[];
  suites?: TestSuite[];
  tests?: Test[];
}

export interface RunExecution {
  id: string;
  runId: string;
  name?: string;
  status?: string;
  isShard?: boolean;
  shardIndex?: number;
  shardCountExpected?: number;
  metadata?: Record<string, unknown>;
  startTime?: string;
  endTime?: string;
  duration?: number;
  createdAt: string;
  updatedAt: string;
}

export interface RunStatistics {
  runId: string;
  name: string;
  pending: number;
  notRun: number;
  total: number;
  passed: number;
  failed: number;
  skipped: number;
  running?: number;
  broken?: number;
  timedout?: number;
  interrupted?: number;
  unknown?: number;
  lastUpdated?: string;
  expected?: number;
  flaky?: number;
  duration?: number; // Duration in nanoseconds
  createdAt?: string;
  updatedAt?: string;
}
