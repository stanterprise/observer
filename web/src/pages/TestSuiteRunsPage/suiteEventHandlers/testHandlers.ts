import type { WebSocketTestData, EventType } from "@/types/webSocket";
import type { TestStatus } from "@/types/common";
import type { TestRun } from "@/types/testRun";

export const handleUpdateRun = (
  data: WebSocketTestData,
  type: EventType,
  setRuns: React.Dispatch<React.SetStateAction<TestRun[]>>
) => {
  const testData = data as WebSocketTestData;
  console.log(
    "[handleUpdateRun] Raw data structure:",
    JSON.stringify(data, null, 2)
  );

  const runId =
    testData.testCase?.runId || testData.runId || testData.testRunId;

  console.log(
    "[handleUpdateRun] Processing event:",
    type,
    "runId:",
    runId,
    "data:",
    data
  );

  // Safely extract status - handle both string and non-string values
  let status: TestStatus | undefined;
  if (type === "test.end") {
    // Now receiving camelCase: testCase.status (numeric)
    const rawStatus = testData.testCase?.status;
    console.log(
      "[handleUpdateRun] Raw status value:",
      rawStatus,
      "type:",
      typeof rawStatus
    );

    // Protobuf enum mapping: 0=UNKNOWN, 1=PASSED, 2=FAILED, 3=SKIPPED, etc.
    if (typeof rawStatus === "number") {
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
    } else {
      status = "UNKNOWN";
    }
    console.log("[handleUpdateRun] Extracted status:", status);
  }

  if (runId) {
    setRuns((prevRuns) => {
      try {
        console.log(
          "[handleUpdateRun] Looking for run in",
          prevRuns.length,
          "runs"
        );
        const existingIndex = prevRuns.findIndex((r) => r.id === runId);
        console.log("[handleUpdateRun] Found run at index:", existingIndex);
        if (existingIndex >= 0) {
          const updated = [...prevRuns];
          const currentRun = { ...updated[existingIndex] };
          // Keep existing total from the current run state
          // Update statistics based on event type
          if (type === "test.begin") {
            console.log("[handleUpdateRun] Incrementing running count");
            currentRun.statistics!.running =
              (currentRun.statistics!.running || 0) + 1;
          } else if (type === "test.end") {
            console.log(
              "[handleUpdateRun] Processing test.end with status:",
              status
            );
            // Decrement running count
            if (
              currentRun.statistics!.running &&
              currentRun.statistics!.running > 0
            ) {
              currentRun.statistics!.running--;
            }
            // Increment appropriate status counter
            if (status) {
              console.log(
                "[handleUpdateRun] Updating counter for status:",
                status
              );
            }
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
          console.log(
            "[handleUpdateRun] Updated run statistics:",
            currentRun.statistics
          );
          return updated;
        }
        console.warn("[handleUpdateRun] Run not found with id:", runId);
        return prevRuns;
      } catch (error) {
        console.error("Error updating runs from WebSocket:", error);
        return prevRuns;
      }
    });
  } else {
    console.warn("[handleUpdateRun] No runId found in event data");
  }
};
