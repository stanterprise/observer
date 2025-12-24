import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { apiUrl } from "../../lib/config";
import { Card, CardContent } from "../../components/Card";
import { Badge } from "../../components/Badge";
import type { WebSocketEvent, TestStatus, WebSocketTestData } from "../../types";
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
  running?: number;
  broken?: number;
  timedout?: number;
  interrupted?: number;
  unknown?: number;
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
      // Fetch all run statistics in a single request
      const response = await fetch(apiUrl("/runs/stats"));
      if (!response.ok) {
        throw new Error(`Failed to fetch runs: ${response.statusText}`);
      }
      const data = await response.json();
      const stats = (data.runs || []) as TestRunStats[];

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

  // Handle WebSocket events to update run statistics locally
  useEffect(() => {
    if (!onWebSocketEvent) return;

    const { type, data } = onWebSocketEvent;

    if (type === "test.begin" || type === "test.end") {
      const testData = data as WebSocketTestData;
      const runId = testData.run_id || testData.test_case?.run_id;

      // Safely extract status - handle both string and non-string values
      let status = "RUNNING";
      if (type === "test.end") {
        const rawStatus = testData.test_case?.status || testData.status;
        // Handle numeric status codes (protobuf enums) - need to map them
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

      if (runId) {
        setRuns((prevRuns) => {
          try {
            const existingIndex = prevRuns.findIndex((r) => r.runId === runId);

            if (existingIndex >= 0) {
              const updated = [...prevRuns];
              const currentRun = { ...updated[existingIndex] };

              // Update statistics based on event type
              if (type === "test.begin") {
                currentRun.running = (currentRun.running || 0) + 1;
                currentRun.total = currentRun.total + 1;
              } else if (type === "test.end") {
                // Decrement running count
                if (currentRun.running && currentRun.running > 0) {
                  currentRun.running--;
                }

                // Increment appropriate status counter
                switch (status) {
                  case "PASSED":
                    currentRun.passed++;
                    break;
                  case "FAILED":
                    currentRun.failed++;
                    break;
                  case "SKIPPED":
                    currentRun.skipped++;
                    break;
                  case "BROKEN":
                    currentRun.broken = (currentRun.broken || 0) + 1;
                    break;
                  case "TIMEDOUT":
                    currentRun.timedout = (currentRun.timedout || 0) + 1;
                    break;
                  case "INTERRUPTED":
                    currentRun.interrupted = (currentRun.interrupted || 0) + 1;
                    break;
                  default:
                    currentRun.unknown = (currentRun.unknown || 0) + 1;
                }
              }

              currentRun.lastUpdated = new Date().toISOString();
              updated[existingIndex] = currentRun;
              return updated;
            } else if (type === "test.begin") {
              // New run started
              const newRun: TestRunStats = {
                runId,
                total: 1,
                passed: 0,
                failed: 0,
                skipped: 0,
                running: 1,
                broken: 0,
                timedout: 0,
                interrupted: 0,
                unknown: 0,
                lastUpdated: new Date().toISOString(),
              };
              return [newRun, ...prevRuns];
            }

            return prevRuns;
          } catch (error) {
            console.error("Error updating runs from WebSocket:", error);
            return prevRuns;
          }
        });
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
    // Prioritize error states
    if (run.failed > 0) return "failed";
    if (run.broken && run.broken > 0) return "broken";
    if (run.timedout && run.timedout > 0) return "timedout";
    if (run.interrupted && run.interrupted > 0) return "interrupted";

    // Then check for success or skip (all tests completed)
    if (run.passed === run.total && run.total > 0) return "passed";
    if (run.skipped === run.total && run.total > 0) return "skipped";

    // Check for active running tests (tests in progress)
    if (run.running && run.running > 0) return "running";

    // Check for unknown status (actual UNKNOWN status from backend)
    if (run.unknown && run.unknown > 0) return "unknown";

    // If no tests or all tests are in a mixed state, default to running
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
                    <th
                      scope="col"
                      className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
                    >
                      Run ID
                    </th>
                    <th
                      scope="col"
                      className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
                    >
                      Status
                    </th>
                    <th
                      scope="col"
                      className="px-6 py-3 text-center text-xs font-medium text-gray-500 uppercase tracking-wider"
                    >
                      <div className="flex items-center justify-center">
                        <CheckCircle className="h-4 w-4 mr-1 text-green-600" />
                        Passed
                      </div>
                    </th>
                    <th
                      scope="col"
                      className="px-6 py-3 text-center text-xs font-medium text-gray-500 uppercase tracking-wider"
                    >
                      <div className="flex items-center justify-center">
                        <XCircle className="h-4 w-4 mr-1 text-red-600" />
                        Failed
                      </div>
                    </th>
                    <th
                      scope="col"
                      className="px-6 py-3 text-center text-xs font-medium text-gray-500 uppercase tracking-wider"
                    >
                      <div className="flex items-center justify-center">
                        <CircleDashed className="h-4 w-4 mr-1 text-gray-600" />
                        Skipped
                      </div>
                    </th>
                    <th
                      scope="col"
                      className="px-6 py-3 text-center text-xs font-medium text-gray-500 uppercase tracking-wider"
                    >
                      <div className="flex items-center justify-center">
                        <Play className="h-4 w-4 mr-1 text-blue-600" />
                        Total
                      </div>
                    </th>
                    <th
                      scope="col"
                      className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
                    >
                      <button
                        onClick={toggleSortOrder}
                        className="flex items-center hover:text-gray-700 transition-colors focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 rounded-md px-2 py-1 -mx-2 -my-1"
                        aria-label={`Sort by last updated, currently ${
                          sortOrder === "desc" ? "newest first" : "oldest first"
                        }`}
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
                          className="text-blue-600 hover:text-blue-800 font-medium hover:underline focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 rounded"
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
