import { humanizeMilliseconds } from "@/utils/duration";
import type { TestStatus } from "@/types/common";
import {
  BAR_HEIGHT_PX,
  BAR_HORIZONTAL_GAP_PX,
  DEFAULT_STATUS_STYLES,
  ROW_HEIGHT_PX,
  ROW_VERTICAL_PADDING_PX,
  TOOLTIP_OFFSET,
  TOOLTIP_VIEWPORT_PADDING,
  TOOLTIP_WIDTH,
} from "./constants";
import type {
  TimelineBar,
  TimelineStatusStyles,
  TooltipPosition,
} from "./types";

export function normalizeStatus(status?: string): TestStatus {
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

export function statusStyles(status: TestStatus): TimelineStatusStyles {
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

export function compactDurationLabel(durationMs: number): string {
  if (durationMs < 1_000) {
    return `${Math.max(Math.round(durationMs), 1)}ms`;
  }

  if (durationMs < 60_000) {
    const seconds = durationMs / 1_000;
    return seconds >= 10 ? `${seconds.toFixed(0)}s` : `${seconds.toFixed(1)}s`;
  }

  return humanizeMilliseconds(durationMs);
}

export function compactRowHeightPx(): number {
  return ROW_HEIGHT_PX - BAR_HORIZONTAL_GAP_PX * 2;
}

export function compactBarTopPx(rowIndex: number): number {
  return (
    ROW_VERTICAL_PADDING_PX +
    rowIndex * ROW_HEIGHT_PX +
    Math.max((ROW_HEIGHT_PX - BAR_HEIGHT_PX) / 2, 0)
  );
}

export function compactBarClassName(widthPx: number): string {
  if (widthPx < 30) {
    return "rounded-md px-0 py-0";
  }

  if (widthPx < 72) {
    return "rounded-md px-1.5 py-0.5";
  }

  return "rounded-lg px-2 py-1";
}

export function formatStatusLabel(status: TestStatus): string {
  return status
    .toLowerCase()
    .split("_")
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}

export function attemptAriaLabel(bar: TimelineBar): string {
  return [
    bar.testTitle,
    `attempt ${bar.attemptIndex + 1}`,
    bar.status.toLowerCase().replace("_", " "),
    compactDurationLabel(bar.durationMs),
    bar.approximated ? "timing inferred" : null,
  ]
    .filter(Boolean)
    .join(", ");
}

export function formatTooltipTime(timestampMs: number): string {
  return new Date(timestampMs).toLocaleTimeString([], {
    hour: "numeric",
    minute: "2-digit",
    second: "2-digit",
  });
}

export function getTooltipPosition(rect: DOMRect): TooltipPosition {
  const left = Math.min(
    Math.max(
      rect.left + rect.width / 2 - TOOLTIP_WIDTH / 2,
      TOOLTIP_VIEWPORT_PADDING,
    ),
    window.innerWidth - TOOLTIP_WIDTH - TOOLTIP_VIEWPORT_PADDING,
  );
  const placement = rect.top < 180 ? "bottom" : "top";

  return {
    left,
    top:
      placement === "top"
        ? rect.top - TOOLTIP_OFFSET
        : rect.bottom + TOOLTIP_OFFSET,
    placement,
  };
}

export function formatAxisTimestamp(
  timestampMs: number,
  spanMs: number,
): string {
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

export function formatRelativeOffset(offsetMs: number): string {
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

export function metricValue(value: number): string {
  return Intl.NumberFormat().format(value);
}
