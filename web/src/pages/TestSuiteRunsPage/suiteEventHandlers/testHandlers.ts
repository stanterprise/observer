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

// Extended type for WebSocket test data with additional fields
interface ExtendedTestData extends WebSocketTestData {
  id?: string;
  runId?: string;
  status?: string;
  retryIndex?: number;
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
  const extendedData = data as ExtendedTestData;

  // WebSocket sends TestDocument directly (not wrapped in testCase)
  // TestDocument has: id, runId, suiteId, status, retryIndex, etc.
  const runId =
    extendedData.runId ||
    extendedData.testCase?.runId ||
    extendedData.testRunId;

  const testId = extendedData.id || extendedData.testCase?.id;
  const retryIndex = extendedData.retryIndex ?? extendedData.testCase?.retryIndex ?? 0;

  // Safely extract status - TestDocument.status is already a string
  let status: TestStatus;
  const rawStatus = extendedData.status;

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
          
          // If tests array is empty but we have existing statistics from MongoDB,
          // initialize the map with placeholder tests based on MongoDB statistics.
          // This allows WebSocket events to update existing counts incrementally.
          if (currentRun.tests.length === 0 && currentRun.statistics) {
            // Create placeholder tests for each status category from MongoDB stats
            let placeholderId = 0;
            const stats = currentRun.statistics;
            
            // Helper to add placeholder tests for a given status
            const addPlaceholders = (status: TestStatus, count: number) => {
              for (let i = 0; i < count; i++) {
                const key = getTestKey(`__placeholder_${status}_${placeholderId}`, 0);
                testStates.set(key, {
                  id: `__placeholder_${status}_${placeholderId}`,
                  retryIndex: 0,
                  status: status,
                });
                placeholderId++;
              }
            };
            
            // Add placeholders for each status
            if (stats.passed) addPlaceholders("PASSED", stats.passed);
            if (stats.failed) addPlaceholders("FAILED", stats.failed);
            if (stats.skipped) addPlaceholders("SKIPPED", stats.skipped);
            if (stats.running) addPlaceholders("RUNNING", stats.running);
            if (stats.broken) addPlaceholders("BROKEN", stats.broken);
            if (stats.timedout) addPlaceholders("TIMEDOUT", stats.timedout);
            if (stats.interrupted) addPlaceholders("INTERRUPTED", stats.interrupted);
            if (stats.unknown) addPlaceholders("UNKNOWN", stats.unknown);
          } else {
            // Load existing test states from tests array
            for (const test of currentRun.tests) {
              const key = getTestKey(test.id, test.retryIndex ?? 0);
              testStates.set(key, {
                id: test.id,
                retryIndex: test.retryIndex ?? 0,
                status: test.status,
              });
            }
          }

          // Update or add the test state based on the event
          const testKey = getTestKey(testId, retryIndex);
          
          // If this is a real test (not a placeholder), check if we need to remove a placeholder
          // This happens when WebSocket events arrive after page refresh with MongoDB placeholders
          const oldTestState = testStates.get(testKey);
          if (!oldTestState) {
            // New test - check if we should replace a placeholder of the old status
            // Find and remove one placeholder of the previous status (if transitioning states)
            if (type === "test.end") {
              // Test is ending - it was previously RUNNING, find and remove a RUNNING placeholder
              for (const [key] of testStates.entries()) {
                if (key.startsWith("__placeholder_RUNNING_")) {
                  testStates.delete(key);
                  break;
                }
              }
            }
          } else if (oldTestState && type === "test.end") {
            // Test is transitioning from oldStatus to newStatus
            // Remove a placeholder of the old status if it exists
            const oldStatus = oldTestState.status;
            if (oldStatus !== status) {
              for (const [key] of testStates.entries()) {
                if (key.startsWith(`__placeholder_${oldStatus}_`)) {
                  testStates.delete(key);
                  break;
                }
              }
            }
          }
          
          // Add or update the real test
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
          }));

          // Recalculate statistics from test states (absolute counting, not incremental)
          // This mirrors the MongoDB processor's approach
          // With placeholder initialization, we always have complete state
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
