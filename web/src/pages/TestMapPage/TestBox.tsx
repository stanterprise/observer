import { memo, useEffect, useMemo, useRef, useState } from "react";
import { createPortal } from "react-dom";
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

type TooltipContent = {
  title: string;
  status: string;
  description?: string;
  duration?: string;
  retries?: number;
  startedAt?: string;
  tags: string[];
};

type TooltipPosition = {
  left: number;
  top: number;
  placement: "top" | "bottom";
};

const TOOLTIP_DELAY_MS = 650;
const TOOLTIP_WIDTH = 320;
const TOOLTIP_VIEWPORT_PADDING = 12;
const TOOLTIP_OFFSET = 10;

const getTooltipContent = (test: Test): TooltipContent => {
  const duration = formatDuration(test.duration);
  const startedAt = formatStartedAt(test.createdAt ?? test.startTime);
  const retries = (test.attempts?.length ?? 1) - 1;

  return {
    title: test.title || test.id,
    status: getStatusLabel(test),
    description: test.description,
    duration: duration ?? undefined,
    retries: retries > 0 ? retries : undefined,
    startedAt: startedAt ?? undefined,
    tags: test.tags ?? [],
  };
};

const getTooltipPosition = (rect: DOMRect): TooltipPosition => {
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
  const buttonRef = useRef<HTMLButtonElement | null>(null);
  const hoverTimerRef = useRef<number | null>(null);
  const [tooltipPosition, setTooltipPosition] =
    useState<TooltipPosition | null>(null);
  const accentColor = useMemo(() => getTestAccentColor(test), [test]);
  const tooltipText = useMemo(() => buildTooltipText(test), [test]);
  const tooltipContent = useMemo(() => getTooltipContent(test), [test]);
  const isMicro = width <= 8 || height <= 8;
  const isTiny = width <= 14 || height <= 12;
  const borderRadius = isMicro
    ? "2px"
    : isTiny
      ? "3px"
      : density === "comfortable"
        ? "6px"
        : "4px";

  const clearHoverTimer = () => {
    if (hoverTimerRef.current !== null) {
      window.clearTimeout(hoverTimerRef.current);
      hoverTimerRef.current = null;
    }
  };

  const showTooltip = () => {
    if (!buttonRef.current) {
      return;
    }

    setTooltipPosition(
      getTooltipPosition(buttonRef.current.getBoundingClientRect()),
    );
  };

  const hideTooltip = () => {
    clearHoverTimer();
    setTooltipPosition(null);
  };

  const handlePointerEnter = () => {
    clearHoverTimer();
    hoverTimerRef.current = window.setTimeout(() => {
      showTooltip();
      hoverTimerRef.current = null;
    }, TOOLTIP_DELAY_MS);
  };

  const handlePointerLeave = () => {
    hideTooltip();
  };

  const handleFocus = () => {
    clearHoverTimer();
    showTooltip();
  };

  const handleBlur = () => {
    hideTooltip();
  };

  useEffect(() => {
    if (!tooltipPosition) {
      return;
    }

    const hide = () => {
      setTooltipPosition(null);
    };

    window.addEventListener("scroll", hide, true);
    window.addEventListener("resize", hide);

    return () => {
      window.removeEventListener("scroll", hide, true);
      window.removeEventListener("resize", hide);
    };
  }, [tooltipPosition]);

  useEffect(() => {
    return () => {
      clearHoverTimer();
    };
  }, []);

  const tooltip =
    tooltipPosition && typeof document !== "undefined"
      ? createPortal(
          <div
            className="pointer-events-none fixed z-[90] w-80 rounded-xl border border-[var(--stitch-outline)] bg-[var(--stitch-surface-lowest)] px-4 py-3 text-sm text-[var(--stitch-on-surface)] shadow-2xl shadow-black/10"
            style={{
              left: `${tooltipPosition.left}px`,
              top: `${tooltipPosition.top}px`,
              transform:
                tooltipPosition.placement === "top"
                  ? "translateY(-100%)"
                  : undefined,
              backgroundColor: "var(--stitch-surface-card)",
              borderColor: "var(--stitch-outline)",
              color: "var(--stitch-on-surface)",
              backdropFilter: "blur(14px)",
              WebkitBackdropFilter: "blur(14px)",
            }}
            role="tooltip"
          >
            <div
              className="absolute left-1/2 h-3 w-3 -translate-x-1/2 rotate-45 border-[var(--stitch-outline)] bg-[var(--stitch-surface-lowest)]"
              style={{
                top:
                  tooltipPosition.placement === "top"
                    ? "calc(100% - 6px)"
                    : "-6px",
                borderWidth:
                  tooltipPosition.placement === "top"
                    ? "0 1px 1px 0"
                    : "1px 0 0 1px",
                borderStyle: "solid",
                backgroundColor: "var(--stitch-surface-card)",
                borderColor: "var(--stitch-outline)",
              }}
            />
            <div className="relative space-y-3">
              <div className="space-y-1">
                <div className="flex items-center gap-2">
                  <span
                    className="h-2.5 w-2.5 rounded-full"
                    style={{ backgroundColor: accentColor }}
                  />
                  <span className="text-[11px] font-semibold uppercase tracking-[0.18em] text-[var(--stitch-on-surface-muted)]">
                    {tooltipContent.status}
                  </span>
                </div>
                <p className="text-sm font-semibold leading-5 text-[var(--stitch-on-surface)]">
                  {tooltipContent.title}
                </p>
                {tooltipContent.description && (
                  <p className="text-xs leading-5 text-[var(--stitch-on-surface-muted)]">
                    {tooltipContent.description}
                  </p>
                )}
              </div>
              <div className="grid grid-cols-2 gap-x-3 gap-y-2 text-xs text-[var(--stitch-on-surface-muted)]">
                {tooltipContent.duration && (
                  <div>
                    <span className="block text-[10px] font-semibold uppercase tracking-[0.12em] text-[var(--stitch-on-surface-subtle)]">
                      Duration
                    </span>
                    <span className="text-[var(--stitch-on-surface)]">
                      {tooltipContent.duration}
                    </span>
                  </div>
                )}
                {tooltipContent.retries && (
                  <div>
                    <span className="block text-[10px] font-semibold uppercase tracking-[0.12em] text-[var(--stitch-on-surface-subtle)]">
                      Retries
                    </span>
                    <span className="text-[var(--stitch-on-surface)]">
                      {tooltipContent.retries}
                    </span>
                  </div>
                )}
                {tooltipContent.startedAt && (
                  <div className="col-span-2">
                    <span className="block text-[10px] font-semibold uppercase tracking-[0.12em] text-[var(--stitch-on-surface-subtle)]">
                      Started
                    </span>
                    <span className="text-[var(--stitch-on-surface)]">
                      {tooltipContent.startedAt}
                    </span>
                  </div>
                )}
              </div>
              {tooltipContent.tags.length > 0 && (
                <div className="flex flex-wrap gap-1.5">
                  {tooltipContent.tags.slice(0, 6).map((tag) => (
                    <span
                      key={tag}
                      className="rounded-full bg-[var(--stitch-primary-soft)] px-2 py-1 text-[11px] font-medium text-[var(--stitch-primary)]"
                    >
                      {tag}
                    </span>
                  ))}
                  {tooltipContent.tags.length > 6 && (
                    <span className="rounded-full bg-[var(--stitch-surface-low)] px-2 py-1 text-[11px] font-medium text-[var(--stitch-on-surface-muted)]">
                      +{tooltipContent.tags.length - 6}
                    </span>
                  )}
                </div>
              )}
            </div>
          </div>,
          document.body,
        )
      : null;

  return (
    <>
      <button
        ref={buttonRef}
        type="button"
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
        onMouseEnter={handlePointerEnter}
        onMouseLeave={handlePointerLeave}
        onFocus={handleFocus}
        onBlur={handleBlur}
      />
      {tooltip}
    </>
  );
}

export default memo(TestBox);
