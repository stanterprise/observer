import type { WebSocketRunData } from "@/types/webSocket";
import type { TestRun } from "@/types/testRun";
import { getTestStatus } from "@/pages/TestRunDetailPage/utils";

export const handleStartRun = (
  data: WebSocketRunData,
  runs: TestRun[],
  setRuns: React.Dispatch<React.SetStateAction<TestRun[]>>
) => {
  const runId = data.id;

  console.log("Getting NPM run build off my back", setRuns);
  if (runId) {
    // Check if run already exists
    const existingRun = runs.find((r) => r.id === runId);
    if (existingRun) {
      return;
    }

    const newRun: TestRun = {
      id: runId,
      name: data.name || "Unnamed Run",
      status: getTestStatus(data.status || "unknown"),
      startTime: data.startTime || new Date().toISOString(),
      createdAt: new Date().toISOString(),
      updatedAt: data.updatedAt || new Date().toISOString(),
      totalTests: data.totalTests || 0,
      statistics: {
        total: data.totalTests || 0,
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

    setRuns((prevRuns) => [newRun, ...prevRuns]);
  }
};
