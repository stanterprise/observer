import type { RunStatistics } from "./testRun";

export type EventType =
  | "test.begin"
  | "test.end"
  | "step.begin"
  | "step.end"
  | "suite.begin"
  | "suite.end"
  | "run.start"
  | "run.end";

export interface WebSocketTestData {
  id?: string;
  runId?: string;
  testCase?: {
    id?: string;
    name?: string;
    description?: string;
    runId?: string;
    testSuiteId?: string; // Changed from suiteId to match protobuf
    status?: number; // Numeric status code (0=UNKNOWN, 1=PASSED, 2=FAILED, etc.)
    metadata?: Record<string, any>;
    startTime?: string;
    endTime?: string;
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

export interface WebSocketRunData {
  runId?: string; // Primary field from model
  id?: string; // Fallback for compatibility
  name?: string;
  totalTests?: number;
  metadata?: Record<string, any>;
  startTime?: string;
  finishedAt?: string;
  updatedAt?: string;
  status?: string;
  statistics?: RunStatistics;
  testSuites?: unknown[]; // From protobuf
}

export interface WebSocketEvent {
  type: EventType;
  timestamp: string;
  data: unknown;
}
