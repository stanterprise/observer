import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { apiUrl } from "../lib/config";
import { Card, CardHeader, CardTitle, CardContent } from "./Card";
import { Badge } from "./Badge";
import type { WebSocketEvent, TestStatus, TestCaseResponse, WebSocketTestData } from "../types";
import { Play, CheckCircle, XCircle, CircleDashed, Clock } from "lucide-react";

interface TestRunsPageProps {
  onWebSocketEvent?: WebSocketEvent | null;
}

interface TestRunStats {
  runId: string;
  total: number;
  passed: number;
  failed: number;
  skipped: number;
  lastUpdated?: string;
}

export function TestSuiteRunsPage({ onWebSocketEvent }: TestRunsPageProps) {
  const [runs, setRuns] = useState<TestRunStats[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetchRuns();
  }, []);

  const fetchRuns = async () => {
    try {
      setLoading(true);
      const response = await fetch(apiUrl("/runs"));
      if (!response.ok) {
        throw new Error(`Failed to fetch runs: ${response.statusText}`);
      }
      const data = await response.json();
      const runIds = data.runs || [];

      // Fetch stats for each run
      const statsPromises = runIds.map(async (runId: string) => {
        const statsResponse = await fetch(apiUrl(`/runs/${runId}`));
        if (!statsResponse.ok) {
          return null;
        }
        const statsData = await statsResponse.json();
        return {
          runId: statsData.runId,
          total: statsData.statistics.total || 0,
          passed: statsData.statistics.passed || 0,
          failed: statsData.statistics.failed || 0,
          skipped: statsData.statistics.skipped || 0,
          lastUpdated:
            statsData.tests && statsData.tests.length > 0
              ? new Date(
                  Math.max(
                    ...statsData.tests.map((t: TestCaseResponse) =>
                      new Date(t.UpdatedAt).getTime()
                    )
                  )
                ).toISOString()
              : undefined,
        };
      });

      const stats = (await Promise.all(statsPromises)).filter(
        (s): s is TestRunStats => s !== null
      );
      setRuns(stats);
      setError(null);
    } catch (err) {
      console.error("Error fetching runs:", err);
      setError(err instanceof Error ? err.message : "Failed to fetch runs");
    } finally {
      setLoading(false);
    }
  };

  // Handle WebSocket events to update run statistics
  useEffect(() => {
    if (!onWebSocketEvent) return;

    const { type, data } = onWebSocketEvent;
    if (type === "test.begin" || type === "test.end") {
      const testData = data as WebSocketTestData;
      const runId = testData.run_id || testData.test_case?.run_id;

      if (runId) {
        // Refetch statistics for the affected run
        fetch(apiUrl(`/runs/${runId}`))
          .then((res) => res.json())
          .then((statsData) => {
            setRuns((prevRuns) => {
              const existingIndex = prevRuns.findIndex(
                (r) => r.runId === runId
              );
              const updatedRun: TestRunStats = {
                runId: statsData.runId,
                total: statsData.statistics.total || 0,
                passed: statsData.statistics.passed || 0,
                failed: statsData.statistics.failed || 0,
                skipped: statsData.statistics.skipped || 0,
                lastUpdated: new Date().toISOString(),
              };

              if (existingIndex >= 0) {
                const updated = [...prevRuns];
                updated[existingIndex] = updatedRun;
                return updated;
              } else {
                return [updatedRun, ...prevRuns];
              }
            });
          })
          .catch((err) => console.error("Failed to update run stats:", err));
      }
    }
  }, [onWebSocketEvent]);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-gray-600">Loading test runs...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-red-600">Error: {error}</div>
      </div>
    );
  }

  const getRunStatus = (run: TestRunStats): TestStatus => {
    if (run.failed > 0) return "failed";
    if (run.passed === run.total && run.total > 0) return "passed";
    if (run.skipped === run.total && run.total > 0) return "skipped";
    return "running";
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold text-gray-900">Test Suite Runs</h1>
        <button
          onClick={fetchRuns}
          className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors"
        >
          Refresh
        </button>
      </div>

      {runs.length === 0 ? (
        <Card>
          <CardContent>
            <div className="text-center py-12">
              <Play className="mx-auto h-12 w-12 text-gray-400" />
              <h3 className="mt-2 text-sm font-medium text-gray-900">
                No test runs found
              </h3>
              <p className="mt-1 text-sm text-gray-500">
                Test suite runs will appear here once tests are executed.
              </p>
            </div>
          </CardContent>
        </Card>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {runs.map((run) => (
            <Link key={run.runId} to={`/runs/${run.runId}`}>
              <Card className="hover:shadow-lg transition-shadow cursor-pointer h-full">
                <CardHeader>
                  <div className="flex items-center justify-between">
                    <CardTitle className="text-base truncate flex-1">
                      {run.runId}
                    </CardTitle>
                    <Badge status={getRunStatus(run)} />
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="space-y-3">
                    <div className="grid grid-cols-2 gap-3 text-sm">
                      <div className="flex items-center">
                        <CheckCircle className="h-4 w-4 mr-2 text-green-600" />
                        <span className="text-gray-600">Passed:</span>
                        <span className="ml-1 font-semibold text-green-600">
                          {run.passed}
                        </span>
                      </div>
                      <div className="flex items-center">
                        <XCircle className="h-4 w-4 mr-2 text-red-600" />
                        <span className="text-gray-600">Failed:</span>
                        <span className="ml-1 font-semibold text-red-600">
                          {run.failed}
                        </span>
                      </div>
                      <div className="flex items-center">
                        <CircleDashed className="h-4 w-4 mr-2 text-gray-600" />
                        <span className="text-gray-600">Skipped:</span>
                        <span className="ml-1 font-semibold text-gray-600">
                          {run.skipped}
                        </span>
                      </div>
                      <div className="flex items-center">
                        <Play className="h-4 w-4 mr-2 text-blue-600" />
                        <span className="text-gray-600">Total:</span>
                        <span className="ml-1 font-semibold text-blue-600">
                          {run.total}
                        </span>
                      </div>
                    </div>
                    {run.lastUpdated && (
                      <div className="flex items-center text-xs text-gray-500 pt-2 border-t border-gray-100">
                        <Clock className="h-3 w-3 mr-1" />
                        <span>
                          Last updated: {new Date(run.lastUpdated).toLocaleString()}
                        </span>
                      </div>
                    )}
                  </div>
                </CardContent>
              </Card>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}
