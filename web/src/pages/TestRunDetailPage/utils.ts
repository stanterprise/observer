import type { TestStatus } from "@/types/common";

export const getTestStatus = (status: string): TestStatus => {
  const statusMap: Record<string, TestStatus> = {
    PASSED: "PASSED",
    FAILED: "FAILED",
    SKIPPED: "SKIPPED",
    RUNNING: "RUNNING",
    UNKNOWN: "UNKNOWN",
    BROKEN: "BROKEN",
    TIMEDOUT: "TIMEDOUT",
    INTERRUPTED: "INTERRUPTED",
    NOT_RUN: "NOT_RUN",
  };
  return (statusMap[status] || "UNKNOWN") as TestStatus;
};

export const formatDuration = (nanoseconds?: number) => {
  if (!nanoseconds) return "N/A";
  const milliseconds = nanoseconds / 1000000;
  if (milliseconds < 1000) return `${milliseconds.toFixed(0)}ms`;
  return `${(milliseconds / 1000).toFixed(2)}s`;
};
