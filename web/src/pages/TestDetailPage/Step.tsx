import { Card, CardContent } from "@/components/Card";
import { TagList } from "@/components/TagList";
import { useState, useEffect } from "react";
import {
  ChevronRight,
  ChevronDown,
  Clock3,
  Layers3,
  MapPin,
} from "lucide-react";
import type { Step as StepType } from "@/types/testCase";
import { Badge } from "@/components/Badge";
import { ansiToHtml } from "@/utils/ansi";
import {
  countNestedSteps,
  formatCompactDuration,
  formatDuration,
  formatStepLocation,
  getDurationFromRangeNs,
  parseTimestampMs,
  type StepTimelineContext,
} from "./utils";

type StepProps = {
  step: StepType;
  globalExpandAll?: boolean;
  depth?: number;
  timelineContext?: StepTimelineContext;
};

export const Step = ({
  step,
  globalExpandAll,
  depth = 0,
  timelineContext,
}: StepProps) => {
  const [isExpanded, setIsExpanded] = useState(globalExpandAll ?? false);
  const hasChildren = step.steps && step.steps.length > 0;
  const hasError = step.error || (step.errors && step.errors.length > 0);
  const shouldShowError =
    hasError &&
    (step.status === "FAILED" ||
      step.status === "BROKEN" ||
      step.status === "TIMEDOUT");

  // Extract error metadata from step.metadata
  const errorStack = step.metadata?.error_stack as string | undefined;
  const errorSnippet = step.metadata?.error_snippet as string | undefined;
  const errorLocation = step.metadata?.error_location as string | undefined;
  const errorValue = step.metadata?.error_value as string | undefined;
  const nestedStepCount = countNestedSteps(step.steps);
  const locationLabel = formatStepLocation(step.location);
  const timelineMetrics = getTimelineMetrics(step, timelineContext);
  const surfaceClass =
    depth === 0
      ? "bg-(--stitch-surface-card)"
      : depth === 1
        ? "bg-(--stitch-surface-low)"
        : "bg-(--stitch-surface-highest)";
  const categoryLabel = step.category || step.type;

  // Update local state when global state changes
  useEffect(() => {
    setIsExpanded(globalExpandAll ?? false);
  }, [globalExpandAll]);

  return (
    <div className={depth > 0 ? "relative pl-4" : undefined}>
      {depth > 0 && (
        <div
          aria-hidden="true"
          className="absolute bottom-0 left-1.5 top-0 w-px bg-(--stitch-outline)"
        />
      )}
      <Card className={`mb-2 rounded-lg ${surfaceClass}`}>
        <CardContent className="px-4 py-3">
          <div className="flex gap-2.5">
            <div className="pt-0.5">
              {hasChildren ? (
                <button
                  onClick={() => setIsExpanded(!isExpanded)}
                  className="flex h-7 w-7 items-center justify-center rounded-full bg-(--stitch-surface-card) text-(--stitch-on-surface-muted) transition-colors hover:bg-(--stitch-surface-highest)"
                  aria-label={
                    isExpanded ? "Collapse substeps" : "Expand substeps"
                  }
                >
                  {isExpanded ? (
                    <ChevronDown className="h-3.5 w-3.5" />
                  ) : (
                    <ChevronRight className="h-3.5 w-3.5" />
                  )}
                </button>
              ) : (
                <div className="flex h-7 w-7 items-center justify-center">
                  <div className="h-2 w-2 rounded-full bg-(--stitch-surface-highest)" />
                </div>
              )}
            </div>

            <div className="min-w-0 flex-1">
              <div className="flex flex-wrap items-center gap-1.5">
                <Badge
                  status={step.status || "UNKNOWN"}
                  showIcon={false}
                  className="px-2 py-0 text-[11px]"
                />
                {categoryLabel && (
                  <span className="inline-flex items-center rounded-full bg-(--stitch-surface-card) px-2 py-0.5 text-[11px] font-medium text-(--stitch-on-surface-muted)">
                    {categoryLabel}
                  </span>
                )}
                {step.duration !== undefined && step.duration > 0 && (
                  <span className="inline-flex items-center gap-1 rounded-full bg-(--stitch-surface-card) px-2 py-0.5 text-[11px] font-medium text-(--stitch-on-surface-muted)">
                    <Clock3 className="h-3 w-3" />
                    {formatDuration(step.duration)}
                  </span>
                )}
                {hasChildren && (
                  <span className="inline-flex items-center gap-1 rounded-full bg-(--stitch-surface-card) px-2 py-0.5 text-[11px] font-medium text-(--stitch-on-surface-muted)">
                    <Layers3 className="h-3 w-3" />
                    {nestedStepCount} nested
                  </span>
                )}
              </div>

              {timelineMetrics && (
                <div
                  className="mt-1.5 flex items-center gap-2 text-[10px] text-(--stitch-on-surface-subtle)"
                  title={`${timelineMetrics.startLabel} into ${timelineMetrics.totalLabel} total, duration ${timelineMetrics.durationLabel}`}
                >
                  <span className="w-10 shrink-0 text-right font-mono">
                    {timelineMetrics.startLabel}
                  </span>
                  <div className="relative h-1.5 min-w-0 flex-1 overflow-hidden rounded-full bg-(--stitch-surface-highest)">
                    <div
                      className="absolute inset-y-0 rounded-full bg-(--stitch-primary)"
                      style={{
                        left: `${timelineMetrics.offsetPercent}%`,
                        width: `${timelineMetrics.widthPercent}%`,
                      }}
                    />
                  </div>
                  <span className="w-10 shrink-0 font-mono">
                    {timelineMetrics.totalLabel}
                  </span>
                  <span className="shrink-0 rounded-full bg-(--stitch-surface-card) px-1.5 py-0.5 font-mono text-(--stitch-on-surface-muted)">
                    {timelineMetrics.durationLabel}
                  </span>
                </div>
              )}

              <div className="mt-2 flex items-start justify-between gap-2">
                <div className="min-w-0 flex-1">
                  <h3 className="text-[15px] font-semibold leading-5 text-(--stitch-on-surface) wrap-break-word">
                    {step.title || step.id}
                  </h3>
                  {locationLabel && (
                    <p className="mt-0.5 inline-flex items-center gap-1 text-[11px] text-(--stitch-on-surface-subtle)">
                      <MapPin className="h-3 w-3" />
                      <span className="font-mono">{locationLabel}</span>
                    </p>
                  )}
                </div>
              </div>

              {step.description && (
                <p className="mt-1.5 text-xs text-(--stitch-on-surface-muted) wrap-break-word">
                  {step.description}
                </p>
              )}

              {step.tags && step.tags.length > 0 && (
                <div className="mt-2">
                  <TagList tags={step.tags} />
                </div>
              )}

              {shouldShowError && (
                <div className="mt-4 p-3 bg-(--status-failure-soft) border border-(--status-failure-border) rounded space-y-2">
                  <p className="text-sm font-semibold text-(--status-failure)">
                    Error
                  </p>

                  {/* Error Message */}
                  <div>
                    <div
                      className="text-sm text-(--status-failure) whitespace-pre-wrap wrap-break-word"
                      dangerouslySetInnerHTML={{
                        __html: ansiToHtml(
                          step.error || step.errors?.[0] || "Unknown error",
                        ),
                      }}
                    />
                  </div>

                  {/* Error Value (if different from message) */}
                  {errorValue && errorValue !== step.error && (
                    <div>
                      <p className="text-xs font-medium text-(--status-failure) mb-1">
                        Value:
                      </p>
                      <div
                        className="text-xs text-(--status-failure) whitespace-pre-wrap wrap-break-word"
                        dangerouslySetInnerHTML={{
                          __html: ansiToHtml(errorValue),
                        }}
                      />
                    </div>
                  )}

                  {/* Error Location */}
                  {errorLocation && (
                    <div>
                      <p className="text-xs font-medium text-(--status-failure) mb-1">
                        Location:
                      </p>
                      <p className="text-xs text-(--status-failure) font-mono">
                        {errorLocation}
                      </p>
                    </div>
                  )}

                  {/* Code Snippet */}
                  {errorSnippet && (
                    <div>
                      <p className="text-xs font-medium text-(--status-failure) mb-1">
                        Code Snippet:
                      </p>
                      <pre
                        className="text-xs text-(--status-failure) bg-(--status-failure-soft) p-2 rounded overflow-x-auto"
                        dangerouslySetInnerHTML={{
                          __html: ansiToHtml(errorSnippet),
                        }}
                      />
                    </div>
                  )}

                  {/* Stack Trace */}
                  {errorStack && (
                    <details className="mt-2">
                      <summary className="text-xs font-medium text-(--status-failure) cursor-pointer hover:text-(--status-failure)">
                        Stack Trace
                      </summary>
                      <pre
                        className="text-xs text-(--status-failure) bg-(--status-failure-soft) p-2 rounded overflow-x-auto mt-1 whitespace-pre-wrap"
                        dangerouslySetInnerHTML={{
                          __html: ansiToHtml(errorStack),
                        }}
                      />
                    </details>
                  )}
                </div>
              )}
            </div>
          </div>
        </CardContent>
      </Card>
      {hasChildren && isExpanded && (
        <div className="ml-3 space-y-2">
          {step.steps?.map((subStep) => (
            <Step
              key={subStep.id}
              step={subStep}
              globalExpandAll={globalExpandAll}
              timelineContext={timelineContext}
              depth={depth + 1}
            />
          ))}
        </div>
      )}
    </div>
  );
};

function getTimelineMetrics(
  step: StepType,
  timelineContext?: StepTimelineContext,
) {
  if (!timelineContext) return undefined;

  const stepStartMs = parseTimestampMs(step.startTime || step.createdAt);
  if (stepStartMs === undefined) return undefined;

  const stepDurationNs =
    step.duration ??
    getDurationFromRangeNs(step.startTime, step.updatedAt) ??
    0;

  const relativeStartNs = Math.max(
    (stepStartMs - timelineContext.startTimeMs) * 1000000,
    0,
  );
  const boundedStartNs = Math.min(
    relativeStartNs,
    timelineContext.totalDurationNs,
  );

  const availableDurationNs = Math.max(
    timelineContext.totalDurationNs - boundedStartNs,
    0,
  );
  const boundedDurationNs = Math.min(stepDurationNs, availableDurationNs);

  const rawOffsetPercent =
    (boundedStartNs / timelineContext.totalDurationNs) * 100;
  const rawWidthPercent =
    (boundedDurationNs / timelineContext.totalDurationNs) * 100;
  const widthPercent = Math.min(
    Math.max(rawWidthPercent, 1.5),
    Math.max(100 - rawOffsetPercent, 1.5),
  );

  return {
    startLabel: formatCompactDuration(boundedStartNs) || "0ms",
    durationLabel: formatCompactDuration(stepDurationNs) || "0ms",
    totalLabel: timelineContext.totalDurationLabel,
    offsetPercent: rawOffsetPercent,
    widthPercent,
  };
}
