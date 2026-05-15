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
import { cn } from "@/lib/utils";
import type { TestStatus } from "@/types/common";
import type { Attempt, Test } from "@/types/testCase";
import type { RunExecution, TestRun } from "@/types/testRun";
import { humanizeDuration, humanizeMilliseconds } from "@/utils/duration";

const LABEL_COLUMN_WIDTH_PX = 264;
const TIMELINE_MIN_WIDTH_PX = 960;
const TIMELINE_MAX_WIDTH_PX = 4200;
const ROW_HEIGHT_PX = 54;
const ROW_VERTICAL_PADDING_PX = 10;

const STATUS_ORDER: TestStatus[] = [
  "RUNNING",
  "FAILED",
  "BROKEN",
  "TIMEDOUT",
  "INTERRUPTED",
  "FLAKY",
  "PASSED",
  "SKIPPED",
  "PENDING",
  "NOT_RUN",
  "UNKNOWN",
];

const NICE_TICK_INTERVALS_MS = [
  100, 250, 500, 1_000, 2_000, 5_000, 10_000, 15_000, 30_000, 60_000, 120_000,
  300_000, 600_000, 900_000, 1_800_000, 3_600_000, 7_200_000, 14_400_000,
  28_800_000, 43_200_000, 86_400_000,
];

type FetchRunDetailOptions = {
  silent?: boolean;
  shouldIgnore?: () => boolean;
};

type TimelineBar = {
  key: string;
  testId: string;
  testTitle: string;
  executionId: string;
  status: TestStatus;
  attemptIndex: number;
  startMs: number;
  endMs: number;
  durationMs: number;
  rowIndex: number;
  synthetic: boolean;
  approximated: boolean;
  location?: string;
  tags: string[];
};

type TimelineLane = {
  key: string;
  executionId: string;
  label: string;
  subtitle: string;
  status?: TestStatus;
  statusLabel?: string;
  bars: TimelineBar[];
  rowCount: number;
  earliestMs: number;
  latestMs: number;
  durationMs: number;
};

type TimelineModel = {
  lanes: TimelineLane[];
  totalAttempts: number;
  approximateCount: number;
  syntheticCount: number;
  startMs: number;
  endMs: number;
  spanMs: number;
  chartWidthPx: number;
  ticks: number[];
  maxLaneRowCount: number;
  statusCounts: Partial<Record<TestStatus, number>>;
};

type AttemptCandidate = {
  attempt: Attempt;
  synthetic: boolean;
};

const DEFAULT_STATUS_STYLES = {
  backgroundColor: "var(--status-neutral-soft)",
  borderColor: "var(--status-neutral-border)",
  color: "var(--status-neutral)",
};

function toTimestamp(value?: string): number | null {
  if (!value) {
    return null;
  }

  const parsed = Date.parse(value);
  return Number.isNaN(parsed) ? null : parsed;
}

function normalizeStatus(status?: string): TestStatus {
  switch (status?.toUpperCase()) {
    case "PASSED":
      return "PASSED";
    case "FLAKY":
      return "FLAKY";
    case "FAILED":
      return "FAILED";
    case "SKIPPED":
      return "SKIPPED";
    case "RUNNING":
      return "RUNNING";
    case "PENDING":
      return "PENDING";
    case "BROKEN":
      return "BROKEN";
    case "TIMEDOUT":
      return "TIMEDOUT";
    case "INTERRUPTED":
      return "INTERRUPTED";
    case "NOT_RUN":
      return "NOT_RUN";
    default:
      return "UNKNOWN";
  }
}

function statusStyles(status: TestStatus): {
  backgroundColor: string;
  borderColor: string;
  color: string;
} {
  switch (status) {
    case "PASSED":
      return {
        backgroundColor: "var(--status-success-soft)",
        borderColor: "var(--status-success-border)",
        color: "var(--status-success)",
      };
    case "FLAKY":
      return {
        backgroundColor: "var(--status-warning-soft)",
        borderColor: "var(--status-warning-border)",
        color: "var(--status-warning)",
      };
    case "FAILED":
      return {
        backgroundColor: "var(--status-failure-soft)",
        borderColor: "var(--status-failure-border)",
        color: "var(--status-failure)",
      };
    case "RUNNING":
      return {
        backgroundColor: "var(--status-running-soft)",
        borderColor: "var(--status-running-border)",
        color: "var(--status-running)",
      };
    case "BROKEN":
      return {
        backgroundColor: "var(--status-broken-soft)",
        borderColor: "var(--status-broken-border)",
        color: "var(--status-broken)",
      };
    case "TIMEDOUT":
      return {
        backgroundColor: "var(--status-timedout-soft)",
        borderColor: "var(--status-timedout-border)",
        color: "var(--status-timedout)",
      };
    case "INTERRUPTED":
      return {
        backgroundColor: "var(--status-interrupted-soft)",
        borderColor: "var(--status-interrupted-border)",
        color: "var(--status-interrupted)",
      };
    default:
      return DEFAULT_STATUS_STYLES;
  }
}

function buildAttemptCandidates(test: Test): AttemptCandidate[] {
  if (test.attempts && test.attempts.length > 0) {
    return test.attempts.map((attempt) => ({ attempt, synthetic: false }));
  }

  return [
    {
      synthetic: true,
      attempt: {
        attemptIndex: test.retryIndex ?? 0,
        executionId: "",
        status: test.status,
        startTime: test.startTime,
        endTime: test.endTime,
        duration: test.duration,
        createdAt: test.createdAt,
        updatedAt: test.updatedAt,
      },
    },
  ];
}

function resolveAttemptWindow(
  run: TestRun,
  test: Test,
  attempt: Attempt,
): { startMs: number; endMs: number; approximated: boolean } | null {
  const explicitStartMs = toTimestamp(attempt.startTime);
  const explicitEndMs = toTimestamp(attempt.endTime);
  const createdAtMs = toTimestamp(attempt.createdAt);
  const updatedAtMs = toTimestamp(attempt.updatedAt);
  const testStartMs = toTimestamp(test.startTime);
  const testEndMs = toTimestamp(test.endTime);
  const testCreatedAtMs = toTimestamp(test.createdAt);
  const testUpdatedAtMs = toTimestamp(test.updatedAt);
  const runStartMs = toTimestamp(run.startTime) ?? toTimestamp(run.createdAt);
  const runEndMs = toTimestamp(run.endTime) ?? toTimestamp(run.updatedAt);
  const durationMs =
    typeof attempt.duration === "number" && Number.isFinite(attempt.duration)
      ? Math.max(attempt.duration / 1_000_000, 0)
      : null;

  let startMs =
    explicitStartMs ??
    createdAtMs ??
    testStartMs ??
    testCreatedAtMs ??
    runStartMs;
  let endMs =
    explicitEndMs ??
    (startMs !== null && durationMs !== null ? startMs + durationMs : null) ??
    updatedAtMs ??
    testEndMs ??
    testUpdatedAtMs ??
    runEndMs;

  if (startMs === null && endMs !== null && durationMs !== null) {
    startMs = endMs - durationMs;
  }

  if (startMs === null || endMs === null) {
    return null;
  }

  if (endMs < startMs) {
    endMs =
      durationMs !== null && durationMs > 0 ? startMs + durationMs : startMs;
  }

  if (endMs === startMs) {
    endMs = startMs + Math.max(durationMs ?? 0, 1);
  }

  return {
    startMs,
    endMs,
    approximated: explicitStartMs === null || explicitEndMs === null,
  };
}

function packLaneRows(bars: TimelineBar[]): {
  bars: TimelineBar[];
  rowCount: number;
} {
  const packedBars = [...bars].sort((left, right) => {
    if (left.startMs === right.startMs) {
      if (left.endMs === right.endMs) {
        return left.testTitle.localeCompare(right.testTitle);
      }
      return left.endMs - right.endMs;
    }
    return left.startMs - right.startMs;
  });

  const rowEndTimes: number[] = [];

  for (const bar of packedBars) {
    let rowIndex = rowEndTimes.findIndex((endMs) => bar.startMs >= endMs);

    if (rowIndex === -1) {
      rowIndex = rowEndTimes.length;
      rowEndTimes.push(bar.endMs);
    } else {
      rowEndTimes[rowIndex] = Math.max(rowEndTimes[rowIndex], bar.endMs);
    }

    bar.rowIndex = rowIndex;
  }

  return {
    bars: packedBars,
    rowCount: Math.max(rowEndTimes.length, 1),
  };
}

function pickTickInterval(spanMs: number): number {
  const rawInterval = spanMs / 6;
  const match = NICE_TICK_INTERVALS_MS.find(
    (candidate) => candidate >= rawInterval,
  );

  if (match) {
    return match;
  }

  const oneDayMs = 86_400_000;
  return Math.max(Math.ceil(rawInterval / oneDayMs) * oneDayMs, oneDayMs);
}

function buildTicks(spanMs: number): number[] {
  if (!Number.isFinite(spanMs) || spanMs <= 0) {
    return [0];
  }

  const intervalMs = pickTickInterval(spanMs);
  const ticks = [0];

  for (let offsetMs = intervalMs; offsetMs < spanMs; offsetMs += intervalMs) {
    ticks.push(offsetMs);
  }

  if (ticks[ticks.length - 1] !== spanMs) {
    ticks.push(spanMs);
  }

  return ticks;
}

function formatAxisTimestamp(timestampMs: number, spanMs: number): string {
  const date = new Date(timestampMs);

  if (spanMs >= 86_400_000) {
    return date.toLocaleString(undefined, {
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  }

  return date.toLocaleTimeString(undefined, {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}

function formatRelativeOffset(offsetMs: number): string {
  if (offsetMs <= 0) {
    return "+0ms";
  }

  if (offsetMs < 1_000) {
    return `+${Math.round(offsetMs)}ms`;
  }

  if (offsetMs < 60_000) {
    const seconds = offsetMs / 1_000;
    return `+${seconds >= 10 ? seconds.toFixed(0) : seconds.toFixed(1)}s`;
  }

  return `+${humanizeMilliseconds(offsetMs)}`;
}

function buildLaneLabel(
  execution: RunExecution | undefined,
  executionId: string,
  fallbackIndex: number,
): {
  label: string;
  subtitle: string;
  status?: TestStatus;
  statusLabel?: string;
} {
  if (execution?.isShard && typeof execution.shardIndex === "number") {
    const shardNumber = execution.shardIndex + 1;
    const suffix =
      typeof execution.shardCountExpected === "number"
        ? ` of ${execution.shardCountExpected}`
        : "";

    return {
      label: `Shard ${shardNumber}${suffix}`,
      subtitle: executionId || execution.name || "Unnamed execution",
      status: execution.status ? normalizeStatus(execution.status) : undefined,
      statusLabel: execution.status,
    };
  }

  if (execution?.name) {
    return {
      label: execution.name,
      subtitle: executionId || execution.name,
      status: execution.status ? normalizeStatus(execution.status) : undefined,
      statusLabel: execution.status,
    };
  }

  if (executionId) {
    return {
      label: `Execution ${fallbackIndex + 1}`,
      subtitle: executionId,
      status: execution?.status ? normalizeStatus(execution.status) : undefined,
      statusLabel: execution?.status,
    };
  }

  return {
    label: "Primary execution",
    subtitle: "No execution id",
    status: execution?.status ? normalizeStatus(execution.status) : undefined,
    statusLabel: execution?.status,
  };
}

function buildTimelineModel(runDetail: TestRun | null): TimelineModel | null {
  if (!runDetail) {
    return null;
  }

  const tests = runDetail.tests ?? [];
  const executions = runDetail.executions ?? [];
  const executionById = new Map(
    executions.map((execution) => [execution.id, execution]),
  );
  const executionOrder = new Map(
    executions.map((execution, index) => [execution.id, index]),
  );
  const statusCounts: Partial<Record<TestStatus, number>> = {};
  const laneBars = new Map<string, TimelineBar[]>();
  let approximateCount = 0;
  let syntheticCount = 0;
  let earliestMs = Number.POSITIVE_INFINITY;
  let latestMs = Number.NEGATIVE_INFINITY;

  for (const test of tests) {
    for (const candidate of buildAttemptCandidates(test)) {
      const window = resolveAttemptWindow(runDetail, test, candidate.attempt);
      if (!window) {
        continue;
      }

      const executionId = candidate.attempt.executionId ?? "";
      const laneKey = executionId || "__primary__";
      const status = normalizeStatus(candidate.attempt.status ?? test.status);
      const bar: TimelineBar = {
        key: [
          test.id,
          executionId || "primary",
          candidate.attempt.attemptIndex,
        ].join(":"),
        testId: test.id,
        testTitle: test.title || test.id,
        executionId,
        status,
        attemptIndex: candidate.attempt.attemptIndex,
        startMs: window.startMs,
        endMs: window.endMs,
        durationMs: Math.max(window.endMs - window.startMs, 1),
        rowIndex: 0,
        synthetic: candidate.synthetic,
        approximated: window.approximated,
        location: test.location,
        tags: test.tags ?? [],
      };

      earliestMs = Math.min(earliestMs, bar.startMs);
      latestMs = Math.max(latestMs, bar.endMs);
      approximateCount += bar.approximated ? 1 : 0;
      syntheticCount += bar.synthetic ? 1 : 0;
      statusCounts[status] = (statusCounts[status] ?? 0) + 1;
      laneBars.set(laneKey, [...(laneBars.get(laneKey) ?? []), bar]);
    }
  }

  if (!Number.isFinite(earliestMs) || !Number.isFinite(latestMs)) {
    const fallbackStart =
      toTimestamp(runDetail.startTime) ??
      toTimestamp(runDetail.createdAt) ??
      Date.now();
    const fallbackEnd =
      toTimestamp(runDetail.endTime) ??
      toTimestamp(runDetail.updatedAt) ??
      fallbackStart;

    earliestMs = fallbackStart;
    latestMs = Math.max(fallbackEnd, fallbackStart + 1);
  }

  const spanMs = Math.max(latestMs - earliestMs, 1);
  const timelineWidthPx = Math.max(
    TIMELINE_MIN_WIDTH_PX,
    Math.min(TIMELINE_MAX_WIDTH_PX, Math.round((spanMs / 1_000) * 16)),
  );

  const laneEntries = Array.from(laneBars.entries())
    .map(([laneKey, bars], laneIndex) => {
      const executionId = laneKey === "__primary__" ? "" : laneKey;
      const execution = executionId
        ? executionById.get(executionId)
        : undefined;
      const packed = packLaneRows(bars);
      const laneStartMs = Math.min(...packed.bars.map((bar) => bar.startMs));
      const laneEndMs = Math.max(...packed.bars.map((bar) => bar.endMs));
      const label = buildLaneLabel(
        execution,
        executionId,
        executionOrder.get(executionId) ?? laneIndex,
      );

      return {
        key: laneKey,
        executionId,
        label: label.label,
        subtitle: label.subtitle,
        status: label.status,
        statusLabel: label.statusLabel,
        bars: packed.bars,
        rowCount: packed.rowCount,
        earliestMs: laneStartMs,
        latestMs: laneEndMs,
        durationMs: Math.max(laneEndMs - laneStartMs, 1),
      } as TimelineLane;
    })
    .sort((left, right) => {
      const leftOrder = left.executionId
        ? executionOrder.get(left.executionId)
        : undefined;
      const rightOrder = right.executionId
        ? executionOrder.get(right.executionId)
        : undefined;

      if (typeof leftOrder === "number" && typeof rightOrder === "number") {
        return leftOrder - rightOrder;
      }

      if (typeof leftOrder === "number") {
        return -1;
      }

      if (typeof rightOrder === "number") {
        return 1;
      }

      if (left.earliestMs === right.earliestMs) {
        return left.label.localeCompare(right.label);
      }

      return left.earliestMs - right.earliestMs;
    });

  return {
    lanes: laneEntries,
    totalAttempts: laneEntries.reduce(
      (total, lane) => total + lane.bars.length,
      0,
    ),
    approximateCount,
    syntheticCount,
    startMs: earliestMs,
    endMs: latestMs,
    spanMs,
    chartWidthPx: timelineWidthPx,
    ticks: buildTicks(spanMs),
    maxLaneRowCount: laneEntries.reduce(
      (maximum, lane) => Math.max(maximum, lane.rowCount),
      0,
    ),
    statusCounts,
  };
}

function metricValue(value: number): string {
  return Intl.NumberFormat().format(value);
}

function TimelineMetric({
  label,
  value,
  detail,
}: {
  label: string;
  value: string;
  detail: string;
}) {
  return (
    <Card className="border border-(--stitch-outline) shadow-sm">
      <CardContent className="space-y-1 py-5">
        <p className="text-[11px] font-semibold uppercase tracking-[0.16em] text-(--stitch-on-surface-subtle)">
          {label}
        </p>
        <p className="text-2xl font-semibold text-(--stitch-on-surface)">
          {value}
        </p>
        <p className="text-sm text-(--stitch-on-surface-muted)">{detail}</p>
      </CardContent>
    </Card>
  );
}

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
        });
      } catch (err) {
        if (shouldIgnore()) {
          return;
        }

        console.error("Error fetching run timeline:", err);
        setError(
          err instanceof Error ? err.message : "Failed to fetch run timeline",
        );
      } finally {
        if (!silent && !shouldIgnore()) {
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

  return (
    <div className="space-y-6 pb-8 animate-in fade-in duration-300">
      <div className="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
        <div className="flex items-center gap-4">
          <Link
            to={`/runs/${runId}`}
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
            to={`/runs/${runId}`}
            className="inline-flex items-center gap-2 rounded-lg border border-(--stitch-outline) bg-(--stitch-surface-card) px-4 py-2 text-(--stitch-on-surface) transition-colors hover:bg-(--stitch-surface-low) focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-(--stitch-primary) focus-visible:ring-offset-2 focus-visible:ring-offset-(--stitch-background)"
          >
            <ArrowLeft className="h-4 w-4" />
            <span className="font-medium">Run Overview</span>
          </Link>
          <Link
            to={`/runs/${runId}/map`}
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

      <Card className="border border-(--stitch-outline) shadow-sm">
        <CardContent className="space-y-4 py-5">
          <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
            <div className="space-y-1">
              <h2 className="text-lg font-semibold text-(--stitch-on-surface)">
                Attempt Status Breakdown
              </h2>
              <p className="text-sm text-(--stitch-on-surface-muted)">
                Click any bar in the chart to open the underlying test attempt
                details.
              </p>
            </div>

            {(timeline.approximateCount > 0 || timeline.syntheticCount > 0) && (
              <div className="rounded-xl border border-(--status-warning-border) bg-(--status-warning-soft) px-4 py-3 text-sm text-(--status-warning)">
                {timeline.approximateCount > 0 && (
                  <p>
                    {metricValue(timeline.approximateCount)} attempt
                    {timeline.approximateCount === 1 ? " used" : "s used"}{" "}
                    inferred start or end bounds.
                  </p>
                )}
                {timeline.syntheticCount > 0 && (
                  <p>
                    {metricValue(timeline.syntheticCount)} legacy test
                    {timeline.syntheticCount === 1 ? " was" : "s were"} plotted
                    from aggregated test timing because attempt rows were
                    missing.
                  </p>
                )}
              </div>
            )}
          </div>

          <div className="flex flex-wrap gap-2">
            {orderedStatuses.map((status) => (
              <span
                key={status}
                className="inline-flex items-center gap-2 rounded-full border px-3 py-1.5 text-sm font-medium"
                style={statusStyles(status)}
              >
                <span className="capitalize">
                  {status.toLowerCase().replace("_", " ")}
                </span>
                <span className="rounded-full bg-black/5 px-2 py-0.5 text-xs font-semibold text-current">
                  {metricValue(timeline.statusCounts[status] ?? 0)}
                </span>
              </span>
            ))}
          </div>
        </CardContent>
      </Card>

      <Card className="overflow-hidden border border-(--stitch-outline) shadow-sm">
        <CardContent className="p-0">
          <div className="border-b border-(--stitch-outline) bg-(--stitch-surface-low) px-6 py-4">
            <div className="flex flex-col gap-2 lg:flex-row lg:items-center lg:justify-between">
              <div>
                <h2 className="text-lg font-semibold text-(--stitch-on-surface)">
                  Execution Timeline
                </h2>
                <p className="mt-1 text-sm text-(--stitch-on-surface-muted)">
                  Horizontal position maps to wall-clock time. Overlapping
                  attempts within the same execution are stacked into additional
                  rows.
                </p>
              </div>
              <div className="flex flex-wrap items-center gap-2 text-xs text-(--stitch-on-surface-muted)">
                <span className="rounded-full bg-(--stitch-surface-card) px-3 py-1.5">
                  Start {new Date(timeline.startMs).toLocaleString()}
                </span>
                <span className="rounded-full bg-(--stitch-surface-card) px-3 py-1.5">
                  End {new Date(timeline.endMs).toLocaleString()}
                </span>
                <span className="rounded-full bg-(--stitch-surface-card) px-3 py-1.5">
                  Width {timeline.chartWidthPx}px
                </span>
              </div>
            </div>
          </div>

          <div className="overflow-x-auto">
            <div
              className="min-w-full"
              style={{ width: LABEL_COLUMN_WIDTH_PX + timeline.chartWidthPx }}
            >
              <div className="sticky top-0 z-20 flex border-b border-(--stitch-outline) bg-(--stitch-surface-card)/95 backdrop-blur supports-[backdrop-filter]:bg-(--stitch-surface-card)/85">
                <div
                  className="sticky left-0 z-30 shrink-0 border-r border-(--stitch-outline) bg-(--stitch-surface-card)/95 px-4 py-4 backdrop-blur supports-[backdrop-filter]:bg-(--stitch-surface-card)/90"
                  style={{ width: LABEL_COLUMN_WIDTH_PX }}
                >
                  <p className="text-[11px] font-semibold uppercase tracking-[0.16em] text-(--stitch-on-surface-subtle)">
                    Execution Lane
                  </p>
                  <p className="mt-1 text-sm text-(--stitch-on-surface-muted)">
                    Grouped by execution id
                  </p>
                </div>

                <div
                  className="relative h-20 shrink-0 bg-(--stitch-surface-low)"
                  style={{ width: timeline.chartWidthPx }}
                >
                  {timeline.ticks.map((offsetMs) => {
                    const leftPx =
                      (offsetMs / timeline.spanMs) * timeline.chartWidthPx;
                    return (
                      <div
                        key={offsetMs}
                        className="absolute inset-y-0"
                        style={{ left: leftPx }}
                      >
                        <div className="absolute inset-y-0 w-px bg-(--stitch-outline)" />
                        <div className="absolute left-2 top-3 min-w-28">
                          <p className="text-xs font-semibold text-(--stitch-on-surface)">
                            {formatAxisTimestamp(
                              timeline.startMs + offsetMs,
                              timeline.spanMs,
                            )}
                          </p>
                          <p className="text-[11px] text-(--stitch-on-surface-subtle)">
                            {formatRelativeOffset(offsetMs)}
                          </p>
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>

              <div className="divide-y divide-(--stitch-outline)">
                {timeline.lanes.map((lane) => {
                  const laneHeightPx =
                    lane.rowCount * ROW_HEIGHT_PX + ROW_VERTICAL_PADDING_PX * 2;

                  return (
                    <div
                      key={lane.key}
                      className="flex bg-(--stitch-surface-card)"
                    >
                      <div
                        className="sticky left-0 z-10 shrink-0 border-r border-(--stitch-outline) bg-(--stitch-surface-card)/95 px-4 py-4 backdrop-blur supports-[backdrop-filter]:bg-(--stitch-surface-card)/90"
                        style={{ width: LABEL_COLUMN_WIDTH_PX }}
                      >
                        <div className="space-y-2">
                          <div className="flex flex-wrap items-center gap-2">
                            <h3 className="text-sm font-semibold text-(--stitch-on-surface)">
                              {lane.label}
                            </h3>
                            {lane.status && <Badge status={lane.status} />}
                          </div>
                          <p className="truncate font-mono text-xs text-(--stitch-on-surface-subtle)">
                            {lane.subtitle}
                          </p>
                          <div className="flex flex-wrap gap-2 text-xs text-(--stitch-on-surface-muted)">
                            <span className="rounded-full bg-(--stitch-surface-low) px-2.5 py-1 font-medium text-(--stitch-on-surface)">
                              {metricValue(lane.bars.length)} attempt
                              {lane.bars.length === 1 ? "" : "s"}
                            </span>
                            <span className="rounded-full bg-(--stitch-surface-low) px-2.5 py-1 font-medium">
                              {metricValue(lane.rowCount)} row
                              {lane.rowCount === 1 ? "" : "s"}
                            </span>
                            <span className="rounded-full bg-(--stitch-surface-low) px-2.5 py-1 font-medium">
                              {humanizeMilliseconds(lane.durationMs)}
                            </span>
                          </div>
                        </div>
                      </div>

                      <div
                        className="relative shrink-0 bg-(--stitch-surface-card)"
                        style={{
                          width: timeline.chartWidthPx,
                          height: laneHeightPx,
                        }}
                      >
                        {timeline.ticks.map((offsetMs) => {
                          const leftPx =
                            (offsetMs / timeline.spanMs) *
                            timeline.chartWidthPx;
                          return (
                            <div
                              key={`${lane.key}:${offsetMs}`}
                              className="pointer-events-none absolute inset-y-0 w-px bg-(--stitch-outline)"
                              style={{ left: leftPx }}
                              aria-hidden="true"
                            />
                          );
                        })}

                        {Array.from(
                          { length: lane.rowCount },
                          (_, rowIndex) => (
                            <div
                              key={`${lane.key}:row:${rowIndex}`}
                              className={cn(
                                "pointer-events-none absolute left-0 right-0 rounded-lg",
                                rowIndex % 2 === 0
                                  ? "bg-(--stitch-surface-low)/65"
                                  : "bg-(--stitch-surface-low)/30",
                              )}
                              style={{
                                top:
                                  ROW_VERTICAL_PADDING_PX +
                                  rowIndex * ROW_HEIGHT_PX,
                                height: ROW_HEIGHT_PX - 10,
                              }}
                              aria-hidden="true"
                            />
                          ),
                        )}

                        {lane.bars.map((bar) => {
                          const leftPx =
                            ((bar.startMs - timeline.startMs) /
                              timeline.spanMs) *
                            timeline.chartWidthPx;
                          const widthPx = Math.max(
                            (bar.durationMs / timeline.spanMs) *
                              timeline.chartWidthPx,
                            18,
                          );
                          const compact = widthPx < 168;
                          const tiny = widthPx < 112;
                          const barStyles = statusStyles(bar.status);

                          return (
                            <Link
                              key={bar.key}
                              to={`/runs/${runId}/tests/${bar.testId}`}
                              className={cn(
                                "absolute overflow-hidden rounded-xl border px-3 py-2 shadow-[0_6px_18px_rgba(15,23,42,0.08)] transition-all hover:-translate-y-0.5 hover:shadow-[0_12px_24px_rgba(15,23,42,0.14)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-(--stitch-primary) focus-visible:ring-offset-2 focus-visible:ring-offset-(--stitch-background)",
                                bar.status === "RUNNING" && "animate-pulse",
                              )}
                              style={{
                                left: leftPx,
                                top:
                                  ROW_VERTICAL_PADDING_PX +
                                  bar.rowIndex * ROW_HEIGHT_PX,
                                width: widthPx,
                                height: ROW_HEIGHT_PX - 10,
                                backgroundColor: barStyles.backgroundColor,
                                borderColor: barStyles.borderColor,
                                color: barStyles.color,
                              }}
                              title={`${bar.testTitle}\nAttempt ${bar.attemptIndex + 1}\n${formatAxisTimestamp(
                                bar.startMs,
                                timeline.spanMs,
                              )} -> ${formatAxisTimestamp(bar.endMs, timeline.spanMs)}\n${humanizeMilliseconds(
                                bar.durationMs,
                              )}`}
                            >
                              <div className="flex h-full min-w-0 flex-col justify-between gap-2">
                                <div className="flex min-w-0 items-center gap-2">
                                  <span className="inline-flex shrink-0 items-center rounded-full bg-black/8 px-2 py-0.5 text-[11px] font-semibold text-current">
                                    #{bar.attemptIndex + 1}
                                  </span>
                                  {!tiny && (
                                    <span className="truncate text-xs font-medium uppercase tracking-[0.12em] text-current/80">
                                      {bar.status
                                        .toLowerCase()
                                        .replace("_", " ")}
                                    </span>
                                  )}
                                </div>

                                <div className="min-w-0">
                                  <p className="truncate text-sm font-semibold text-current">
                                    {bar.testTitle}
                                  </p>
                                  {!tiny && (
                                    <p className="truncate text-xs text-current/80">
                                      {compact
                                        ? humanizeMilliseconds(bar.durationMs)
                                        : `${formatRelativeOffset(
                                            bar.startMs - timeline.startMs,
                                          )} • ${humanizeMilliseconds(bar.durationMs)}`}
                                    </p>
                                  )}
                                </div>
                              </div>
                            </Link>
                          );
                        })}
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card className="border border-(--stitch-outline) shadow-sm">
        <CardContent className="py-5">
          <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
            <div>
              <h2 className="text-lg font-semibold text-(--stitch-on-surface)">
                Run Window
              </h2>
              <p className="mt-1 text-sm text-(--stitch-on-surface-muted)">
                The overall run currently spans{" "}
                {humanizeMilliseconds(timeline.spanMs)} across{" "}
                {metricValue(timeline.totalAttempts)} attempt windows.
              </p>
            </div>
            {typeof runDetail.duration === "number" &&
              runDetail.duration > 0 && (
                <div className="rounded-xl border border-(--stitch-outline) bg-(--stitch-surface-low) px-4 py-3 text-sm text-(--stitch-on-surface-muted)">
                  Run duration field:{" "}
                  <span className="font-semibold text-(--stitch-on-surface)">
                    {humanizeDuration(runDetail.duration, 1_000_000_000)}
                  </span>
                </div>
              )}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
