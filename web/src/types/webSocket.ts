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

export interface WebSocketRunData {
  id?: string;
  name?: string;
  startTime?: string;
  finishedAt?: string;
  updatedAt?: string;
  status?: string;
  totalTests?: number;
  metadata?: Record<string, unknown>;
  statistics?: RunStatistics;
}

export interface WebSocketEvent {
  type: EventType;
  timestamp: string;
  data: unknown;
}
