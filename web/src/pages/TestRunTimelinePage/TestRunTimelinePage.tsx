import {
  startTransition,
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";
import { Link, useParams } from "react-router-dom";
import { ArrowLeft, Clock, Map as MapIcon } from "lucide-react";
import { Badge } from "@/components/Badge";
import { Card, CardContent } from "@/components/Card";
import { config } from "@/lib/config";
import { useRefresh } from "@/lib/refresh";
import { fetchRunDetailData } from "@/lib/runDetail";
import type { TestRun } from "@/types/testRun";
import { humanizeMilliseconds } from "@/utils/duration";
import { STATUS_ORDER } from "./constants";
import { TimelineChart } from "./TimelineChart";
import { TimelineMetric } from "./TimelineMetric";
import { buildTimelineModel } from "./timelineModel";
import { metricValue, normalizeStatus } from "./timelineFormatting";
import { TimelineRunWindowCard } from "./TimelineRunWindowCard";
import { TimelineStatusBreakdown } from "./TimelineStatusBreakdown";

type FetchRunDetailOptions = {
  silent?: boolean;
  shouldIgnore?: () => boolean;
};

export function TestRunTimelinePage() {
  const pollIntervalMs = config.pollingIntervalMs;
  const { autoRefreshEnabled } = useRefresh();
  const { runId } = useParams<{ runId: string }>();
  const [runDetail, setRunDetail] = useState<TestRun | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchRunDetail = useCallback(
    async (id: string, options?: FetchRunDetailOptions) => {
      const silent = options?.silent ?? false;
      const shouldIgnore = options?.shouldIgnore ?? (() => false);

      try {
        if (!silent) {
          setLoading(true);
        }

        const data = await fetchRunDetailData(id);

        if (shouldIgnore()) {
          return;
        }

        startTransition(() => {
          setRunDetail(data);
          setError(null);
          if (!silent) {
            setLoading(false);
          }
        });
      } catch (err) {
        if (shouldIgnore()) {
          return;
        }

        console.error("Error fetching run timeline:", err);
        setError(
          err instanceof Error ? err.message : "Failed to fetch run timeline",
        );
        if (!silent) {
          setLoading(false);
        }
      }
    },
    [],
  );

  useEffect(() => {
    if (!runId) {
      return undefined;
    }

    let cancelled = false;

    void fetchRunDetail(runId, {
      shouldIgnore: () => cancelled,
    });

    return () => {
      cancelled = true;
    };
  }, [fetchRunDetail, runId]);

  useEffect(() => {
    if (!runId || !autoRefreshEnabled) {
      return;
    }

    let cancelled = false;
    const intervalId = window.setInterval(() => {
      void fetchRunDetail(runId, {
        silent: true,
        shouldIgnore: () => cancelled,
      });
    }, pollIntervalMs);

    return () => {
      cancelled = true;
      window.clearInterval(intervalId);
    };
  }, [autoRefreshEnabled, fetchRunDetail, pollIntervalMs, runId]);

  const timeline = useMemo(() => buildTimelineModel(runDetail), [runDetail]);
  const orderedStatuses = useMemo(
    () =>
      STATUS_ORDER.filter(
        (status) => (timeline?.statusCounts[status] ?? 0) > 0,
      ),
    [timeline],
  );

  if (loading) {
    return (
      <div className="space-y-6 animate-in fade-in duration-300">
        <div className="flex items-center gap-4">
          <div className="h-10 w-10 rounded-lg bg-(--stitch-surface-highest) animate-pulse" />
          <div className="space-y-2">
            <div className="h-8 w-56 rounded bg-(--stitch-surface-highest) animate-pulse" />
            <div className="h-4 w-64 rounded bg-(--stitch-surface-low) animate-pulse" />
          </div>
        </div>

        <div className="grid gap-4 md:grid-cols-4">
          {[1, 2, 3, 4].map((item) => (
            <div
              key={item}
              className="h-28 rounded-lg bg-(--stitch-surface-low) animate-pulse"
            />
          ))}
        </div>

        <div className="h-[480px] rounded-xl bg-(--stitch-surface-low) animate-pulse" />
      </div>
    );
  }

  if (error || !runDetail) {
    return (
      <div className="space-y-6 animate-in fade-in duration-300">
        <Link
          to={runId ? `/runs/${runId}` : "/runs"}
          className="group inline-flex items-center gap-2 rounded-md px-2 py-1 text-(--stitch-primary) transition-colors hover:bg-(--stitch-primary-soft) focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-(--stitch-primary) focus-visible:ring-offset-2 focus-visible:ring-offset-(--stitch-background)"
        >
          <ArrowLeft className="h-5 w-5 group-hover:-translate-x-1 transition-transform" />
          <span className="font-medium">Back to Test Run</span>
        </Link>

        <Card
          className="border-(--status-failure-border)"
          style={{ backgroundColor: "var(--status-failure-soft)" }}
        >
          <CardContent className="py-12">
            <div className="mx-auto max-w-md text-center">
              <div
                className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full"
                style={{ backgroundColor: "var(--status-failure-soft)" }}
              >
                <Clock className="h-8 w-8 text-(--status-failure)" />
              </div>
              <h3 className="mb-2 text-lg font-semibold text-(--stitch-on-surface)">
                {error ? "Failed to Load Timeline" : "Timeline Not Available"}
              </h3>
              <p className="mb-6 text-sm text-(--stitch-on-surface-muted)">
                {error || "The requested run could not be loaded."}
              </p>
              <Link
                to={runId ? `/runs/${runId}` : "/runs"}
                className="inline-flex items-center rounded-lg px-4 py-2 text-white transition-opacity hover:opacity-90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-(--stitch-primary) focus-visible:ring-offset-2 focus-visible:ring-offset-(--stitch-background)"
                style={{
                  background:
                    "linear-gradient(135deg, var(--stitch-primary), var(--stitch-primary-end))",
                }}
              >
                Return to Run Overview
              </Link>
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  if (!timeline || timeline.totalAttempts === 0) {
    return (
      <div className="space-y-6 animate-in fade-in duration-300">
        <div className="flex items-center justify-between gap-4">
          <div className="flex items-center gap-4">
            <Link
              to={`/runs/${runId}`}
              className="group inline-flex h-10 w-10 items-center justify-center rounded-lg border border-(--stitch-outline) bg-(--stitch-surface-card) text-(--stitch-on-surface-muted) transition-colors hover:bg-(--stitch-surface-low) focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-(--stitch-primary) focus-visible:ring-offset-2 focus-visible:ring-offset-(--stitch-background)"
              aria-label="Back to test run"
            >
              <ArrowLeft className="h-5 w-5 group-hover:-translate-x-0.5 transition-transform" />
            </Link>
            <div>
              <h1 className="text-2xl font-bold tracking-tight text-(--stitch-on-surface) md:text-3xl">
                Timeline
              </h1>
              <p className="mt-1 text-sm text-(--stitch-on-surface-muted)">
                {runDetail.name || runDetail.id}
              </p>
            </div>
          </div>
        </div>

        <Card className="border border-(--stitch-outline) shadow-sm">
          <CardContent className="py-12 text-center">
            <h2 className="text-lg font-semibold text-(--stitch-on-surface)">
              No attempt timeline data yet
            </h2>
            <p className="mx-auto mt-2 max-w-xl text-sm text-(--stitch-on-surface-muted)">
              This run currently has no attempt timing windows that can be
              plotted. Once test attempts start reporting timestamps, they will
              appear here.
            </p>
          </CardContent>
        </Card>
      </div>
    );
  }

  const resolvedRunId = runId ?? runDetail.id;

  return (
    <div className="space-y-6 pb-8 animate-in fade-in duration-300">
      <div className="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
        <div className="flex items-center gap-4">
          <Link
            to={`/runs/${resolvedRunId}`}
            className="group inline-flex h-10 w-10 items-center justify-center rounded-lg border border-(--stitch-outline) bg-(--stitch-surface-card) text-(--stitch-on-surface-muted) transition-colors hover:bg-(--stitch-surface-low) focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-(--stitch-primary) focus-visible:ring-offset-2 focus-visible:ring-offset-(--stitch-background)"
            aria-label="Back to test run"
          >
            <ArrowLeft className="h-5 w-5 group-hover:-translate-x-0.5 transition-transform" />
          </Link>

          <div>
            <div className="flex flex-wrap items-center gap-2">
              <span className="inline-flex items-center gap-2 rounded-full border border-(--stitch-outline) bg-(--stitch-surface-card) px-3 py-1 text-xs font-semibold uppercase tracking-[0.16em] text-(--stitch-on-surface-subtle)">
                <Clock className="h-3.5 w-3.5 text-(--stitch-primary)" />
                Timeline
              </span>
              <Badge
                status={normalizeStatus(runDetail.status)}
                className="hidden sm:inline-flex"
              />
            </div>
            <h1 className="mt-3 text-2xl font-bold tracking-tight text-(--stitch-on-surface) md:text-3xl">
              {runDetail.name || runDetail.id}
            </h1>
            <p className="mt-1 max-w-2xl text-sm text-(--stitch-on-surface-muted)">
              All attempt windows derived from the run detail payload. Swimlanes
              are grouped by execution id until worker indices are available.
            </p>
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <Link
            to={`/runs/${resolvedRunId}`}
            className="inline-flex items-center gap-2 rounded-lg border border-(--stitch-outline) bg-(--stitch-surface-card) px-4 py-2 text-(--stitch-on-surface) transition-colors hover:bg-(--stitch-surface-low) focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-(--stitch-primary) focus-visible:ring-offset-2 focus-visible:ring-offset-(--stitch-background)"
          >
            <ArrowLeft className="h-4 w-4" />
            <span className="font-medium">Run Overview</span>
          </Link>
          <Link
            to={`/runs/${resolvedRunId}/map`}
            className="inline-flex items-center gap-2 rounded-lg px-4 py-2 text-white transition-opacity hover:opacity-90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-(--stitch-primary) focus-visible:ring-offset-2 focus-visible:ring-offset-(--stitch-background)"
            style={{
              background:
                "linear-gradient(135deg, var(--stitch-primary), var(--stitch-primary-end))",
            }}
          >
            <MapIcon className="h-4 w-4" />
            <span className="font-medium">View Test Map</span>
          </Link>
        </div>
      </div>

      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <TimelineMetric
          label="Execution Lanes"
          value={metricValue(timeline.lanes.length)}
          detail="Distinct execution ids with plotted attempts"
        />
        <TimelineMetric
          label="Attempt Windows"
          value={metricValue(timeline.totalAttempts)}
          detail="Every attempt found under run.tests[].attempts"
        />
        <TimelineMetric
          label="Timeline Span"
          value={humanizeMilliseconds(timeline.spanMs)}
          detail={`${new Date(timeline.startMs).toLocaleString()} to ${new Date(
            timeline.endMs,
          ).toLocaleString()}`}
        />
        <TimelineMetric
          label="Peak Lane Overlap"
          value={metricValue(timeline.maxLaneRowCount)}
          detail="Stacked rows needed to keep overlapping attempts visible"
        />
      </div>

      <TimelineStatusBreakdown
        orderedStatuses={orderedStatuses}
        statusCounts={timeline.statusCounts}
        approximateCount={timeline.approximateCount}
        syntheticCount={timeline.syntheticCount}
      />

      <TimelineChart runId={resolvedRunId} timeline={timeline} />

      <TimelineRunWindowCard
        spanMs={timeline.spanMs}
        totalAttempts={timeline.totalAttempts}
        runDuration={runDetail.duration}
      />
    </div>
  );
}
