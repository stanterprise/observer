import type { WebSocketRunData } from "@/types/webSocket";
import type { TestRun } from "@/types/testRun";
import { getTestStatus } from "@/pages/TestRunDetailPage/utils";

export const handleStartRun = (
  data: WebSocketRunData,
  setRuns: React.Dispatch<React.SetStateAction<TestRun[]>>
) => {
  console.log(
    "[handleStartRun] Raw data structure:",
    JSON.stringify(data, null, 2)
  );
  // Protobuf JSON name is "runId" (from json=runId in proto)
  const runId = (data as any).runId || data.id;
  console.log(
    "[handleStartRun] Processing run.start with runId:",
    runId,
    "data:",
    data
  );

  if (runId) {
    setRuns((prevRuns) => {
      console.log("[handleStartRun] Current runs count:", prevRuns.length);
      // Check if run already exists
      const existingRun = prevRuns.find((r) => r.id === runId);
      if (existingRun) {
        console.log("[handleStartRun] Run already exists, skipping");
        return prevRuns;
      }

      console.log("[handleStartRun] Creating new run");
      const newRun: TestRun = {
        id: runId,
        name: data.name || "Unnamed Run",
        status: getTestStatus(data.status || "unknown"),
        startTime: data.startTime || new Date().toISOString(),
        createdAt: new Date().toISOString(),
        updatedAt: data.updatedAt || new Date().toISOString(),
        totalTests: data.totalTests!,
        statistics: {
          total: data.totalTests!,
          passed: 0,
          failed: 0,
          skipped: 0,
          broken: 0,
          timedout: 0,
          interrupted: 0,
          running: 0,
          unknown: 0,
        },
        metadata: data.metadata || {},
      };

      console.log(
        "[handleStartRun] Returning updated runs with new run at start"
      );
      return [newRun, ...prevRuns];
    });
  } else {
    console.warn("[handleStartRun] No runId provided in data");
  }
};
