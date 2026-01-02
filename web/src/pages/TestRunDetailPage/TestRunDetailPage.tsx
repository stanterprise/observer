import { useEffect, useState } from "react";
import { useParams, Link } from "react-router-dom";
import { apiUrl } from "@/lib/config";
import { Card, CardContent } from "@/components/Card";
import type { WebSocketEvent, WebSocketTestData } from "@/types";
import { ArrowLeft, Play } from "lucide-react";
import type { RunDetail } from "./types";
import TestCaseRecord from "./TestCaseRecord";
import { SuiteTitleCard } from "./SuiteTitleCard";
import type { TestStatus } from "@/types/common";
import { assembleSuiteHierarchy } from "../TestSuiteRunsPage/utils";
import type { TestSuite } from "@/types/testSuite";

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

  const countTests = (suites: TestSuite[]): number => {
    let total = 0;
    for (const suite of suites) {
      total += suite.tests?.length || 0; // API returns 'tests' (lowercase)
      total += countTests(suite.suites ?? []);
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

      data.statistics = {
        total: data.tests.length,
        passed: data.tests.filter((t: any) => t.status === "PASSED").length,
        failed: data.tests.filter((t: any) => t.status === "FAILED").length,
        skipped: data.tests.filter((t: any) => t.status === "SKIPPED").length,
        running: data.tests.filter((t: any) => t.status === "RUNNING").length,
        broken: data.tests.filter((t: any) => t.status === "BROKEN").length,
        timedout: data.tests.filter((t: any) => t.status === "TIMEDOUT").length,
        interrupted: data.tests.filter((t: any) => t.status === "INTERRUPTED")
          .length,
        unknown: data.tests.filter((t: any) => t.status === "UNKNOWN").length,
      };
      console.log("Fetched run details:", data);
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
      const testRunId = testData.runId || testData.testCase?.runId;

      if (testRunId === runId) {
        console.log("WebSocket event received for run:", testData);
        setRunDetail((prevDetail) => {
          if (!prevDetail || !prevDetail.tests) return prevDetail;

          try {
            const testId = testData.testCase?.id || testData.id;
            // Safely extract status - handle both string and numeric values (protobuf enums)
            const rawStatus = testData.testCase?.status || testData.status;
            let status = "running";
            if (type === "test.end") {
              if (typeof rawStatus === "number") {
                // Protobuf enum mapping: 0=UNKNOWN, 1=PASSED, 2=FAILED, 3=SKIPPED, etc.
                const statusMap: Record<number, string> = {
                  0: "unknown",
                  1: "passed",
                  2: "failed",
                  3: "skipped",
                  4: "broken",
                  5: "timedout",
                  6: "interrupted",
                };
                status = statusMap[rawStatus] || "unknown";
              } else if (typeof rawStatus === "string") {
                status = rawStatus.toLowerCase();
              }
            }

            // Update or add test in the tests array
            const updatedTests = [...prevDetail.tests];
            const testIndex = updatedTests.findIndex((t) => t.id === testId);

            if (type === "test.begin") {
              if (testIndex === -1) {
                // Add new test
                updatedTests.push({
                  id: testId || "",
                  runId: testRunId || "",
                  title: testData.testCase?.title || "",
                  status: "running",
                  createdAt: new Date().toISOString(),
                  updatedAt: new Date().toISOString(),
                });
              } else {
                // Update existing test
                updatedTests[testIndex] = {
                  ...updatedTests[testIndex],
                  status: "running",
                  updatedAt: new Date().toISOString(),
                };
              }
            } else if (type === "test.end") {
              if (testIndex >= 0) {
                updatedTests[testIndex] = {
                  ...updatedTests[testIndex],
                  status: status as TestStatus,
                  updatedAt: new Date().toISOString(),
                };
              }
            }

            // Recalculate statistics
            const newStats = runDetail.statistics
              ? runDetail.statistics
              : {
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
              switch (test.status) {
                case "passed":
                  newStats.passed++;
                  break;
                case "failed":
                  newStats.failed++;
                  break;
                case "skipped":
                  newStats.skipped++;
                  break;
                case "running":
                  newStats.running
                    ? newStats.running++
                    : (newStats.running = 1);
                  break;
                case "broken":
                  newStats.broken ? newStats.broken++ : (newStats.broken = 1);
                  break;
                case "timedout":
                  newStats.timedout
                    ? newStats.timedout++
                    : (newStats.timedout = 1);
                  break;
                case "interrupted":
                  newStats.interrupted
                    ? newStats.interrupted++
                    : (newStats.interrupted = 1);
                  break;
                default:
                  newStats.unknown
                    ? newStats.unknown++
                    : (newStats.unknown = 1);
              }
            });

            console.log("Updated run detail from WebSocket:", {
              tests: updatedTests,
              statistics: newStats,
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

  console.log(
    "Rendering run detail:",
    runDetail,
    "Overall status:",
    overallStatus
  );

  console.log(
    "Assembled suite hierarchy:",
    assembleSuiteHierarchy(runDetail.suites || [])
  );

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
      <SuiteTitleCard runDetail={runDetail} overallStatus={overallStatus} />

      {/* Test Cases List */}
      <div>
        <h2 className="text-xl font-semibold text-gray-900 mb-4">
          Test Cases ({countTests(runDetail.suites || [])})
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
              <TestCaseRecord key={test.id} test={test} runId={runDetail.id} />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
