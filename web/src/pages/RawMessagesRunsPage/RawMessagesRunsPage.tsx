import { useCallback, useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { apiUrl } from "@/lib/config";
import { Card, CardContent } from "@/components/Card";
import { FileText, ArrowRight, RefreshCw } from "lucide-react";
import type {
  RawMessagesRunListResponse,
  RawMessagesRunSummary,
} from "@/types/rawMessages";

export function RawMessagesRunsPage() {
  const [runs, setRuns] = useState<RawMessagesRunSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [notEnabled, setNotEnabled] = useState(false);

  const fetchRuns = useCallback(async () => {
    try {
      setLoading(true);
      const response = await fetch(apiUrl("/raw-messages/runs?limit=200"));

      if (response.status === 404) {
        const body = await response.text();
        if (body.includes("retention")) {
          setNotEnabled(true);
          setRuns([]);
          return;
        }
      }

      if (!response.ok) {
        throw new Error(
          `Failed to fetch retained-message runs: ${response.statusText}`,
        );
      }

      const data: RawMessagesRunListResponse = await response.json();
      setRuns(data.runs ?? []);
      setError(null);
      setNotEnabled(false);
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Failed to fetch retained-message runs",
      );
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchRuns();
  }, [fetchRuns]);

  if (loading) {
    return (
      <div className="space-y-4">
        <h1 className="text-3xl font-bold text-(--stitch-on-surface)">Raw Message Runs</h1>
        <div className="space-y-2">
          {[1, 2, 3, 4].map((i) => (
            <div
              key={i}
              className="h-16 bg-(--stitch-surface-low) rounded-lg animate-pulse"
            />
          ))}
        </div>
      </div>
    );
  }

  if (notEnabled) {
    return (
      <Card className="border border-(--status-warning-border) bg-(--status-warning-soft)/50">
        <CardContent className="py-10 text-center">
          <h2 className="text-lg font-semibold text-(--stitch-on-surface) mb-2">
            Message Retention Not Enabled
          </h2>
          <p className="text-sm text-(--stitch-on-surface-muted)">
            Start the processor with{" "}
            <span className="font-mono">RETAIN_MESSAGES=true</span> to populate
            this directory.
          </p>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between flex-wrap gap-3">
        <div>
          <h1 className="text-3xl font-bold text-(--stitch-on-surface)">Raw Message Runs</h1>
          <p className="text-sm text-(--stitch-on-surface-subtle) mt-1">
            Directory of runs with retained NATS messages
          </p>
        </div>
        <button
          onClick={fetchRuns}
          className="inline-flex items-center gap-2 px-4 py-2 bg-(--stitch-surface-card) text-(--stitch-on-surface-muted) border border-(--stitch-outline) rounded-lg hover:bg-(--stitch-surface-low) hover:border-(--stitch-outline) transition-colors shadow-sm"
        >
          <RefreshCw className="h-4 w-4" />
          Refresh
        </button>
      </div>

      {error && (
        <Card className="border border-(--status-failure-border) bg-(--status-failure-soft)/50">
          <CardContent className="py-4 text-(--status-failure)">{error}</CardContent>
        </Card>
      )}

      {runs.length === 0 ? (
        <Card>
          <CardContent className="py-12 text-center text-(--stitch-on-surface-subtle)">
            No runs with retained messages yet.
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-3">
          {runs.map((run) => (
            <Link
              key={run.runId}
              to={`/suite_runs/${run.runId}/raw-messages`}
              className="block"
            >
              <Card className="hover:border-(--status-running-border) hover:shadow-md transition-all">
                <CardContent className="py-4">
                  <div className="flex items-center justify-between gap-4">
                    <div className="min-w-0">
                      <p className="font-mono text-sm text-(--stitch-on-surface) truncate">
                        {run.runId}
                      </p>
                      <p className="text-xs text-(--stitch-on-surface-subtle) mt-1">
                        Updated {new Date(run.updatedAt).toLocaleString()}
                      </p>
                    </div>
                    <div className="flex items-center gap-3 shrink-0">
                      <span className="inline-flex items-center gap-1.5 px-3 py-1.5 bg-(--stitch-surface-card) border border-(--stitch-outline) rounded-lg text-sm font-medium text-(--stitch-on-surface-muted)">
                        <FileText className="h-4 w-4 text-(--stitch-on-surface-subtle)" />
                        {run.messageCount} message
                        {run.messageCount !== 1 ? "s" : ""}
                      </span>
                      <ArrowRight className="h-4 w-4 text-(--stitch-on-surface-subtle)" />
                    </div>
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
