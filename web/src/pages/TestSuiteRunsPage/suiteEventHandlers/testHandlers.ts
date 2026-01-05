import type { WebSocketTestData, EventType } from "@/types/webSocket";
import type { TestStatus } from "@/types/common";
import type { TestRun } from "@/types/testRun";

export const handleUpdateRun = (
  data: WebSocketTestData,
  type: EventType,
  setRuns: React.Dispatch<React.SetStateAction<TestRun[]>>
) => {
  const testData = data as WebSocketTestData;

  // WebSocket sends TestDocument directly (not wrapped in testCase)
  // TestDocument has: id, runId, suiteId, status, name, etc.
  const runId =
    (data as any).runId ||
    testData.testCase?.runId ||
    testData.runId ||
    testData.testRunId;

  // Safely extract status - TestDocument.status is already a string
  let status: TestStatus | undefined;
  if (type === "test.end") {
    // TestDocument.Status field is converted to string via .String() in Go
    // JSON field is "status" (lowercase)
    const rawStatus = (data as any).status;

    if (typeof rawStatus === "string") {
      status = rawStatus.toUpperCase() as TestStatus;
    } else {
      status = "UNKNOWN";
    }
  }

  if (runId) {
    setRuns((prevRuns) => {
      try {
        const existingIndex = prevRuns.findIndex((r) => r.id === runId);

        if (existingIndex >= 0) {
          const updated = [...prevRuns];
          const currentRun = { ...updated[existingIndex] };

          // Ensure statistics object exists
          if (!currentRun.statistics) {
            currentRun.statistics = {
              total: 0,
              passed: 0,
              failed: 0,
              skipped: 0,
              running: 0,
              broken: 0,
              timedout: 0,
              interrupted: 0,
              unknown: 0,
            };
          }

          // Keep existing total from the current run state
          // Update statistics based on event type
          if (type === "test.begin") {
            currentRun.statistics.running =
              (currentRun.statistics.running || 0) + 1;
          } else if (type === "test.end") {
            // Decrement running count
            if (
              currentRun.statistics.running &&
              currentRun.statistics.running > 0
            ) {
              currentRun.statistics.running--;
            }

            switch (status) {
              case "PASSED":
                currentRun.statistics.passed =
                  (currentRun.statistics.passed || 0) + 1;
                break;
              case "FAILED":
                currentRun.statistics.failed =
                  (currentRun.statistics.failed || 0) + 1;
                break;
              case "SKIPPED":
                currentRun.statistics.skipped =
                  (currentRun.statistics.skipped || 0) + 1;
                break;
              case "BROKEN":
                currentRun.statistics.broken =
                  (currentRun.statistics.broken || 0) + 1;
                break;
              case "TIMEDOUT":
                currentRun.statistics.timedout =
                  (currentRun.statistics.timedout || 0) + 1;
                break;
              case "INTERRUPTED":
                currentRun.statistics.interrupted =
                  (currentRun.statistics.interrupted || 0) + 1;
                break;
              default:
                currentRun.statistics.unknown =
                  (currentRun.statistics.unknown || 0) + 1;
            }
          }
          currentRun.updatedAt = new Date().toISOString();
          updated[existingIndex] = currentRun;

          return updated;
        }

        return prevRuns;
      } catch (error) {
        return prevRuns;
      }
    });
  }
};
