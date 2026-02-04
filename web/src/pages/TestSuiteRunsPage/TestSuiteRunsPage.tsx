import { useEffect, useState, useCallback } from "react";
import { Link } from "react-router-dom";
import { apiUrl } from "@/lib/config";
import { Card, CardContent } from "@/components/Card";
import { Badge } from "@/components/Badge";

import {
  Play,
  CheckCircle,
  XCircle,
  CircleDashed,
  Clock,
  ArrowUpDown,
  Trash2,
  AlertCircle,
  Tag,
  X,
} from "lucide-react";

import type { TestRun } from "@/types/testRun";
import { getRunStatus } from "./utils";

export function TestSuiteRunsPage() {
  const pollIntervalMs = 10_000;
  const [runs, setRuns] = useState<TestRun[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [sortOrder, setSortOrder] = useState<"asc" | "desc">("desc");
  const [selectedRuns, setSelectedRuns] = useState<Set<string>>(new Set());
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [showMarkerDialog, setShowMarkerDialog] = useState(false);
  const [markerValue, setMarkerValue] = useState("");
  const [updatingMarker, setUpdatingMarker] = useState(false);

  const fetchRuns = useCallback(async (options?: { silent?: boolean }) => {
    const silent = options?.silent ?? false;
    try {
      if (!silent) {
        setLoading(true);
      }
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
      if (!silent) {
        setLoading(false);
      }
    }
  }, []);

  useEffect(() => {
    fetchRuns();
  }, [fetchRuns]);

  useEffect(() => {
    const intervalId = window.setInterval(() => {
      fetchRuns({ silent: true });
    }, pollIntervalMs);

    return () => {
      window.clearInterval(intervalId);
    };
  }, [fetchRuns, pollIntervalMs]);

  const toggleRunSelection = (runId: string) => {
    setSelectedRuns((prev) => {
      const newSet = new Set(prev);
      if (newSet.has(runId)) {
        newSet.delete(runId);
      } else {
        newSet.add(runId);
      }
      return newSet;
    });
  };

  const toggleSelectAll = () => {
    if (selectedRuns.size === runs.length) {
      setSelectedRuns(new Set());
    } else {
      setSelectedRuns(new Set(runs.map((run) => run.id)));
    }
  };

  const handleDeleteSelected = async () => {
    if (selectedRuns.size === 0) return;

    setDeleting(true);
    try {
      const response = await fetch(apiUrl("/runs/delete"), {
        method: "DELETE",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          runIds: Array.from(selectedRuns),
        }),
      });

      if (!response.ok) {
        throw new Error(`Failed to delete runs: ${response.statusText}`);
      }

      const data = await response.json();
      console.log(`Deleted ${data.deleted} of ${data.requested} runs`);

      // Remove deleted runs from the list
      setRuns((prev) => prev.filter((run) => !selectedRuns.has(run.id)));
      setSelectedRuns(new Set());
      setShowDeleteConfirm(false);
      setError(null);
    } catch (err) {
      console.error("Error deleting runs:", err);
      setError(err instanceof Error ? err.message : "Failed to delete runs");
    } finally {
      setDeleting(false);
    }
  };

  const handleUpdateMarker = async (marker: string | null) => {
    if (selectedRuns.size === 0) return;

    setUpdatingMarker(true);
    try {
      const response = await fetch(apiUrl("/runs/marker"), {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          runIds: Array.from(selectedRuns),
          marker: marker,
        }),
      });

      if (!response.ok) {
        throw new Error(`Failed to update marker: ${response.statusText}`);
      }

      const data = await response.json();
      console.log(
        `Updated marker for ${data.modified} of ${data.requested} runs`,
      );

      // Update runs in the list with new marker
      setRuns((prev) =>
        prev.map((run) => {
          if (selectedRuns.has(run.id)) {
            return {
              ...run,
              metadata: {
                ...run.metadata,
                MARKER: marker || undefined,
              },
            };
          }
          return run;
        }),
      );

      setSelectedRuns(new Set());
      setShowMarkerDialog(false);
      setMarkerValue("");
      setError(null);
    } catch (err) {
      console.error("Error updating marker:", err);
      setError(err instanceof Error ? err.message : "Failed to update marker");
    } finally {
      setUpdatingMarker(false);
    }
  };

  const handleSetMarker = () => {
    setShowMarkerDialog(true);
    // Pre-fill with existing marker if all selected runs have the same marker
    const selectedRunsList = runs.filter((run) => selectedRuns.has(run.id));
    if (selectedRunsList.length > 0) {
      const firstMarker = selectedRunsList[0].metadata?.MARKER as
        | string
        | undefined;
      const allSame = selectedRunsList.every(
        (run) => run.metadata?.MARKER === firstMarker,
      );
      if (allSame && firstMarker) {
        setMarkerValue(firstMarker);
      } else {
        setMarkerValue("");
      }
    }
  };

  const handleRemoveMarker = async () => {
    await handleUpdateMarker(null);
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
        <div className="flex gap-2">
          {selectedRuns.size > 0 && (
            <>
              <button
                onClick={handleSetMarker}
                className="px-4 py-2 bg-indigo-600 text-white rounded-md hover:bg-indigo-700 transition-colors flex items-center gap-2"
                disabled={updatingMarker}
              >
                <Tag className="h-4 w-4" />
                Set Marker
              </button>
              <button
                onClick={handleRemoveMarker}
                className="px-4 py-2 bg-gray-600 text-white rounded-md hover:bg-gray-700 transition-colors flex items-center gap-2"
                disabled={updatingMarker}
              >
                <X className="h-4 w-4" />
                Remove Marker
              </button>
              <button
                onClick={() => setShowDeleteConfirm(true)}
                className="px-4 py-2 bg-red-600 text-white rounded-md hover:bg-red-700 transition-colors flex items-center gap-2"
                disabled={deleting}
              >
                <Trash2 className="h-4 w-4" />
                Delete ({selectedRuns.size})
              </button>
            </>
          )}
          <button
            onClick={() => fetchRuns()}
            className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors"
          >
            Refresh
          </button>
        </div>
      </div>

      {error && (
        <Card className="border-red-200 bg-red-50">
          <CardContent className="py-4">
            <div className="flex items-center gap-2 text-red-800">
              <AlertCircle className="h-5 w-5" />
              <span>{error}</span>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Delete Confirmation Dialog */}
      {showDeleteConfirm && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <Card className="max-w-md w-full mx-4">
            <CardContent className="p-6">
              <div className="flex items-start gap-4">
                <div className="flex-shrink-0">
                  <AlertCircle className="h-6 w-6 text-red-600" />
                </div>
                <div className="flex-1">
                  <h3 className="text-lg font-semibold text-gray-900 mb-2">
                    Delete Test Runs
                  </h3>
                  <p className="text-gray-600 mb-4">
                    Are you sure you want to delete {selectedRuns.size} test run
                    {selectedRuns.size !== 1 ? "s" : ""}? This action cannot be
                    undone.
                  </p>
                  <div className="flex gap-3 justify-end">
                    <button
                      onClick={() => setShowDeleteConfirm(false)}
                      className="px-4 py-2 border border-gray-300 rounded-md hover:bg-gray-50 transition-colors"
                      disabled={deleting}
                    >
                      Cancel
                    </button>
                    <button
                      onClick={handleDeleteSelected}
                      className="px-4 py-2 bg-red-600 text-white rounded-md hover:bg-red-700 transition-colors flex items-center gap-2"
                      disabled={deleting}
                    >
                      {deleting ? (
                        <>
                          <div className="animate-spin h-4 w-4 border-2 border-white border-t-transparent rounded-full" />
                          Deleting...
                        </>
                      ) : (
                        <>
                          <Trash2 className="h-4 w-4" />
                          Delete
                        </>
                      )}
                    </button>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      )}

      {/* Marker Dialog */}
      {showMarkerDialog && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <Card className="max-w-md w-full mx-4">
            <CardContent className="p-6">
              <div className="space-y-4">
                <div className="flex items-start gap-4">
                  <div className="flex-shrink-0">
                    <Tag className="h-6 w-6 text-indigo-600" />
                  </div>
                  <div className="flex-1">
                    <h3 className="text-lg font-semibold text-gray-900 mb-2">
                      Set Marker for Test Runs
                    </h3>
                    <p className="text-gray-600 mb-4">
                      Set a marker for {selectedRuns.size} test run
                      {selectedRuns.size !== 1 ? "s" : ""}. Markers help
                      organize and filter runs.
                    </p>
                    <div className="mb-4">
                      <label
                        htmlFor="marker-input"
                        className="block text-sm font-medium text-gray-700 mb-2"
                      >
                        Marker Value
                      </label>
                      <input
                        id="marker-input"
                        type="text"
                        value={markerValue}
                        onChange={(e) => setMarkerValue(e.target.value)}
                        placeholder="e.g., release-1.0, nightly, staging"
                        className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500"
                        disabled={updatingMarker}
                      />
                    </div>
                    <div className="flex gap-3 justify-end">
                      <button
                        onClick={() => {
                          setShowMarkerDialog(false);
                          setMarkerValue("");
                        }}
                        className="px-4 py-2 border border-gray-300 rounded-md hover:bg-gray-50 transition-colors"
                        disabled={updatingMarker}
                      >
                        Cancel
                      </button>
                      <button
                        onClick={() => handleUpdateMarker(markerValue || null)}
                        className="px-4 py-2 bg-indigo-600 text-white rounded-md hover:bg-indigo-700 transition-colors flex items-center gap-2"
                        disabled={updatingMarker || !markerValue.trim()}
                      >
                        {updatingMarker ? (
                          <>
                            <div className="animate-spin h-4 w-4 border-2 border-white border-t-transparent rounded-full" />
                            Updating...
                          </>
                        ) : (
                          <>
                            <Tag className="h-4 w-4" />
                            Set Marker
                          </>
                        )}
                      </button>
                    </div>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      )}

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
                    <th scope="col" className="px-6 py-3 text-left">
                      <input
                        type="checkbox"
                        checked={
                          runs.length > 0 && selectedRuns.size === runs.length
                        }
                        onChange={toggleSelectAll}
                        className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded cursor-pointer"
                        aria-label="Select all runs"
                      />
                    </th>
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
                      <div className="flex items-center">
                        <Tag className="h-4 w-4 mr-1 text-indigo-600" />
                        Marker
                      </div>
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
                        <input
                          type="checkbox"
                          checked={selectedRuns.has(run.id)}
                          onChange={() => toggleRunSelection(run.id)}
                          className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded cursor-pointer"
                          aria-label={`Select ${run.name || run.id}`}
                        />
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <Link
                          to={`/suite_runs/${run.id}`}
                          className="text-blue-600 hover:text-blue-800 font-medium hover:underline focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 rounded"
                        >
                          {run.name || run.id}
                        </Link>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        {run.metadata?.MARKER ? (
                          <Link
                            to={`/marker/${encodeURIComponent(
                              run.metadata.MARKER as string,
                            )}/stats`}
                            className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-indigo-100 text-indigo-800 hover:bg-indigo-200 transition-colors"
                          >
                            <Tag className="h-3 w-3 mr-1" />
                            {run.metadata.MARKER as string}
                          </Link>
                        ) : (
                          <span className="text-gray-400 text-sm italic">
                            No marker
                          </span>
                        )}
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
                          {run.statistics!.failed +
                            (run.statistics!.broken || 0) +
                            (run.statistics!.timedout || 0) +
                            (run.statistics!.interrupted || 0)}
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
