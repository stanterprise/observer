import type { TestStatus } from "@/types/common";
import type { TimelineStatusStyles } from "./types";

export const LABEL_COLUMN_WIDTH_PX = 264;
export const TIMELINE_MIN_WIDTH_PX = 960;
export const TIMELINE_MAX_WIDTH_PX = 4200;
export const BAR_HEIGHT_PX = 16;
export const ROW_HEIGHT_PX = 24;
export const ROW_VERTICAL_PADDING_PX = 6;
export const BAR_HORIZONTAL_GAP_PX = 2;
export const TOOLTIP_DELAY_MS = 650;
export const TOOLTIP_WIDTH = 320;
export const TOOLTIP_VIEWPORT_PADDING = 12;
export const TOOLTIP_OFFSET = 10;

export const STATUS_ORDER: TestStatus[] = [
  "RUNNING",
  "FAILED",
  "BROKEN",
  "TIMEDOUT",
  "INTERRUPTED",
  "FLAKY",
  "PASSED",
  "SKIPPED",
  "PENDING",
  "NOT_RUN",
  "UNKNOWN",
];

export const NICE_TICK_INTERVALS_MS = [
  100, 250, 500, 1_000, 2_000, 5_000, 10_000, 15_000, 30_000, 60_000, 120_000,
  300_000, 600_000, 900_000, 1_800_000, 3_600_000, 7_200_000, 14_400_000,
  28_800_000, 43_200_000, 86_400_000,
];

export const DEFAULT_STATUS_STYLES: TimelineStatusStyles = {
  backgroundColor: "var(--status-neutral-soft)",
  borderColor: "var(--status-neutral-border)",
  color: "var(--status-neutral)",
};
