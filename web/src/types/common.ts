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
