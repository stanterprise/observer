import type { TestStatus } from "./common";

export interface TestCaseRun {
  id: string;
  testCaseId: string;
  testRunId: string;
  status: TestStatus;
  title: string;
  file: string;
  project: string;
  errorMessage?: string;
  metadata?: Record<string, unknown>;
  startedAt: string;
  finishedAt?: string;
  createdAt: string;
  updatedAt: string;
}

export interface StepRun {
  id: string;
  testCaseRunId: string;
  parentStepId?: string; // Reference to parent step for nested steps
  title: string;
  category: string;
  status: TestStatus;
  errorMessage?: string;
  metadata?: Record<string, unknown>;
  startedAt: string;
  finishedAt?: string;
  createdAt: string;
  updatedAt: string;
}

export interface TestRunStats {
  runId: string;
  total: number;
  passed: number;
  failed: number;
  skipped: number;
  running: number;
  broken?: number;
  timedout?: number;
  interrupted?: number;
  unknown?: number;
}

// WebSocket event types
export type EventType =
  | "test.begin"
  | "test.end"
  | "step.begin"
  | "step.end"
  | "suite.begin"
  | "suite.end"
  | "map.suites"
  | "run.end";

export interface WebSocketEvent {
  type: EventType;
  timestamp: string;
  data: unknown;
}

// API Response types
export interface TestCaseResponse {
  id: string;
  runId: string;
  title: string;
  status: string;
  metadata?: Record<string, unknown>;
  duration?: number;
  createdAt: string;
  updatedAt: string;
}

export interface WebSocketTestData {
  id?: string;
  runId?: string;
  testCase?: {
    id?: string;
    title?: string;
    name?: string;
    runId?: string;
    status?: string; // Status is in test_case for test.end events
    location?: {
      file?: string;
    };
    project?: string;
  };
  status?: string;
  startedAt?: string;
  finishedAt?: string;
  error?: {
    message?: string;
  };
  testRunId?: string;
}

export interface WebSocketStepData {
  testCaseRunId?: string;
  id?: string;
  parentStepId?: string;
  status?: string;
  category?: string;
  title?: string;
}

export interface Test {
  id: string;
  runId: string;
  title: string;
  parentSuiteId?: string;
  steps?: Step[];
  metadata?: Record<string, unknown>;
  owner?: string;
  author?: string;
  status?: TestStatus;
  location?: {
    file?: string;
  };
  startedAt?: string;
  finishedAt?: string;
  error?: {
    message?: string;
  };
}

export interface Step {
  id: string;
  runId: string;
  parentStepId?: string;
  title?: string;
  category?: string;
  steps?: Step[];
  metadata?: Record<string, unknown>;
  status?: TestStatus;
  startedAt?: string;
  finishedAt?: string;
}
