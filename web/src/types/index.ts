// TypeScript types for Observer test data

export type TestStatus =
  | "passed"
  | "failed"
  | "skipped"
  | "running"
  | "pending"
  | "unknown"
  | "broken"
  | "timedout"
  | "interrupted";

export interface TestCaseRun {
  id: string;
  test_case_id: string;
  test_run_id: string;
  status: TestStatus;
  title: string;
  file: string;
  project: string;
  error_message?: string;
  metadata?: Record<string, unknown>;
  started_at: string;
  finished_at?: string;
  created_at: string;
  updated_at: string;
}

export interface StepRun {
  id: string;
  test_case_run_id: string;
  parent_step_id?: string; // Reference to parent step for nested steps
  title: string;
  category: string;
  status: TestStatus;
  error_message?: string;
  metadata?: Record<string, unknown>;
  started_at: string;
  finished_at?: string;
  created_at: string;
  updated_at: string;
}

export interface TestRunStats {
  run_id: string;
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
export type EventType = "test.begin" | "test.end" | "step.begin" | "step.end";

export interface WebSocketEvent {
  type: EventType;
  timestamp: string;
  data: unknown;
}

// API Response types
export interface TestCaseResponse {
  ID: string;
  RunID: string;
  Title: string;
  Status: string;
  Metadata?: Record<string, unknown>;
  Duration?: number;
  CreatedAt: string;
  UpdatedAt: string;
}

export interface WebSocketTestData {
  id?: string;
  run_id?: string;
  test_case?: {
    id?: string;
    title?: string;
    name?: string;
    run_id?: string;
    status?: string; // Status is in test_case for test.end events
    location?: {
      file?: string;
    };
    project?: string;
  };
  status?: string;
  started_at?: string;
  finished_at?: string;
  error?: {
    message?: string;
  };
  test_run_id?: string;
}

export interface WebSocketStepData {
  test_case_run_id?: string;
  id?: string;
  parent_step_id?: string;
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
