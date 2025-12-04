import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { apiUrl } from "../lib/config";
import { Card, CardHeader, CardTitle, CardContent } from "./Card";
import { Badge } from "./Badge";
import type {
  WebSocketEvent,
  TestStatus,
  TestCaseResponse,
  WebSocketTestData,
} from "../types";
import {
  Play,
  CheckCircle,
  XCircle,
  CircleDashed,
  Clock,
  ArrowUpDown,
} from "lucide-react";

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
  const [sortOrder, setSortOrder] = useState<"asc" | "desc">("desc");

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

      // Sort by lastUpdated (most recent first by default)
      stats.sort((a, b) => {
        const aTime = a.lastUpdated ? new Date(a.lastUpdated).getTime() : 0;
        const bTime = b.lastUpdated ? new Date(b.lastUpdated).getTime() : 0;
        return bTime - aTime; // Descending order (newest first)
      });

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

  const toggleSortOrder = () => {
    setSortOrder((prev) => (prev === "desc" ? "asc" : "desc"));
  };

  // Sort runs based on current sort order
  const sortedRuns = [...runs].sort((a, b) => {
    const aTime = a.lastUpdated ? new Date(a.lastUpdated).getTime() : 0;
    const bTime = b.lastUpdated ? new Date(b.lastUpdated).getTime() : 0;
    return sortOrder === "desc" ? bTime - aTime : aTime - bTime;
  });

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
        <Card>
          <CardContent className="p-0">
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-gray-200">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Run ID
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Status
                    </th>
                    <th className="px-6 py-3 text-center text-xs font-medium text-gray-500 uppercase tracking-wider">
                      <div className="flex items-center justify-center">
                        <CheckCircle className="h-4 w-4 mr-1 text-green-600" />
                        Passed
                      </div>
                    </th>
                    <th className="px-6 py-3 text-center text-xs font-medium text-gray-500 uppercase tracking-wider">
                      <div className="flex items-center justify-center">
                        <XCircle className="h-4 w-4 mr-1 text-red-600" />
                        Failed
                      </div>
                    </th>
                    <th className="px-6 py-3 text-center text-xs font-medium text-gray-500 uppercase tracking-wider">
                      <div className="flex items-center justify-center">
                        <CircleDashed className="h-4 w-4 mr-1 text-gray-600" />
                        Skipped
                      </div>
                    </th>
                    <th className="px-6 py-3 text-center text-xs font-medium text-gray-500 uppercase tracking-wider">
                      <div className="flex items-center justify-center">
                        <Play className="h-4 w-4 mr-1 text-blue-600" />
                        Total
                      </div>
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      <button
                        onClick={toggleSortOrder}
                        className="flex items-center hover:text-gray-700 transition-colors"
                      >
                        <Clock className="h-4 w-4 mr-1" />
                        Last Updated
                        <ArrowUpDown className="h-3 w-3 ml-1" />
                      </button>
                    </th>
                  </tr>
                </thead>
                <tbody className="bg-white divide-y divide-gray-200">
                  {sortedRuns.map((run) => (
                    <tr
                      key={run.runId}
                      className="hover:bg-gray-50 transition-colors"
                    >
                      <td className="px-6 py-4 whitespace-nowrap">
                        <Link
                          to={`/suite_runs/${run.runId}`}
                          className="text-blue-600 hover:text-blue-800 font-medium"
                        >
                          {run.runId}
                        </Link>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <Badge status={getRunStatus(run)} />
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-center">
                        <span className="text-green-600 font-semibold">
                          {run.passed}
                        </span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-center">
                        <span className="text-red-600 font-semibold">
                          {run.failed}
                        </span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-center">
                        <span className="text-gray-600 font-semibold">
                          {run.skipped}
                        </span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-center">
                        <span className="text-blue-600 font-semibold">
                          {run.total}
                        </span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                        {run.lastUpdated ? (
                          <div className="flex flex-col">
                            <span>
                              {new Date(run.lastUpdated).toLocaleDateString()}
                            </span>
                            <span className="text-xs text-gray-400">
                              {new Date(run.lastUpdated).toLocaleTimeString()}
                            </span>
                          </div>
                        ) : (
                          <span className="text-gray-400">N/A</span>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
