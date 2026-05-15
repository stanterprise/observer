import type { TestStatus } from "@/types/common";

export const getTestStatus = (status: string): TestStatus => {
  const statusMap: Record<string, TestStatus> = {
    PASSED: "PASSED",
    FLAKY: "FLAKY",
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
