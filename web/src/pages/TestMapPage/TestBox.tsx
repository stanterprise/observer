import { memo, useMemo } from "react";
import { cn } from "@/lib/utils";
import type { Test } from "@/types/testCase";
import type { TestStatus } from "@/types/common";

// Helper to determine if a test is flaky (passed with retries)
const isFlaky = (test: Test): boolean => {
  return test.status === "PASSED" && (test.attempts?.length ?? 0) > 1;
};

const ACCENT_COLORS: Record<TestStatus | "FLAKY_RETRY", string> = {
  PASSED: "var(--status-success)",
  FLAKY: "var(--stitch-tertiary)",
  FLAKY_RETRY: "var(--stitch-tertiary)",
  FAILED: "var(--status-failure)",
  SKIPPED: "var(--status-neutral)",
  RUNNING: "var(--status-running)",
  PENDING: "var(--status-warning)",
  BROKEN: "var(--status-broken)",
  TIMEDOUT: "var(--status-timedout)",
  INTERRUPTED: "var(--status-interrupted)",
  NOT_RUN: "var(--status-neutral)",
  UNKNOWN: "var(--status-neutral)",
};

const getTestAccentColor = (test: Test): string => {
  if (isFlaky(test)) {
    return ACCENT_COLORS.FLAKY_RETRY;
  }
  return ACCENT_COLORS[test.status] ?? ACCENT_COLORS.UNKNOWN;
};

const formatDuration = (nanoseconds?: number): string | null => {
  if (!nanoseconds) {
    return null;
  }

  const milliseconds = nanoseconds / 1_000_000;
  if (milliseconds < 1000) {
    return `${milliseconds.toFixed(0)}ms`;
  }

  return `${(milliseconds / 1000).toFixed(2)}s`;
};

const formatStartedAt = (value?: string): string | null => {
  if (!value) {
    return null;
  }

  return new Date(value).toLocaleTimeString([], {
    hour: "numeric",
    minute: "2-digit",
  });
};

export type TestMapDensity = "comfortable" | "compact" | "dense" | "ultra";

const getStatusLabel = (test: Test): string => {
  if (isFlaky(test)) {
    return "Flaky";
  }

  return test.status
    .toLowerCase()
    .split("_")
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
};

const buildTooltipText = (test: Test): string => {
  const duration = formatDuration(test.duration);
  const startedAt = formatStartedAt(test.createdAt ?? test.startTime);
  const retryCount = Math.max(
    (test.attempts?.length ?? 0) - 1,
    test.retryCount ?? 0,
  );

  const lines = [test.title || test.id, `Status: ${getStatusLabel(test)}`];

  if (test.description) {
    lines.push(`Description: ${test.description}`);
  }
  if (duration) {
    lines.push(`Duration: ${duration}`);
  }
  if (retryCount > 0) {
    lines.push(`Retries: ${retryCount}`);
  }
  if (startedAt) {
    lines.push(`Started: ${startedAt}`);
  }
  if (test.tags && test.tags.length > 0) {
    lines.push(`Tags: ${test.tags.join(", ")}`);
  }

  return lines.join("\n");
};

interface TestBoxProps {
  test: Test;
  isHighlighted: boolean;
  isFaded: boolean;
  onClick: () => void;
  density: TestMapDensity;
  width: number;
  height: number;
}

function TestBox({
  test,
  isHighlighted,
  isFaded,
  onClick,
  density,
  width,
  height,
}: TestBoxProps) {
  const accentColor = useMemo(() => getTestAccentColor(test), [test]);
  const tooltipText = useMemo(() => buildTooltipText(test), [test]);
  const isMicro = width <= 8 || height <= 8;
  const isTiny = width <= 14 || height <= 12;
  const borderRadius = isMicro
    ? "2px"
    : isTiny
      ? "3px"
      : density === "comfortable"
        ? "6px"
        : "4px";

  return (
    <button
      type="button"
      title={tooltipText}
      aria-label={tooltipText}
      className={cn(
        "block transition-all duration-150",
        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--stitch-primary)] focus-visible:ring-offset-1 focus-visible:ring-offset-[var(--stitch-background)]",
        isHighlighted && "ring-1 ring-[var(--stitch-primary)]/80",
      )}
      style={{
        width: `${width}px`,
        height: `${height}px`,
        borderRadius,
        backgroundColor: accentColor,
        border: `1px solid ${accentColor}`,
        opacity: isFaded ? 0.22 : 0.9,
        boxShadow: isHighlighted
          ? "0 0 0 1px var(--stitch-primary), 0 0 0 2px color-mix(in srgb, var(--stitch-primary) 18%, transparent)"
          : undefined,
      }}
      onClick={onClick}
    />
  );
}

export default memo(TestBox);
