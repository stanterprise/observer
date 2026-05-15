import type { Attempt, Test } from "@/types/testCase";
import type { RunExecution, TestRun } from "@/types/testRun";
import {
  NICE_TICK_INTERVALS_MS,
  TIMELINE_MAX_WIDTH_PX,
  TIMELINE_MIN_WIDTH_PX,
  BAR_HORIZONTAL_GAP_PX,
} from "./constants";
import { normalizeStatus } from "./timelineFormatting";
import type {
  AttemptCandidate,
  TimelineBar,
  TimelineLane,
  TimelineModel,
} from "./types";

function toTimestamp(value?: string): number | null {
  if (!value) {
    return null;
  }

  const parsed = Date.parse(value);
  return Number.isNaN(parsed) ? null : parsed;
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

function minimumLaneBarWidthPx(barCount: number): number {
  if (barCount >= 800) {
    return 4;
  }

  if (barCount >= 300) {
    return 5;
  }

  if (barCount >= 120) {
    return 6;
  }

  return 8;
}

function measureBarPixels(
  bar: TimelineBar,
  timelineStartMs: number,
  spanMs: number,
  chartWidthPx: number,
  minWidthPx: number,
): { leftPx: number; widthPx: number } {
  const rawLeftPx = ((bar.startMs - timelineStartMs) / spanMs) * chartWidthPx;
  const rawWidthPx = (bar.durationMs / spanMs) * chartWidthPx;
  const widthPx = Math.max(rawWidthPx, minWidthPx);
  const leftPx = Math.min(rawLeftPx, Math.max(chartWidthPx - widthPx, 0));

  return {
    leftPx,
    widthPx,
  };
}

function packLaneRows(
  bars: TimelineBar[],
  timelineStartMs: number,
  spanMs: number,
  chartWidthPx: number,
): {
  bars: TimelineBar[];
  rowCount: number;
} {
  const minWidthPx = minimumLaneBarWidthPx(bars.length);
  const packedBars = bars
    .map((bar) => {
      const pixels = measureBarPixels(
        bar,
        timelineStartMs,
        spanMs,
        chartWidthPx,
        minWidthPx,
      );

      return {
        ...bar,
        renderLeftPx: pixels.leftPx,
        renderWidthPx: pixels.widthPx,
      };
    })
    .sort((left, right) => {
      if (left.renderLeftPx === right.renderLeftPx) {
        if (left.renderWidthPx === right.renderWidthPx) {
          if (left.startMs === right.startMs) {
            if (left.endMs === right.endMs) {
              return left.testTitle.localeCompare(right.testTitle);
            }
            return left.endMs - right.endMs;
          }
          return left.startMs - right.startMs;
        }
        return left.renderWidthPx - right.renderWidthPx;
      }
      return left.renderLeftPx - right.renderLeftPx;
    });

  const rowEndPixels: number[] = [];

  for (const bar of packedBars) {
    let rowIndex = rowEndPixels.findIndex(
      (endPx) => bar.renderLeftPx >= endPx + BAR_HORIZONTAL_GAP_PX,
    );

    if (rowIndex === -1) {
      rowIndex = rowEndPixels.length;
      rowEndPixels.push(bar.renderLeftPx + bar.renderWidthPx);
    } else {
      rowEndPixels[rowIndex] = bar.renderLeftPx + bar.renderWidthPx;
    }

    bar.rowIndex = rowIndex;
  }

  return {
    bars: packedBars,
    rowCount: Math.max(rowEndPixels.length, 1),
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

function isShardDisplayName(value?: string): boolean {
  if (!value) {
    return false;
  }

  return /^shard\s+\d+(?:\s+of\s+\d+)?$/i.test(value.trim());
}

function buildLaneLabel(
  execution: RunExecution | undefined,
  executionId: string,
  fallbackIndex: number,
): {
  label: string;
  subtitle: string;
  status?: TimelineLane["status"];
  statusLabel?: string;
} {
  if (execution?.name && !isShardDisplayName(execution.name) && !executionId) {
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

export function buildTimelineModel(
  runDetail: TestRun | null,
): TimelineModel | null {
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
  const statusCounts: TimelineModel["statusCounts"] = {};
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
        renderLeftPx: 0,
        renderWidthPx: 0,
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
  const chartWidthPx = Math.max(
    TIMELINE_MIN_WIDTH_PX,
    Math.min(TIMELINE_MAX_WIDTH_PX, Math.round((spanMs / 1_000) * 16)),
  );

  const lanes = Array.from(laneBars.entries())
    .map(([laneKey, bars], laneIndex) => {
      const executionId = laneKey === "__primary__" ? "" : laneKey;
      const execution = executionId
        ? executionById.get(executionId)
        : undefined;
      const packed = packLaneRows(bars, earliestMs, spanMs, chartWidthPx);
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
      } satisfies TimelineLane;
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
    lanes,
    totalAttempts: lanes.reduce((total, lane) => total + lane.bars.length, 0),
    approximateCount,
    syntheticCount,
    startMs: earliestMs,
    endMs: latestMs,
    spanMs,
    chartWidthPx,
    ticks: buildTicks(spanMs),
    maxLaneRowCount: lanes.reduce(
      (maximum, lane) => Math.max(maximum, lane.rowCount),
      0,
    ),
    statusCounts,
  };
}
