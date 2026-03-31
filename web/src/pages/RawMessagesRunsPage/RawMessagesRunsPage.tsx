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
        <h1 className="text-3xl font-bold text-gray-900">Raw Message Runs</h1>
        <div className="space-y-2">
          {[1, 2, 3, 4].map((i) => (
            <div
              key={i}
              className="h-16 bg-gray-100 rounded-lg animate-pulse"
            />
          ))}
        </div>
      </div>
    );
  }

  if (notEnabled) {
    return (
      <Card className="border-yellow-200 bg-yellow-50/50">
        <CardContent className="py-10 text-center">
          <h2 className="text-lg font-semibold text-gray-900 mb-2">
            Message Retention Not Enabled
          </h2>
          <p className="text-sm text-gray-600">
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
          <h1 className="text-3xl font-bold text-gray-900">Raw Message Runs</h1>
          <p className="text-sm text-gray-500 mt-1">
            Directory of runs with retained NATS messages
          </p>
        </div>
        <button
          onClick={fetchRuns}
          className="inline-flex items-center gap-2 px-4 py-2 bg-white text-gray-700 border border-gray-200 rounded-lg hover:bg-gray-50 hover:border-gray-300 transition-colors shadow-sm"
        >
          <RefreshCw className="h-4 w-4" />
          Refresh
        </button>
      </div>

      {error && (
        <Card className="border-red-200 bg-red-50/50">
          <CardContent className="py-4 text-red-700">{error}</CardContent>
        </Card>
      )}

      {runs.length === 0 ? (
        <Card>
          <CardContent className="py-12 text-center text-gray-500">
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
              <Card className="hover:border-blue-300 hover:shadow-md transition-all">
                <CardContent className="py-4">
                  <div className="flex items-center justify-between gap-4">
                    <div className="min-w-0">
                      <p className="font-mono text-sm text-gray-900 truncate">
                        {run.runId}
                      </p>
                      <p className="text-xs text-gray-500 mt-1">
                        Updated {new Date(run.updatedAt).toLocaleString()}
                      </p>
                    </div>
                    <div className="flex items-center gap-3 shrink-0">
                      <span className="inline-flex items-center gap-1.5 px-3 py-1.5 bg-white border border-gray-200 rounded-lg text-sm font-medium text-gray-700">
                        <FileText className="h-4 w-4 text-gray-400" />
                        {run.messageCount} message
                        {run.messageCount !== 1 ? "s" : ""}
                      </span>
                      <ArrowRight className="h-4 w-4 text-gray-400" />
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
