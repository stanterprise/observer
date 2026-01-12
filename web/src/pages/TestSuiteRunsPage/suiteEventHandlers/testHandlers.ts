import type { WebSocketTestData, EventType } from "@/types/webSocket";
import type { TestStatus } from "@/types/common";
import type { TestRun } from "@/types/testRun";

// Internal type to track test state for statistics calculation
// This mirrors the MongoDB processor's approach of counting actual tests by status
interface TestState {
  id: string;
  retryIndex: number;
  status: TestStatus;
}

// Helper to generate unique key for test (id + retryIndex)
// This matches MongoDB's approach where tests are identified by (id, retry_index) pair
const getTestKey = (testId: string, retryIndex: number): string => {
  return `${testId}-${retryIndex}`;
};

// Helper to recalculate statistics from test state map
// This mirrors the MongoDB processor logic in rest_mongodb.go (lines 269-311)
const calculateStatistics = (testStates: Map<string, TestState>) => {
  const stats = {
    total: testStates.size,
    passed: 0,
    failed: 0,
    skipped: 0,
    running: 0,
    broken: 0,
    timedout: 0,
    interrupted: 0,
    unknown: 0,
  };

  for (const testState of testStates.values()) {
    switch (testState.status) {
      case "PASSED":
        stats.passed++;
        break;
      case "FAILED":
        stats.failed++;
        break;
      case "SKIPPED":
        stats.skipped++;
        break;
      case "RUNNING":
        stats.running++;
        break;
      case "BROKEN":
        stats.broken++;
        break;
      case "TIMEDOUT":
        stats.timedout++;
        break;
      case "INTERRUPTED":
        stats.interrupted++;
        break;
      case "UNKNOWN":
        stats.unknown++;
        break;
      default:
        // Empty status is treated as RUNNING (matching MongoDB processor)
        if (!testState.status) {
          stats.running++;
        } else {
          stats.unknown++;
        }
    }
  }

  return stats;
};

export const handleUpdateRun = (
  data: WebSocketTestData,
  type: EventType,
  setRuns: React.Dispatch<React.SetStateAction<TestRun[]>>
) => {
  const testData = data as WebSocketTestData;

  // WebSocket sends TestDocument directly (not wrapped in testCase)
  // TestDocument has: id, runId, suiteId, status, retryIndex, etc.
  const runId =
    (data as any).runId ||
    testData.testCase?.runId ||
    testData.runId ||
    testData.testRunId;

  const testId = (data as any).id || testData.testCase?.id;
  const retryIndex = (data as any).retryIndex ?? 0; // Default to 0 if not provided

  // Safely extract status - TestDocument.status is already a string
  let status: TestStatus;
  const rawStatus = (data as any).status;

  if (typeof rawStatus === "string") {
    status = rawStatus.toUpperCase() as TestStatus;
  } else if (type === "test.begin") {
    status = "RUNNING";
  } else {
    status = "UNKNOWN";
  }

  if (runId && testId) {
    setRuns((prevRuns) => {
      try {
        const existingIndex = prevRuns.findIndex((r) => r.id === runId);

        if (existingIndex >= 0) {
          const updated = [...prevRuns];
          const currentRun = { ...updated[existingIndex] };

          // Initialize tests array if not present (used to track test states)
          if (!currentRun.tests) {
            currentRun.tests = [];
          }

          // Build test state map from current run's tests
          // This map represents the current state of all tests in this run
          const testStates = new Map<string, TestState>();
          for (const test of currentRun.tests) {
            const key = getTestKey(test.id, (test as any).retryIndex ?? 0);
            testStates.set(key, {
              id: test.id,
              retryIndex: (test as any).retryIndex ?? 0,
              status: test.status,
            });
          }

          // Update or add the test state based on the event
          const testKey = getTestKey(testId, retryIndex);
          testStates.set(testKey, {
            id: testId,
            retryIndex,
            status,
          });

          // Rebuild tests array from test states map
          currentRun.tests = Array.from(testStates.values()).map((state) => ({
            id: state.id,
            runId,
            title: "", // Not needed for statistics
            status: state.status,
            retryIndex: state.retryIndex,
          })) as any;

          // Recalculate statistics from test states (absolute counting, not incremental)
          // This mirrors the MongoDB processor's approach
          currentRun.statistics = calculateStatistics(testStates);
          currentRun.updatedAt = new Date().toISOString();
          updated[existingIndex] = currentRun;

          return updated;
        }

        return prevRuns;
      } catch (error) {
        console.error("[handleUpdateRun] Error updating run statistics:", error);
        return prevRuns;
      }
    });
  }
};
