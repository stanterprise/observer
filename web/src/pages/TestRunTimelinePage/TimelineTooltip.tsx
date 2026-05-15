import { createPortal } from "react-dom";
import {
  compactDurationLabel,
  formatRelativeOffset,
  formatStatusLabel,
  formatTooltipTime,
  statusStyles,
} from "./timelineFormatting";
import type { TimelineTooltipState } from "./types";

type TimelineTooltipProps = {
  timelineStartMs: number;
  tooltipState: TimelineTooltipState | null;
};

export function TimelineTooltip({
  timelineStartMs,
  tooltipState,
}: TimelineTooltipProps) {
  if (!tooltipState || typeof document === "undefined") {
    return null;
  }

  return createPortal(
    <div
      className="pointer-events-none fixed z-90 w-80 rounded-xl border border-(--stitch-outline) bg-(--stitch-surface-lowest) px-4 py-3 text-sm text-(--stitch-on-surface) shadow-2xl shadow-black/10"
      style={{
        left: `${tooltipState.position.left}px`,
        top: `${tooltipState.position.top}px`,
        transform:
          tooltipState.position.placement === "top"
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
        className="absolute left-1/2 h-3 w-3 -translate-x-1/2 rotate-45 border-(--stitch-outline) bg-(--stitch-surface-lowest)"
        style={{
          top:
            tooltipState.position.placement === "top"
              ? "calc(100% - 6px)"
              : "-6px",
          borderWidth:
            tooltipState.position.placement === "top"
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
              style={{
                backgroundColor: statusStyles(tooltipState.bar.status).color,
              }}
            />
            <span className="text-[11px] font-semibold uppercase tracking-[0.18em] text-(--stitch-on-surface-muted)">
              {formatStatusLabel(tooltipState.bar.status)}
            </span>
            {tooltipState.bar.approximated && (
              <span className="rounded-full bg-(--status-warning-soft) px-2 py-0.5 text-[10px] font-medium text-(--status-warning)">
                Inferred
              </span>
            )}
          </div>
          <p className="text-sm font-semibold leading-5 text-(--stitch-on-surface)">
            {tooltipState.bar.testTitle}
          </p>
          <p className="text-xs leading-5 text-(--stitch-on-surface-muted)">
            Attempt {tooltipState.bar.attemptIndex + 1}
            {tooltipState.bar.executionId
              ? ` • ${tooltipState.bar.executionId}`
              : " • No execution id"}
          </p>
        </div>
        <div className="grid grid-cols-2 gap-x-3 gap-y-2 text-xs text-(--stitch-on-surface-muted)">
          <div>
            <span className="block text-[10px] font-semibold uppercase tracking-[0.12em] text-(--stitch-on-surface-subtle)">
              Duration
            </span>
            <span className="text-(--stitch-on-surface)">
              {compactDurationLabel(tooltipState.bar.durationMs)}
            </span>
          </div>
          <div>
            <span className="block text-[10px] font-semibold uppercase tracking-[0.12em] text-(--stitch-on-surface-subtle)">
              Offset
            </span>
            <span className="text-(--stitch-on-surface)">
              {formatRelativeOffset(tooltipState.bar.startMs - timelineStartMs)}
            </span>
          </div>
          <div>
            <span className="block text-[10px] font-semibold uppercase tracking-[0.12em] text-(--stitch-on-surface-subtle)">
              Started
            </span>
            <span className="text-(--stitch-on-surface)">
              {formatTooltipTime(tooltipState.bar.startMs)}
            </span>
          </div>
          <div>
            <span className="block text-[10px] font-semibold uppercase tracking-[0.12em] text-(--stitch-on-surface-subtle)">
              Finished
            </span>
            <span className="text-(--stitch-on-surface)">
              {formatTooltipTime(tooltipState.bar.endMs)}
            </span>
          </div>
        </div>
        {tooltipState.bar.tags.length > 0 && (
          <div className="flex flex-wrap gap-1.5">
            {tooltipState.bar.tags.slice(0, 6).map((tag) => (
              <span
                key={tag}
                className="rounded-full bg-(--stitch-primary-soft) px-2 py-1 text-[11px] font-medium text-(--stitch-primary)"
              >
                {tag}
              </span>
            ))}
            {tooltipState.bar.tags.length > 6 && (
              <span className="rounded-full bg-(--stitch-surface-low) px-2 py-1 text-[11px] font-medium text-(--stitch-on-surface-muted)">
                +{tooltipState.bar.tags.length - 6}
              </span>
            )}
          </div>
        )}
      </div>
    </div>,
    document.body,
  );
}
