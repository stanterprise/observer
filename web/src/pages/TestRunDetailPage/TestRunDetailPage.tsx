import { useEffect, useState } from "react";
import { useParams, Link } from "react-router-dom";
import { apiUrl } from "@/lib/config";
import { Card, CardContent } from "@/components/Card";
import type { WebSocketEvent, WebSocketTestData } from "@/types/webSocket";
import { ArrowLeft, Play, Eye, EyeOff } from "lucide-react";
import { cn } from "@/lib/utils";

import { SuiteTitleCard } from "./SuiteTitleCard";
import type { TestStatus } from "@/types/common";
import { assembleSuiteHierarchy } from "../TestSuiteRunsPage/utils";
import type { TestSuite } from "@/types/testSuite";
import TestSuiteRecord from "./TestSuiteRecord";
import type { TestRun } from "@/types/testRun";

interface TestRunDetailPageProps {
  onWebSocketEvent?: WebSocketEvent | null;
}

export function TestRunDetailPage({
  onWebSocketEvent,
}: TestRunDetailPageProps) {
  const { runId } = useParams<{ runId: string }>();
  const [runDetail, setRunDetail] = useState<TestRun | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [hiddenSuiteTypes, setHiddenSuiteTypes] = useState<Set<string>>(
    new Set(["ROOT", "PROJECT", "FILE"])
  );

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
                  status: "RUNNING",
                  createdAt: new Date().toISOString(),
                  updatedAt: new Date().toISOString(),
                });
              } else {
                // Update existing test
                updatedTests[testIndex] = {
                  ...updatedTests[testIndex],
                  status: "RUNNING",
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
                  newStats.running
                    ? newStats.running++
                    : (newStats.running = 1);
                  break;
                case "BROKEN":
                  newStats.broken ? newStats.broken++ : (newStats.broken = 1);
                  break;
                case "TIMEDOUT":
                  newStats.timedout
                    ? newStats.timedout++
                    : (newStats.timedout = 1);
                  break;
                case "INTERRUPTED":
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
      <div className="space-y-6 animate-in fade-in duration-300">
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-4">
            <div className="h-10 w-10 bg-gray-200 rounded-lg animate-pulse" />
            <div className="h-8 w-48 bg-gray-200 rounded animate-pulse" />
          </div>
        </div>
        <div className="bg-white rounded-lg shadow-md border border-gray-200 overflow-hidden">
          <div className="h-8 bg-gray-200 animate-pulse" />
          <div className="p-6 space-y-4">
            <div className="h-6 bg-gray-200 rounded w-3/4 animate-pulse" />
            <div className="grid grid-cols-2 md:grid-cols-4 gap-6">
              {[1, 2, 3, 4].map((i) => (
                <div
                  key={i}
                  className="h-32 bg-gray-100 rounded-lg animate-pulse"
                />
              ))}
            </div>
          </div>
        </div>
        <div className="space-y-3">
          {[1, 2, 3].map((i) => (
            <div
              key={i}
              className="h-24 bg-gray-100 rounded-lg animate-pulse"
            />
          ))}
        </div>
      </div>
    );
  }

  if (error || !runDetail) {
    return (
      <div className="space-y-6 animate-in fade-in duration-300">
        <Link
          to="/suite_runs"
          className="inline-flex items-center gap-2 text-blue-600 hover:text-blue-700 transition-colors group"
        >
          <ArrowLeft className="h-5 w-5 group-hover:-translate-x-1 transition-transform" />
          <span className="font-medium">Back to Test Runs</span>
        </Link>
        <Card className="border-red-200 bg-red-50/50">
          <CardContent className="py-12">
            <div className="text-center max-w-md mx-auto">
              <div className="mx-auto h-16 w-16 rounded-full bg-red-100 flex items-center justify-center mb-4">
                <svg
                  className="h-8 w-8 text-red-600"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
                  />
                </svg>
              </div>
              <h3 className="text-lg font-semibold text-gray-900 mb-2">
                {error ? "Failed to Load Test Run" : "Test Run Not Found"}
              </h3>
              <p className="text-sm text-gray-600 mb-6">
                {error ||
                  "The test run you're looking for doesn't exist or has been deleted."}
              </p>
              <Link
                to="/suite_runs"
                className="inline-flex items-center px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
              >
                View All Test Runs
              </Link>
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  const overallStatus: TestStatus =
    runDetail.statistics!.running && runDetail.statistics!.running > 0
      ? "RUNNING"
      : runDetail.statistics!.failed > 0
      ? "FAILED"
      : runDetail.statistics!.passed === runDetail.statistics!.total &&
        runDetail.statistics!.total > 0
      ? "PASSED"
      : runDetail.statistics!.skipped === runDetail.statistics!.total &&
        runDetail.statistics!.total > 0
      ? "SKIPPED"
      : "NOT_RUN";
  console.log(
    "Rendering run detail:",
    runDetail,
    "Overall status:",
    overallStatus
  );

  const rootSuite = assembleSuiteHierarchy(
    runDetail.suites || [],
    runDetail.tests!
  );
  console.log("Assembled suite hierarchy:", rootSuite);

  // Get unique suite types from the hierarchy
  const getSuiteTypes = (suite: TestSuite): Set<string> => {
    const types = new Set<string>();
    if (suite.type) types.add(suite.type.toUpperCase());
    suite.suites?.forEach((s) => {
      getSuiteTypes(s).forEach((t) => types.add(t));
    });
    return types;
  };

  const availableSuiteTypes = Array.from(getSuiteTypes(rootSuite)).sort();

  const toggleSuiteType = (type: string) => {
    setHiddenSuiteTypes((prev) => {
      const newSet = new Set(prev);
      if (newSet.has(type)) {
        newSet.delete(type);
      } else {
        newSet.add(type);
      }
      return newSet;
    });
  };

  return (
    <div className="space-y-6 pb-8 animate-in fade-in duration-300">
      {/* Header with improved visual design */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Link
            to="/suite_runs"
            className="inline-flex items-center justify-center h-10 w-10 rounded-lg bg-white border border-gray-200 text-gray-700 hover:bg-gray-50 hover:border-gray-300 transition-all shadow-sm hover:shadow group"
            aria-label="Back to test runs"
          >
            <ArrowLeft className="h-5 w-5 group-hover:-translate-x-0.5 transition-transform" />
          </Link>
          <div>
            <h1 className="text-2xl md:text-3xl font-bold text-gray-900 tracking-tight">
              Test Suite Run
            </h1>
            <p className="text-sm text-gray-500 mt-1">
              {runDetail.name || runDetail.id}
            </p>
          </div>
        </div>
      </div>

      {/* Run Summary Card with improved spacing */}
      <div className="transition-all duration-300">
        <SuiteTitleCard runDetail={runDetail} overallStatus={overallStatus} />
      </div>

      {/* Test Cases List with enhanced design */}
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-xl font-semibold text-gray-900">Test Cases</h2>
          {availableSuiteTypes.length > 0 && (
            <div className="flex items-center gap-2">
              <span className="text-sm text-gray-600 font-medium">Suites:</span>
              <div className="flex gap-2">
                {availableSuiteTypes.map((type) => {
                  const isHidden = hiddenSuiteTypes.has(type);
                  return (
                    <button
                      key={type}
                      onClick={() => toggleSuiteType(type)}
                      className={cn(
                        "inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm font-medium transition-all border",
                        isHidden
                          ? "bg-gray-100 text-gray-500 border-gray-200 hover:bg-gray-200"
                          : "bg-blue-50 text-blue-700 border-blue-200 hover:bg-blue-100"
                      )}
                      aria-label={`${
                        isHidden ? "Show" : "Hide"
                      } ${type} suites`}
                    >
                      {isHidden ? (
                        <EyeOff className="h-4 w-4" />
                      ) : (
                        <Eye className="h-4 w-4" />
                      )}
                      {type}
                    </button>
                  );
                })}
              </div>
            </div>
          )}
        </div>

        {!runDetail.tests || runDetail.tests.length === 0 ? (
          <Card className="border-dashed">
            <CardContent className="py-16">
              <div className="text-center max-w-sm mx-auto">
                <div className="mx-auto h-16 w-16 rounded-full bg-gray-100 flex items-center justify-center mb-4">
                  <Play className="h-8 w-8 text-gray-400" />
                </div>
                <h3 className="text-base font-semibold text-gray-900 mb-2">
                  No Test Cases Yet
                </h3>
                <p className="text-sm text-gray-500">
                  This test run doesn't have any test cases yet. They will
                  appear here as tests are executed.
                </p>
              </div>
            </CardContent>
          </Card>
        ) : (
          <div className="transition-all duration-300">
            <TestSuiteRecord
              suite={rootSuite}
              hiddenSuiteTypes={hiddenSuiteTypes}
            />
          </div>
        )}
      </div>
    </div>
  );
}
