import { useEffect, useState } from "react";
import { useParams, Link } from "react-router-dom";
import { apiUrl } from "../lib/config";
import { Card, CardHeader, CardTitle, CardContent } from "./Card";
import { Badge } from "./Badge";
import type { WebSocketEvent, TestStatus, WebSocketTestData } from "../types";
import {
  ArrowLeft,
  CheckCircle,
  XCircle,
  CircleDashed,
  Play,
  Clock,
} from "lucide-react";

interface TestSuiteRunDetailPageProps {
  onWebSocketEvent?: WebSocketEvent | null;
}

interface TestCase {
  ID: string;
  RunID: string;
  Title: string;
  Status: string;
  Duration?: number;
  RetryCount?: number;
  CreatedAt: string;
  UpdatedAt: string;
}

interface RunStatistics {
  total: number;
  passed: number;
  failed: number;
  skipped: number;
}

interface RunDetail {
  runId: string;
  tests: TestCase[];
  statistics: RunStatistics;
  totalSteps: number;
}

export function TestSuiteRunDetailPage({
  onWebSocketEvent,
}: TestSuiteRunDetailPageProps) {
  const { runId } = useParams<{ runId: string }>();
  const [runDetail, setRunDetail] = useState<RunDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

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

  // Handle WebSocket events to update test statuses
  useEffect(() => {
    if (!onWebSocketEvent || !runDetail) return;

    const { type, data } = onWebSocketEvent;
    if (type === "test.end" || type === "test.begin") {
      const testData = data as WebSocketTestData;
      const testRunId = testData.run_id || testData.test_case?.run_id;

      if (testRunId === runId) {
        // Refetch the run details to get updated statistics
        if (runId) {
          fetchRunDetail(runId);
        }
      }
    }
  }, [onWebSocketEvent, runDetail, runId]);

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
          to="/runs"
          className="inline-flex items-center text-blue-600 hover:text-blue-700"
        >
          <ArrowLeft className="h-4 w-4 mr-2" />
          Back to Runs
        </Link>
        <div className="flex items-center justify-center h-64">
          <div className="text-red-600">Error: {error || "Run not found"}</div>
        </div>
      </div>
    );
  }

  const getTestStatus = (status: string): TestStatus => {
    const statusMap: Record<string, TestStatus> = {
      PASSED: "passed",
      FAILED: "failed",
      SKIPPED: "skipped",
      RUNNING: "running",
      UNKNOWN: "unknown",
      BROKEN: "broken",
      TIMEDOUT: "timedout",
      INTERRUPTED: "interrupted",
    };
    return (statusMap[status] || "unknown") as TestStatus;
  };

  const formatDuration = (nanoseconds?: number) => {
    if (!nanoseconds) return "N/A";
    const milliseconds = nanoseconds / 1000000;
    if (milliseconds < 1000) return `${milliseconds.toFixed(0)}ms`;
    return `${(milliseconds / 1000).toFixed(2)}s`;
  };

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
        <div className="h-2 bg-gray-200 rounded-t-lg overflow-hidden flex">
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
              <CardTitle className="text-xl mb-2">{runDetail.runId}</CardTitle>
              <div className="text-sm text-gray-500">
                Total Steps: {runDetail.totalSteps}
              </div>
            </div>
            <Badge status={overallStatus} className="text-lg px-4 py-2" />
          </div>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-6">
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
          </div>
        </CardContent>
      </Card>

      {/* Test Cases List */}
      <div>
        <h2 className="text-xl font-semibold text-gray-900 mb-4">
          Test Cases ({runDetail.tests.length})
        </h2>
        <div className="space-y-3">
          {runDetail.tests.map((test) => (
            <Link key={test.ID} to={`/tests/${test.ID}`}>
              <Card className="hover:shadow-md transition-shadow cursor-pointer">
                <CardContent className="py-4">
                  <div className="flex items-center justify-between">
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center space-x-3">
                        <Badge status={getTestStatus(test.Status)} />
                        <h3 className="text-base font-medium text-gray-900 truncate">
                          {test.Title || test.ID}
                        </h3>
                      </div>
                      <div className="mt-2 flex items-center space-x-4 text-sm text-gray-500">
                        <div className="flex items-center">
                          <Clock className="h-4 w-4 mr-1" />
                          Duration: {formatDuration(test.Duration)}
                        </div>
                        {test.RetryCount !== undefined &&
                          test.RetryCount > 0 && (
                            <div>Retries: {test.RetryCount}</div>
                          )}
                        <div>
                          Started: {new Date(test.CreatedAt).toLocaleString()}
                        </div>
                      </div>
                    </div>
                  </div>
                </CardContent>
              </Card>
            </Link>
          ))}
        </div>
      </div>
    </div>
  );
}
