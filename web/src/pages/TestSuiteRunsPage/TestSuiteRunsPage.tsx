import { useEffect, useState, useCallback } from "react";
import { Link } from "react-router-dom";
import { apiUrl, config } from "@/lib/config";
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
  FileText,
  RefreshCw,
} from "lucide-react";

import type { TestRun } from "@/types/testRun";
import { getRunCompletionStatus, getRunStatus } from "./utils";
import type { TestStatus } from "@/types/common";

export function TestSuiteRunsPage() {
  const pollIntervalMs = config.pollingIntervalMs;
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
        <div className="text-(--stitch-on-surface-muted)">
          Loading test runs...
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-(--status-failure)">Error: {error}</div>
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
        <h1 className="text-3xl font-bold text-(--stitch-on-surface)">
          Test Suite Runs
        </h1>
        <div className="flex gap-2">
          <Link
            to="/suite_runs/raw-messages"
            className="px-4 py-2 bg-(--stitch-surface-card) text-(--stitch-on-surface) border border-(--stitch-outline) rounded-md hover:bg-(--stitch-surface-card) transition-colors flex items-center gap-2"
          >
            <FileText className="h-4 w-4" />
            Raw Messages
          </Link>
          {selectedRuns.size > 0 && (
            <>
              <button
                onClick={handleSetMarker}
                className="px-4 py-2 bg-(--stitch-primary) text-white rounded-md hover:brightness-105 transition-colors flex items-center gap-2"
                disabled={updatingMarker}
              >
                <Tag className="h-4 w-4" />
                Set Marker
              </button>
              <button
                onClick={handleRemoveMarker}
                className="px-4 py-2 bg-(--stitch-surface-low) text-(--stitch-on-surface) rounded-md hover:bg-(--stitch-surface-card) transition-colors flex items-center gap-2"
                disabled={updatingMarker}
              >
                <X className="h-4 w-4" />
                Remove Marker
              </button>
              <button
                onClick={() => setShowDeleteConfirm(true)}
                className="px-4 py-2 bg-(--status-failure) text-white rounded-md hover:brightness-105 transition-colors flex items-center gap-2"
                disabled={deleting}
              >
                <Trash2 className="h-4 w-4" />
                Delete ({selectedRuns.size})
              </button>
            </>
          )}
          <button
            onClick={() => fetchRuns()}
            className="inline-flex items-center gap-2 rounded-md px-4 py-2 text-sm font-medium text-(--stitch-on-primary) shadow-sm transition-all hover:brightness-105 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-(--stitch-primary) focus-visible:ring-offset-2"
            style={{
              backgroundImage:
                "linear-gradient(135deg, var(--stitch-primary), var(--stitch-primary-strong))",
            }}
          >
            <RefreshCw className="h-4 w-4" />
            Refresh
          </button>
        </div>
      </div>

      {error && (
        <Card className="border border-(--status-failure-border) bg-(--status-failure-soft)">
          <CardContent className="py-4">
            <div className="flex items-center gap-2 text-(--status-failure)">
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
                <div className="shrink-0">
                  <AlertCircle className="h-6 w-6 text-(--status-failure)" />
                </div>
                <div className="flex-1">
                  <h3 className="text-lg font-semibold text-(--stitch-on-surface) mb-2">
                    Delete Test Runs
                  </h3>
                  <p className="text-(--stitch-on-surface-muted) mb-4">
                    Are you sure you want to delete {selectedRuns.size} test run
                    {selectedRuns.size !== 1 ? "s" : ""}? This action cannot be
                    undone.
                  </p>
                  <div className="flex gap-3 justify-end">
                    <button
                      onClick={() => setShowDeleteConfirm(false)}
                      className="px-4 py-2 border border-(--stitch-outline) rounded-md hover:bg-(--stitch-surface-card) transition-colors"
                      disabled={deleting}
                    >
                      Cancel
                    </button>
                    <button
                      onClick={handleDeleteSelected}
                      className="px-4 py-2 bg-(--status-failure) text-white rounded-md hover:brightness-105 transition-colors flex items-center gap-2"
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
                  <div className="shrink-0">
                    <Tag className="h-6 w-6 text-(--stitch-primary)" />
                  </div>
                  <div className="flex-1">
                    <h3 className="text-lg font-semibold text-(--stitch-on-surface) mb-2">
                      Set Marker for Test Runs
                    </h3>
                    <p className="text-(--stitch-on-surface-muted) mb-4">
                      Set a marker for {selectedRuns.size} test run
                      {selectedRuns.size !== 1 ? "s" : ""}. Markers help
                      organize and filter runs.
                    </p>
                    <div className="mb-4">
                      <label
                        htmlFor="marker-input"
                        className="block text-sm font-medium text-(--stitch-on-surface) mb-2"
                      >
                        Marker Value
                      </label>
                      <input
                        id="marker-input"
                        type="text"
                        value={markerValue}
                        onChange={(e) => setMarkerValue(e.target.value)}
                        placeholder="e.g., release-1.0, nightly, staging"
                        className="w-full px-3 py-2 border border-(--stitch-outline) rounded-md focus:outline-none focus:ring-2 focus:ring-(--stitch-primary)"
                        disabled={updatingMarker}
                      />
                    </div>
                    <div className="flex gap-3 justify-end">
                      <button
                        onClick={() => {
                          setShowMarkerDialog(false);
                          setMarkerValue("");
                        }}
                        className="px-4 py-2 border border-(--stitch-outline) rounded-md hover:bg-(--stitch-surface-card) transition-colors"
                        disabled={updatingMarker}
                      >
                        Cancel
                      </button>
                      <button
                        onClick={() => handleUpdateMarker(markerValue || null)}
                        className="px-4 py-2 bg-(--stitch-primary) text-white rounded-md hover:brightness-105 transition-colors flex items-center gap-2"
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
              <Play className="mx-auto h-12 w-12 text-(--stitch-on-surface-muted)" />
              <h3 className="mt-2 text-sm font-medium text-(--stitch-on-surface)">
                No test runs found
              </h3>
              <p className="mt-1 text-sm text-(--stitch-on-surface-muted)">
                Test suite runs will appear here once tests are executed.
              </p>
            </div>
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardContent className="p-0">
            <div className="overflow-x-auto">
              <table className="min-w-[1220px] w-full table-auto divide-y divide-(--stitch-outline)">
                <thead className="bg-(--stitch-surface-card)">
                  <tr>
                    <th scope="col" className="px-6 py-3 text-left w-12">
                      <input
                        type="checkbox"
                        checked={
                          runs.length > 0 && selectedRuns.size === runs.length
                        }
                        onChange={toggleSelectAll}
                        className="h-4 w-4 text-(--stitch-primary) focus:ring-(--stitch-primary) border-(--stitch-outline) rounded cursor-pointer"
                        aria-label="Select all runs"
                      />
                    </th>
                    <th
                      scope="col"
                      className="px-6 py-3 text-left text-xs font-medium text-(--stitch-on-surface-muted) uppercase tracking-wider w-152"
                    >
                      Run Name
                    </th>
                    <th
                      scope="col"
                      className="px-6 py-3 text-left text-xs font-medium text-(--stitch-on-surface-muted) uppercase tracking-wider w-56"
                    >
                      <div className="flex items-center">
                        <Tag className="h-4 w-4 mr-1 text-(--stitch-primary)" />
                        Marker
                      </div>
                    </th>
                    <th
                      scope="col"
                      className="px-6 py-3 text-left text-xs font-medium text-(--stitch-on-surface-muted) uppercase tracking-wider w-40"
                    >
                      Status
                    </th>
                    <th
                      scope="col"
                      className="px-6 py-3 text-left text-xs font-medium text-(--stitch-on-surface-muted) uppercase tracking-wider w-40"
                    >
                      Result
                    </th>
                    <th
                      scope="col"
                      className="px-6 py-3 text-center text-xs font-medium text-(--stitch-on-surface-muted) uppercase tracking-wider w-40"
                    >
                      <div className="inline-flex items-center justify-center">
                        <span className="inline-flex items-center justify-center">
                          <CheckCircle className="h-4 w-4 mr-1 text-(--status-success)" />
                          Passed
                        </span>
                        {" + "}
                        <span className="inline-flex items-center justify-center">
                          <XCircle className="h-4 w-4 mr-1 text-(--status-failure)" />
                          Failed
                        </span>
                        {" + "}
                        <span className="inline-flex items-center justify-center">
                          <CircleDashed className="h-4 w-4 mr-1 text-(--stitch-on-surface-muted)" />
                          Skipped
                        </span>
                      </div>
                      <div className="inline-flex items-center justify-center">
                        {" / "}
                        <div className="flex items-center justify-center">
                          <Play className="h-4 w-4 mr-1 text-(--stitch-primary)" />
                          Total
                        </div>
                      </div>
                    </th>
                    <th
                      scope="col"
                      className="px-6 py-3 text-left text-xs font-medium text-(--stitch-on-surface-muted) uppercase tracking-wider w-56"
                    >
                      <button
                        onClick={toggleSortOrder}
                        className="flex items-center hover:text-(--stitch-on-surface) transition-colors focus:outline-none focus:ring-2 focus:ring-(--stitch-primary) focus:ring-offset-2 rounded-md px-2 py-1 -mx-2 -my-1"
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
                <tbody className="bg-(--stitch-surface-card) divide-y divide-(--stitch-outline)">
                  {sortedRuns.map((run) => {
                    const status = getRunStatus(run);
                    const runCompletionStatus = getRunCompletionStatus(
                      run.status as TestStatus,
                    );
                    return (
                      <tr
                        key={run.id}
                        className="hover:bg-(--stitch-surface-card) transition-colors"
                      >
                        <td className="px-6 py-4 w-12">
                          <input
                            type="checkbox"
                            checked={selectedRuns.has(run.id)}
                            onChange={() => toggleRunSelection(run.id)}
                            className="h-4 w-4 text-(--stitch-primary) focus:ring-(--stitch-primary) border-(--stitch-outline) rounded cursor-pointer"
                            aria-label={`Select ${run.name || run.id}`}
                          />
                        </td>
                        <td className="px-6 py-4 whitespace-normal wrap-break-word max-w-152">
                          <Link
                            to={`/suite_runs/${run.id}`}
                            className="text-(--stitch-primary) hover:text-(--stitch-primary) font-medium hover:underline focus:outline-none focus:ring-2 focus:ring-(--stitch-primary) focus:ring-offset-2 rounded"
                          >
                            {run.name || run.id}
                          </Link>
                        </td>
                        <td className="px-6 py-4 whitespace-normal wrap-break-word max-w-56">
                          {run.metadata?.MARKER ? (
                            <Link
                              to={`/marker/${encodeURIComponent(
                                run.metadata.MARKER as string,
                              )}/stats`}
                              className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-(--stitch-primary-soft) text-(--stitch-primary) hover:bg-(--stitch-primary-soft) transition-colors"
                            >
                              <Tag className="h-3 w-3 mr-1" />
                              {run.metadata.MARKER as string}
                            </Link>
                          ) : (
                            <span className="text-(--stitch-on-surface-muted) text-sm italic">
                              No marker
                            </span>
                          )}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap">
                          <Badge status={runCompletionStatus} />
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap">
                          {status !== "RUNNING" && <Badge status={status} />}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-center">
                          <span className="text-(--status-success) font-semibold">
                            {run.statistics!.passed}
                          </span>
                          {" + "}
                          <span className="text-(--status-failure) font-semibold">
                            {run.statistics!.failed +
                              (run.statistics!.broken || 0) +
                              (run.statistics!.timedout || 0) +
                              (run.statistics!.interrupted || 0)}
                          </span>
                          {" + "}
                          <span className="text-(--stitch-on-surface-muted) font-semibold">
                            {run.statistics!.skipped}
                          </span>
                          {" / "}
                          <span className="text-(--stitch-primary) font-semibold">
                            {run.statistics!.total}
                          </span>
                        </td>
                        <td className="px-6 py-4 whitespace-normal text-sm text-(--stitch-on-surface-muted)">
                          {run.updatedAt ? (
                            <div className="flex flex-col">
                              <span>
                                {new Date(run.updatedAt).toLocaleDateString()}
                              </span>
                              <span className="text-xs text-(--stitch-on-surface-muted)">
                                {new Date(run.updatedAt).toLocaleTimeString()}
                              </span>
                            </div>
                          ) : (
                            <span className="text-(--stitch-on-surface-muted)">
                              N/A
                            </span>
                          )}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
