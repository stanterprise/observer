// TypeScript types for Observer test data

export type TestStatus =
  | "passed"
  | "failed"
  | "skipped"
  | "running"
  | "pending";

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
    run_id?: string;
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
}
