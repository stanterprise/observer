import { useState } from "react";
import { cn } from "@/lib/utils";
import type { Test } from "@/types/testCase";
import type { TestStatus } from "@/types/common";

// Helper to determine if a test is flaky (passed with retries)
const isFlaky = (test: Test): boolean => {
  return test.status === "PASSED" && (test.attempts?.length ?? 0) > 1;
};

// Helper to get color for test status including flaky
const getTestStatusColor = (test: Test): string => {
  if (isFlaky(test)) {
    return "bg-[var(--status-warning-soft)] border-[var(--status-warning-border)]";
  }

  const statusColors: Record<TestStatus, string> = {
    PASSED:
      "bg-[var(--status-success-soft)] border-[var(--status-success-border)]",
    FLAKY:
      "bg-[var(--status-warning-soft)] border-[var(--status-warning-border)]",
    FAILED:
      "bg-[var(--status-failure-soft)] border-[var(--status-failure-border)]",
    SKIPPED:
      "bg-[var(--status-neutral-soft)] border-[var(--status-neutral-border)]",
    RUNNING:
      "bg-[var(--status-running-soft)] border-[var(--status-running-border)]",
    PENDING:
      "bg-[var(--status-warning-soft)] border-[var(--status-warning-border)]",
    BROKEN:
      "bg-[var(--status-broken-soft)] border-[var(--status-broken-border)]",
    TIMEDOUT:
      "bg-[var(--status-timedout-soft)] border-[var(--status-timedout-border)]",
    INTERRUPTED:
      "bg-[var(--status-interrupted-soft)] border-[var(--status-interrupted-border)]",
    NOT_RUN:
      "bg-[var(--status-neutral-soft)] border-[var(--status-neutral-border)]",
    UNKNOWN:
      "bg-[var(--status-neutral-soft)] border-[var(--status-neutral-border)]",
  };

  return statusColors[test.status] || statusColors.UNKNOWN;
};

// Get human-readable status label
const getStatusLabel = (test: Test): string => {
  if (isFlaky(test)) {
    return "Flaky";
  }
  return (
    test.status.charAt(0) + test.status.slice(1).toLowerCase().replace("_", " ")
  );
};

interface TestBoxProps {
  test: Test;
  isHighlighted: boolean;
  isFaded: boolean; // New prop for fading non-highlighted tests
  onClick: () => void;
  width: number; // Width in pixels
  height: number; // Height in pixels
}

export default function TestBox({
  test,
  isHighlighted,
  isFaded,
  onClick,
  width,
  height,
}: TestBoxProps) {
  const [showTooltip, setShowTooltip] = useState(false);
  const colorClass = getTestStatusColor(test);

  // Scale border width based on height (1px for small, 2px for large)
  const borderWidth = height < 24 ? 1 : 2;

  return (
    <div className="relative">
      <div
        className={cn(
          "rounded cursor-pointer transition-all duration-200",
          colorClass,
          isHighlighted && "ring-2 ring-blue-300 scale-110",
          !isHighlighted && "hover:scale-105 hover:shadow-md",
        )}
        style={{
          width: `${width}px`,
          height: `${height}px`,
          borderWidth: `${borderWidth}px`,
          opacity: isFaded ? 0.25 : 1, // Apply fading when isFaded is true
        }}
        onClick={onClick}
        onKeyDown={(e) => {
          if (e.key === "Enter" || e.key === " ") {
            e.preventDefault();
            onClick();
          }
        }}
        onMouseEnter={() => setShowTooltip(true)}
        onMouseLeave={() => setShowTooltip(false)}
        role="button"
        tabIndex={0}
        aria-label={`Test: ${test.title}`}
      />
      {showTooltip && (
        <div className="absolute z-50 bottom-full left-1/2 -translate-x-1/2 mb-2 w-64 p-3 bg-[var(--stitch-surface-highest)] text-[var(--stitch-on-surface)] text-xs rounded-lg shadow-xl pointer-events-none">
          <div className="font-semibold mb-1 truncate">{test.title}</div>
          <div className="space-y-1 text-[var(--stitch-on-surface-muted)]">
            <div>
              Status:{" "}
              <span className="font-medium text-[var(--stitch-on-surface)]">
                {getStatusLabel(test)}
              </span>
            </div>
            {test.duration && (
              <div>Duration: {(test.duration / 1_000_000).toFixed(2)}ms</div>
            )}
            {test.tags && test.tags.length > 0 && (
              <div>Tags: {test.tags.join(", ")}</div>
            )}
            {isFlaky(test) && (
              <div className="text-[var(--status-warning)]">
                ⚠️ Passed with {test.attempts?.length ?? 0} attempts
              </div>
            )}
          </div>
          <div className="absolute top-full left-1/2 -translate-x-1/2 -mt-1">
            <div className="border-4 border-transparent border-t-[var(--stitch-surface-highest)]" />
          </div>
        </div>
      )}
    </div>
  );
}
