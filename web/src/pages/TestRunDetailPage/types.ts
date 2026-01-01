export interface TestCase {
  ID: string;
  RunID: string;
  Title: string;
  Status: string;
  Duration?: number;
  RetryCount?: number;
  CreatedAt: string;
  UpdatedAt: string;
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
  runId: string;
  tests: TestCase[]; // Note: lowercase 'tests' in response
  suites?: RunDetail[];
  statistics: RunStatistics;
  totalSteps: number;
}
