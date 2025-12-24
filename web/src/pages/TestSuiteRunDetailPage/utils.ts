import type { TestStatus } from "../../types";
export const getTestStatus = (status: string): TestStatus => {
  const statusMap: Record<string, TestStatus> = {
    PASSED: "passed",
    FAILED: "failed",
    SKIPPED: "skipped",
    RUNNING: "running",
    UNKNOWN: "unknown",
    BROKEN: "broken",
    TIMEDOUT: "timedout",
    INTERRUPTED: "interrupted",
  };
  return (statusMap[status] || "unknown") as TestStatus;
};

export const formatDuration = (nanoseconds?: number) => {
  if (!nanoseconds) return "N/A";
  const milliseconds = nanoseconds / 1000000;
  if (milliseconds < 1000) return `${milliseconds.toFixed(0)}ms`;
  return `${(milliseconds / 1000).toFixed(2)}s`;
};
