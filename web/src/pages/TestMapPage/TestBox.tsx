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
    return "bg-amber-400 border-amber-500"; // Flaky: amber/yellow
  }

  const statusColors: Record<TestStatus, string> = {
    PASSED: "bg-green-500 border-green-600",
    FAILED: "bg-red-500 border-red-600",
    SKIPPED: "bg-gray-400 border-gray-500",
    RUNNING: "bg-blue-500 border-blue-600",
    PENDING: "bg-yellow-400 border-yellow-500",
    BROKEN: "bg-orange-500 border-orange-600",
    TIMEDOUT: "bg-purple-500 border-purple-600",
    INTERRUPTED: "bg-pink-500 border-pink-600",
    NOT_RUN: "bg-gray-300 border-gray-400",
    UNKNOWN: "bg-gray-400 border-gray-500",
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
        onMouseEnter={() => setShowTooltip(true)}
        onMouseLeave={() => setShowTooltip(false)}
        role="button"
        tabIndex={0}
        aria-label={`Test: ${test.title}`}
      />
      {showTooltip && (
        <div className="absolute z-50 bottom-full left-1/2 -translate-x-1/2 mb-2 w-64 p-3 bg-gray-900 text-white text-xs rounded-lg shadow-xl pointer-events-none">
          <div className="font-semibold mb-1 truncate">{test.title}</div>
          <div className="space-y-1 text-gray-300">
            <div>
              Status:{" "}
              <span className="font-medium text-white">
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
              <div className="text-amber-300">
                ⚠️ Passed with {test.attempts!.length} attempts
              </div>
            )}
          </div>
          <div className="absolute top-full left-1/2 -translate-x-1/2 -mt-1">
            <div className="border-4 border-transparent border-t-gray-900" />
          </div>
        </div>
      )}
    </div>
  );
}
