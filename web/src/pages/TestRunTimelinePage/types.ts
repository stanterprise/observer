import type { TestStatus } from "@/types/common";
import type { Attempt } from "@/types/testCase";

export type TimelineStatusStyles = {
  backgroundColor: string;
  borderColor: string;
  color: string;
};

export type TimelineBar = {
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
  renderLeftPx: number;
  renderWidthPx: number;
};

export type TimelineLane = {
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

export type TimelineModel = {
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

export type AttemptCandidate = {
  attempt: Attempt;
  synthetic: boolean;
};

export type TooltipPosition = {
  left: number;
  top: number;
  placement: "top" | "bottom";
};

export type TimelineTooltipState = {
  bar: TimelineBar;
  position: TooltipPosition;
};
