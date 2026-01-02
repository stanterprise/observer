import type { TestStatus } from "./common";

export interface Test {
  id: string;
  runId: string;
  title: string;
  status: TestStatus;
  testSuiteRunId?: string;
  description?: string;
  steps?: Step[];
  startTime?: string;
  endTime?: string;
  errorMessage?: string;
  stackTrace?: string;
  errors?: string[];
  metadata?: Record<string, string>;
  tags?: string[];
  location?: string;
  timeout?: number;
  duration?: number;
  retryCount?: number;
  createdAt?: string;
  updatedAt?: string;
}

export interface Step {
  id: string;
  runId: string;
  testCaseRunId: string;
  title: string;
  description?: string;
  startTime?: string;
  duration?: number;
  type?: string;
  steps?: Step[];
  metadata?: Record<string, string>;
  parentStepId?: string;
  workerIndex?: string;
  status?: TestStatus;
  error?: string;
  errors?: string[];
  location?: string;
  category?: string;
}
