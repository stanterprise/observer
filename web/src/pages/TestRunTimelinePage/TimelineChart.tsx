import { useCallback, useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { Badge } from "@/components/Badge";
import { Card, CardContent } from "@/components/Card";
import { cn } from "@/lib/utils";
import {
  BAR_HEIGHT_PX,
  LABEL_COLUMN_WIDTH_PX,
  ROW_HEIGHT_PX,
  ROW_VERTICAL_PADDING_PX,
  TOOLTIP_DELAY_MS,
} from "./constants";
import {
  attemptAriaLabel,
  compactBarClassName,
  compactBarTopPx,
  compactDurationLabel,
  compactRowHeightPx,
  formatAxisTimestamp,
  formatRelativeOffset,
  getTooltipPosition,
  metricValue,
  statusStyles,
} from "./timelineFormatting";
import { TimelineTooltip } from "./TimelineTooltip";
import type { TimelineBar, TimelineModel, TimelineTooltipState } from "./types";

type TimelineChartProps = {
  runId: string;
  timeline: TimelineModel;
};

export function TimelineChart({ runId, timeline }: TimelineChartProps) {
  const hoverTimerRef = useRef<number | null>(null);
  const [tooltipState, setTooltipState] = useState<TimelineTooltipState | null>(
    null,
  );

  const clearHoverTimer = useCallback(() => {
    if (hoverTimerRef.current !== null) {
      window.clearTimeout(hoverTimerRef.current);
      hoverTimerRef.current = null;
    }
  }, []);

  const showTooltip = useCallback(
    (target: HTMLAnchorElement, bar: TimelineBar) => {
      setTooltipState({
        bar,
        position: getTooltipPosition(target.getBoundingClientRect()),
      });
    },
    [],
  );

  const hideTooltip = useCallback(() => {
    clearHoverTimer();
    setTooltipState(null);
  }, [clearHoverTimer]);

  const handleBarPointerEnter = useCallback(
    (target: HTMLAnchorElement, bar: TimelineBar) => {
      clearHoverTimer();
      hoverTimerRef.current = window.setTimeout(() => {
        showTooltip(target, bar);
        hoverTimerRef.current = null;
      }, TOOLTIP_DELAY_MS);
    },
    [clearHoverTimer, showTooltip],
  );

  const handleBarFocus = useCallback(
    (target: HTMLAnchorElement, bar: TimelineBar) => {
      clearHoverTimer();
      showTooltip(target, bar);
    },
    [clearHoverTimer, showTooltip],
  );

  useEffect(() => {
    if (!tooltipState) {
      return;
    }

    const hide = () => {
      setTooltipState(null);
    };

    window.addEventListener("scroll", hide, true);
    window.addEventListener("resize", hide);

    return () => {
      window.removeEventListener("scroll", hide, true);
      window.removeEventListener("resize", hide);
    };
  }, [tooltipState]);

  useEffect(() => {
    return () => {
      clearHoverTimer();
    };
  }, [clearHoverTimer]);

  return (
    <>
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
                  rows, and very short intervals are compacted so the visible
                  width stays closer to the real schedule.
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
                              {compactDurationLabel(lane.durationMs)}
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
                                top: compactBarTopPx(rowIndex),
                                height: compactRowHeightPx(),
                              }}
                              aria-hidden="true"
                            />
                          ),
                        )}

                        {lane.bars.map((bar) => {
                          const widthPx = bar.renderWidthPx;
                          const showIndex = widthPx >= 30;
                          const showTitle = widthPx >= 72;
                          const showMeta = widthPx >= 160;
                          const showChip = widthPx >= 120;
                          const barStyles = statusStyles(bar.status);

                          return (
                            <Link
                              key={bar.key}
                              to={`/runs/${runId}/tests/${bar.testId}`}
                              aria-label={attemptAriaLabel(bar)}
                              className={cn(
                                "absolute overflow-hidden border shadow-[0_2px_8px_rgba(15,23,42,0.08)] transition-all hover:-translate-y-0.5 hover:shadow-[0_8px_16px_rgba(15,23,42,0.14)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-(--stitch-primary) focus-visible:ring-offset-2 focus-visible:ring-offset-(--stitch-background)",
                                compactBarClassName(widthPx),
                                bar.status === "RUNNING" && "animate-pulse",
                              )}
                              style={{
                                left: bar.renderLeftPx,
                                top: compactBarTopPx(bar.rowIndex),
                                width: widthPx,
                                height: BAR_HEIGHT_PX,
                                backgroundColor: barStyles.backgroundColor,
                                borderColor: barStyles.borderColor,
                                color: barStyles.color,
                              }}
                              onMouseEnter={(event) =>
                                handleBarPointerEnter(event.currentTarget, bar)
                              }
                              onMouseLeave={hideTooltip}
                              onFocus={(event) =>
                                handleBarFocus(event.currentTarget, bar)
                              }
                              onBlur={hideTooltip}
                            >
                              <div className="flex h-full min-w-0 items-center gap-1.5">
                                {showIndex && (
                                  <span className="inline-flex shrink-0 items-center rounded-full bg-black/8 px-1.5 py-0 text-[10px] font-semibold leading-none text-current">
                                    {bar.attemptIndex + 1}
                                  </span>
                                )}

                                {showTitle && (
                                  <div className="min-w-0 flex-1">
                                    <div className="flex min-w-0 items-center gap-1.5">
                                      <p className="truncate text-[11px] font-semibold leading-none text-current">
                                        {bar.testTitle}
                                      </p>
                                      {showChip && (
                                        <span className="shrink-0 truncate rounded-full bg-black/8 px-1.5 py-0 text-[10px] font-medium uppercase tracking-[0.08em] text-current/85">
                                          {bar.status
                                            .toLowerCase()
                                            .replace("_", " ")}
                                        </span>
                                      )}
                                    </div>
                                    {showMeta && (
                                      <p className="mt-0.5 truncate text-[10px] leading-none text-current/80">
                                        {compactDurationLabel(bar.durationMs)} •{" "}
                                        {formatRelativeOffset(
                                          bar.startMs - timeline.startMs,
                                        )}
                                      </p>
                                    )}
                                  </div>
                                )}

                                {!showTitle && !showIndex && (
                                  <span className="sr-only">
                                    {attemptAriaLabel(bar)}
                                  </span>
                                )}
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

      <TimelineTooltip
        timelineStartMs={timeline.startMs}
        tooltipState={tooltipState}
      />
    </>
  );
}
