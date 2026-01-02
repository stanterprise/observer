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
