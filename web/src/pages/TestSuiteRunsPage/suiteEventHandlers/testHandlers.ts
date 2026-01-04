import type { WebSocketTestData, EventType } from "@/types/webSocket";
import type { TestStatus } from "@/types/common";
import type { TestRun } from "@/types/testRun";

export const handleUpdateRun = (
  data: WebSocketTestData,
  type: EventType,
  runs: TestRun[],
  setRuns: React.Dispatch<React.SetStateAction<TestRun[]>>
) => {
  const testData = data as WebSocketTestData;
  const runId = testData.runId || testData.testCase?.runId;

  console.log("Getting NPM run build off my back", setRuns, runs);
  // Safely extract status - handle both string and non-string values
  let status: TestStatus;
  if (type === "test.end") {
    const rawStatus = testData.testCase?.status || testData.status;
    // Handle numeric status codes (protobuf enums) - need to map them
    if (typeof rawStatus === "number") {
      // Protobuf enum mapping: 0=UNKNOWN, 1=PASSED, 2=FAILED, 3=SKIPPED, etc.
      const statusMap: Record<number, string> = {
        0: "UNKNOWN",
        1: "PASSED",
        2: "FAILED",
        3: "SKIPPED",
        4: "BROKEN",
        5: "TIMEDOUT",
        6: "INTERRUPTED",
      };
      status = (statusMap[rawStatus] as unknown as TestStatus) || "UNKNOWN";
    } else if (typeof rawStatus === "string") {
      status = rawStatus.toUpperCase() as unknown as TestStatus;
    }
  }

  if (runId) {
    setRuns((prevRuns) => {
      try {
        const existingIndex = prevRuns.findIndex((r) => r.id === runId);
        if (existingIndex >= 0) {
          const updated = [...prevRuns];
          const currentRun = { ...updated[existingIndex] };
          currentRun.statistics!.total =
            runs.find((r) => r.id === runId)?.statistics!.total || 0;
          // Update statistics based on event type
          if (type === "test.begin") {
            currentRun.statistics!.running =
              (currentRun.statistics!.running || 0) + 1;
          } else if (type === "test.end") {
            // Decrement running count
            if (
              currentRun.statistics!.running &&
              currentRun.statistics!.running > 0
            ) {
              currentRun.statistics!.running--;
            }
            // Increment appropriate status counter
            switch (status) {
              case "PASSED":
                currentRun.statistics!.passed =
                  (currentRun.statistics!.passed || 0) + 1;
                break;
              case "FAILED":
                currentRun.statistics!.failed =
                  (currentRun.statistics!.failed || 0) + 1;
                break;
              case "SKIPPED":
                currentRun.statistics!.skipped =
                  (currentRun.statistics!.skipped || 0) + 1;
                break;
              case "BROKEN":
                currentRun.statistics!.broken =
                  (currentRun.statistics!.broken || 0) + 1;
                break;
              case "TIMEDOUT":
                currentRun.statistics!.timedout =
                  (currentRun.statistics!.timedout || 0) + 1;
                break;
              case "INTERRUPTED":
                currentRun.statistics!.interrupted =
                  (currentRun.statistics!.interrupted || 0) + 1;
                break;
              default:
                currentRun.statistics!.unknown =
                  (currentRun.statistics!.unknown || 0) + 1;
            }
          }
          currentRun.updatedAt = new Date().toISOString();
          updated[existingIndex] = currentRun;
          return updated;
        }
        return prevRuns;
      } catch (error) {
        console.error("Error updating runs from WebSocket:", error);
        return prevRuns;
      }
    });
  }
};
