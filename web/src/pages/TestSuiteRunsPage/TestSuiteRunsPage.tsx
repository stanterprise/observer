import { useEffect, useState, useCallback } from "react";
import { Link } from "react-router-dom";
import { apiUrl } from "@/lib/config";
import { Card, CardContent } from "@/components/Card";
import { Badge } from "@/components/Badge";
import { useWebSocket } from "@/hooks/useWebSocket";
import type {
  WebSocketEvent,
  WebSocketRunData,
  WebSocketTestData,
} from "@/types/webSocket";

import {
  Play,
  CheckCircle,
  XCircle,
  CircleDashed,
  Clock,
  ArrowUpDown,
} from "lucide-react";

import type { TestRun } from "@/types/testRun";
import { handleStartRun, handleUpdateRun } from "./suiteEventHandlers";
import { getRunStatus } from "./utils";

export function TestSuiteRunsPage() {
  const [runs, setRuns] = useState<TestRun[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [sortOrder, setSortOrder] = useState<"asc" | "desc">("desc");

  // Handle WebSocket events with filtered subscription
  const handleWebSocketMessage = useCallback((event: WebSocketEvent) => {
    const { type, data } = event;
    if (type == "run.start") {
      handleStartRun(data as WebSocketRunData, runs, setRuns);
    }
    if (type == "run.end") {
    }

    if (type === "test.begin" || type === "test.end") {
      handleUpdateRun(data as WebSocketTestData, type, runs, setRuns);
    }
  }, []);

  // Subscribe to test.begin and test.end events only
  useWebSocket({
    onMessage: handleWebSocketMessage,
  });

  useEffect(() => {
    fetchRuns();
  }, []);

  const fetchRuns = async () => {
    try {
      setLoading(true);
      // Fetch all run statistics in a single request
      const response = await fetch(apiUrl("/runs"));
      if (!response.ok) {
        throw new Error(`Failed to fetch runs: ${response.statusText}`);
      }
      const data = await response.json();
      const stats = (data.runs || []) as TestRun[];

      // Sort by lastUpdated (most recent first by default)
      stats.sort((a, b) => {
        const aTime = a.updatedAt ? new Date(a.updatedAt).getTime() : 0;
        const bTime = b.updatedAt ? new Date(b.updatedAt).getTime() : 0;
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

  const toggleSortOrder = () => {
    setSortOrder((prev) => (prev === "desc" ? "asc" : "desc"));
  };

  // Sort runs based on current sort order
  const sortedRuns = [...runs].sort((a, b) => {
    const aTime = a.updatedAt ? new Date(a.updatedAt).getTime() : 0;
    const bTime = b.updatedAt ? new Date(b.updatedAt).getTime() : 0;
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
                      Run Name
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
                      key={run.id}
                      className="hover:bg-gray-50 transition-colors"
                    >
                      <td className="px-6 py-4 whitespace-nowrap">
                        <Link
                          to={`/suite_runs/${run.id}`}
                          className="text-blue-600 hover:text-blue-800 font-medium hover:underline focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 rounded"
                        >
                          {run.name || run.id}
                        </Link>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <Badge status={getRunStatus(run)} />
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-center">
                        <span className="text-green-600 font-semibold">
                          {run.statistics!.passed}
                        </span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-center">
                        <span className="text-red-600 font-semibold">
                          {run.statistics!.failed}
                        </span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-center">
                        <span className="text-gray-600 font-semibold">
                          {run.statistics!.skipped}
                        </span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-center">
                        <span className="text-blue-600 font-semibold">
                          {run.statistics!.total}
                        </span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                        {run.updatedAt ? (
                          <div className="flex flex-col">
                            <span>
                              {new Date(run.updatedAt).toLocaleDateString()}
                            </span>
                            <span className="text-xs text-gray-400">
                              {new Date(run.updatedAt).toLocaleTimeString()}
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
