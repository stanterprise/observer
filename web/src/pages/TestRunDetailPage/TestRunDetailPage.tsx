import { useEffect, useState } from "react";
import { useParams, Link } from "react-router-dom";
import { apiUrl } from "@/lib/config";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/Card";
import { Badge } from "@/components/Badge";
import type { WebSocketEvent, TestStatus, WebSocketTestData } from "@/types";
import {
  ArrowLeft,
  CheckCircle,
  XCircle,
  CircleDashed,
  Play,
  CircleOff,
} from "lucide-react";
import type { RunDetail } from "./types";
import TestCaseRecord from "./TestCaseRecord";

interface TestRunDetailPageProps {
  onWebSocketEvent?: WebSocketEvent | null;
}

export function TestRunDetailPage({
  onWebSocketEvent,
}: TestRunDetailPageProps) {
  const { runId } = useParams<{ runId: string }>();
  const [runDetail, setRunDetail] = useState<RunDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const countTests = (suite: RunDetail): number => {
    let total = suite.tests?.length || 0; // API returns 'tests' (lowercase)
    for (const childSuite of suite.suites ?? []) {
      total += countTests(childSuite);
    }

    return total;
  };

  useEffect(() => {
    if (runId) {
      fetchRunDetail(runId);
    }
  }, [runId]);

  const fetchRunDetail = async (id: string) => {
    try {
      setLoading(true);
      const response = await fetch(apiUrl(`/runs/${id}`));
      if (!response.ok) {
        throw new Error(`Failed to fetch run details: ${response.statusText}`);
      }
      const data = await response.json();
      setRunDetail(data);
      setError(null);
    } catch (err) {
      console.error("Error fetching run details:", err);
      setError(
        err instanceof Error ? err.message : "Failed to fetch run details"
      );
    } finally {
      setLoading(false);
    }
  };

  // Handle WebSocket events to update test statuses locally
  useEffect(() => {
    if (!onWebSocketEvent || !runDetail) return;

    const { type, data } = onWebSocketEvent;
    if (type === "test.end" || type === "test.begin") {
      const testData = data as WebSocketTestData;
      const testRunId = testData.run_id || testData.test_case?.run_id;

      if (testRunId === runId) {
        setRunDetail((prevDetail) => {
          if (!prevDetail || !prevDetail.tests) return prevDetail;

          try {
            const testId = testData.test_case?.id || testData.id;
            // Safely extract status - handle both string and numeric values (protobuf enums)
            const rawStatus = testData.test_case?.status || testData.status;
            let status = "RUNNING";
            if (type === "test.end") {
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
                status = statusMap[rawStatus] || "UNKNOWN";
              } else if (typeof rawStatus === "string") {
                status = rawStatus.toUpperCase();
              }
            }

            // Update or add test in the tests array
            const updatedTests = [...prevDetail.tests];
            const testIndex = updatedTests.findIndex((t) => t.ID === testId);

            if (type === "test.begin") {
              if (testIndex === -1) {
                // Add new test
                updatedTests.push({
                  ID: testId || "",
                  RunID: testRunId || "",
                  Title: testData.test_case?.title || "",
                  Status: "RUNNING",
                  CreatedAt: new Date().toISOString(),
                  UpdatedAt: new Date().toISOString(),
                });
              } else {
                // Update existing test
                updatedTests[testIndex] = {
                  ...updatedTests[testIndex],
                  Status: "RUNNING",
                  UpdatedAt: new Date().toISOString(),
                };
              }
            } else if (type === "test.end") {
              if (testIndex >= 0) {
                updatedTests[testIndex] = {
                  ...updatedTests[testIndex],
                  Status: status,
                  UpdatedAt: new Date().toISOString(),
                };
              }
            }

            // Recalculate statistics
            const newStats = {
              total: updatedTests.length,
              passed: 0,
              failed: 0,
              skipped: 0,
              running: 0,
              broken: 0,
              timedout: 0,
              interrupted: 0,
              unknown: 0,
            };

            updatedTests.forEach((test) => {
              switch (test.Status) {
                case "PASSED":
                  newStats.passed++;
                  break;
                case "FAILED":
                  newStats.failed++;
                  break;
                case "SKIPPED":
                  newStats.skipped++;
                  break;
                case "RUNNING":
                  newStats.running++;
                  break;
                case "BROKEN":
                  newStats.broken++;
                  break;
                case "TIMEDOUT":
                  newStats.timedout++;
                  break;
                case "INTERRUPTED":
                  newStats.interrupted++;
                  break;
                default:
                  newStats.unknown++;
              }
            });

            return {
              ...prevDetail,
              tests: updatedTests,
              statistics: newStats,
            };
          } catch (error) {
            console.error("Error updating run detail from WebSocket:", error);
            return prevDetail;
          }
        });
      }
    }
  }, [onWebSocketEvent, runId]);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-gray-600">Loading run details...</div>
      </div>
    );
  }

  if (error || !runDetail) {
    return (
      <div className="space-y-4">
        <Link
          to="/suite_runs"
          className="inline-flex items-center text-blue-600 hover:text-blue-700"
        >
          <ArrowLeft className="h-4 w-4 mr-2" />
          Back to Test Runs
        </Link>
        <div className="flex items-center justify-center h-64">
          <div className="text-red-600">Error: {error || "Run not found"}</div>
        </div>
      </div>
    );
  }

  const overallStatus: TestStatus =
    runDetail.statistics.failed > 0
      ? "failed"
      : runDetail.statistics.passed === runDetail.statistics.total &&
        runDetail.statistics.total > 0
      ? "passed"
      : runDetail.statistics.skipped === runDetail.statistics.total &&
        runDetail.statistics.total > 0
      ? "skipped"
      : "running";

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-4">
          <Link
            to="/suite_runs"
            className="inline-flex items-center text-blue-600 hover:text-blue-700"
          >
            <ArrowLeft className="h-5 w-5" />
          </Link>
          <h1 className="text-3xl font-bold text-gray-900">Test Suite Run</h1>
        </div>
      </div>

      {/* Run Summary Card */}
      <Card>
        {/* Progress Bar */}
        <div className="h-8 bg-gray-200 rounded-t-lg overflow-hidden flex">
          {runDetail.statistics.passed > 0 && (
            <div
              className="bg-green-500 transition-all duration-300"
              style={{
                width: `${
                  (runDetail.statistics.passed / runDetail.statistics.total) *
                  100
                }%`,
              }}
              title={`${runDetail.statistics.passed} passed`}
            />
          )}
          {runDetail.statistics.failed > 0 && (
            <div
              className="bg-red-500 transition-all duration-300"
              style={{
                width: `${
                  (runDetail.statistics.failed / runDetail.statistics.total) *
                  100
                }%`,
              }}
              title={`${runDetail.statistics.failed} failed`}
            />
          )}
          {runDetail.statistics.skipped > 0 && (
            <div
              className="bg-gray-400 transition-all duration-300"
              style={{
                width: `${
                  (runDetail.statistics.skipped / runDetail.statistics.total) *
                  100
                }%`,
              }}
              title={`${runDetail.statistics.skipped} skipped`}
            />
          )}
          {runDetail.statistics.total > 0 &&
            runDetail.statistics.passed +
              runDetail.statistics.failed +
              runDetail.statistics.skipped <
              runDetail.statistics.total && (
              <div
                className="bg-blue-300 transition-all duration-300 animate-pulse"
                style={{
                  width: `${
                    ((runDetail.statistics.total -
                      runDetail.statistics.passed -
                      runDetail.statistics.failed -
                      runDetail.statistics.skipped) /
                      runDetail.statistics.total) *
                    100
                  }%`,
                }}
                title={`${
                  runDetail.statistics.total -
                  runDetail.statistics.passed -
                  runDetail.statistics.failed -
                  runDetail.statistics.skipped
                } running/pending`}
              />
            )}
        </div>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="text-xl mb-2">
                {runDetail.name ?? runDetail.runId}
              </CardTitle>
              <div className="text-sm text-gray-500">
                Total Steps: {runDetail.totalSteps}
              </div>
            </div>
            <Badge status={overallStatus} className="text-lg px-4 py-2" />
          </div>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 md:grid-cols-5 gap-6">
            <div className="flex flex-col items-center p-4 bg-gray-50 rounded-lg">
              <Play className="h-8 w-8 text-blue-600 mb-2" />
              <div className="text-2xl font-bold text-gray-900">
                {runDetail.statistics.total}
              </div>
              <div className="text-sm text-gray-600">Total Tests</div>
            </div>
            <div className="flex flex-col items-center p-4 bg-green-50 rounded-lg">
              <CheckCircle className="h-8 w-8 text-green-600 mb-2" />
              <div className="text-2xl font-bold text-green-600">
                {runDetail.statistics.passed}
              </div>
              <div className="text-sm text-gray-600">Passed</div>
            </div>
            <div className="flex flex-col items-center p-4 bg-red-50 rounded-lg">
              <XCircle className="h-8 w-8 text-red-600 mb-2" />
              <div className="text-2xl font-bold text-red-600">
                {runDetail.statistics.failed}
              </div>
              <div className="text-sm text-gray-600">Failed</div>
            </div>
            <div className="flex flex-col items-center p-4 bg-gray-50 rounded-lg">
              <CircleDashed className="h-8 w-8 text-gray-600 mb-2" />
              <div className="text-2xl font-bold text-gray-600">
                {runDetail.statistics.skipped}
              </div>
              <div className="text-sm text-gray-600">Skipped</div>
            </div>
            <div className="flex flex-col items-center p-4 bg-gray-50 rounded-lg">
              <CircleOff className="h-8 w-8 text-gray-600 mb-2" />
              <div className="text-2xl font-bold text-gray-600">
                {runDetail.statistics.unknown}
              </div>
              <div className="text-sm text-gray-600">Unknown</div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Test Cases List */}
      <div>
        <h2 className="text-xl font-semibold text-gray-900 mb-4">
          Test Cases ({countTests(runDetail)})
        </h2>
        {!runDetail.tests || runDetail.tests.length === 0 ? (
          <Card>
            <CardContent>
              <div className="text-center py-8 text-gray-500">
                <Play className="mx-auto h-8 w-8 mb-2 text-gray-400" />
                <p>No test cases found in this run.</p>
              </div>
            </CardContent>
          </Card>
        ) : (
          <div className="space-y-3">
            {runDetail.tests.map((test) => (
              <TestCaseRecord
                key={test.ID}
                test={test}
                runId={runDetail.runId}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
