/**
 * Time conversion constants
 */
export const NS_TO_MS = 1_000_000;
export const MS_TO_S = 1_000;

/**
 * Format duration for display
 * @param ms Duration in milliseconds
 * @returns Formatted string (e.g., "1.5s", "750ms")
 */
export function formatDuration(ms: number): string {
  if (ms >= MS_TO_S) {
    return `${(ms / MS_TO_S).toFixed(1)}s`;
  }
  return `${Math.round(ms)}ms`;
}
