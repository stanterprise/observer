import type { TestStatus } from "./common";

// Attempt represents a single test attempt/retry
// Matches AttemptDocument from internal/models/document.go
export interface Attempt {
  retryIndex: number;
  steps?: Step[];
  status?: TestStatus;
  startTime?: string;
  endTime?: string;
  duration?: number;
  attachments?: Record<string, any>[];
  errorMessage?: string;
  stackTrace?: string;
  errorList?: string[];
  failures?: any[];
  errors?: any[];
  stdout?: any[];
  stderr?: any[];
  createdAt?: string;
  updatedAt?: string;
}

export interface Test {
  id: string;
  runId: string;
  title: string;
  status: TestStatus;
  suiteId?: string;
  description?: string;

  // Test-level timing (aggregated across attempts)
  startTime?: string; // Earliest start from first attempt
  endTime?: string; // Latest end from current attempt
  duration?: number; // Duration from current attempt

  // Retry tracking
  retryCount?: number;
  retryIndex?: number;
  timeout?: number;

  // Attempts array: sized to retry_count+1, indexed by retry_index
  attempts?: Attempt[];

  // DEPRECATED: Legacy fields (for backward compatibility)
  // New code should use attempts[retry_index] for step and error data
  steps?: Step[];
  errorMessage?: string;
  stackTrace?: string;
  errors?: string[];
  attachments?: Record<string, any>[];
  failures?: any[];

  metadata?: Record<string, string>;
  tags?: string[];
  location?: string;
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
  tags?: string[];
  parentStepId?: string;
  workerIndex?: string;
  status?: TestStatus;
  error?: string;
  errors?: string[];
  location?: string;
  category?: string;
  retryIndex?: number;
  createdAt?: string;
  updatedAt?: string;
}
