/**
 * Attempt-based retry utilities for accessing test attempt data
 *
 * MongoDB document structure has been updated to support attempt-based retries.
 * Test data is now stored in attempts[retry_index] arrays, with legacy fields
 * maintained for backward compatibility.
 *
 * These utilities provide safe access to attempt-specific data with automatic
 * fallback to legacy fields when attempts array is not available.
 */

import type { Test, Attempt, Step } from "@/types/testCase";

/**
 * Get steps from the current attempt, falling back to legacy steps field
 *
 * @param test - Test document
 * @returns Array of steps from current attempt or legacy field
 */
export function getTestSteps(test: Test): Step[] {
  // Try to get steps from current attempt
  if (test.attempts && test.retryIndex !== undefined) {
    const currentAttempt = test.attempts[test.retryIndex];
    if (currentAttempt?.steps) {
      return currentAttempt.steps;
    }
  }

  // Fallback to legacy field for backward compatibility
  return test.steps || [];
}

/**
 * Get error message from the current attempt
 *
 * @param test - Test document
 * @returns Error message from current attempt or legacy field
 */
export function getTestErrorMessage(test: Test): string | undefined {
  if (test.attempts && test.retryIndex !== undefined) {
    const currentAttempt = test.attempts[test.retryIndex];
    if (currentAttempt) {
      return currentAttempt.errorMessage || undefined;
    }
  }
  return test.errorMessage || undefined;
}

/**
 * Get stack trace from the current attempt
 *
 * @param test - Test document
 * @returns Stack trace from current attempt or legacy field
 */
export function getTestStackTrace(test: Test): string | undefined {
  if (test.attempts && test.retryIndex !== undefined) {
    const currentAttempt = test.attempts[test.retryIndex];
    if (currentAttempt) {
      return currentAttempt.stackTrace || undefined;
    }
  }
  return test.stackTrace || undefined;
}

/**
 * Get errors array from the current attempt
 *
 * @param test - Test document
 * @returns Error array from current attempt or legacy field
 */
export function getTestErrors(test: Test): any[] {
  if (test.attempts && test.retryIndex !== undefined) {
    const currentAttempt = test.attempts[test.retryIndex];
    if (currentAttempt?.errors) {
      return currentAttempt.errors;
    }
  }
  return test.errors || [];
}

/**
 * Get attachments from the current attempt
 *
 * @param test - Test document
 * @returns Attachments from current attempt or legacy field
 */
export function getTestAttachments(test: Test): Record<string, any>[] {
  if (test.attempts && test.retryIndex !== undefined) {
    const currentAttempt = test.attempts[test.retryIndex];
    if (currentAttempt?.attachments) {
      return currentAttempt.attachments;
    }
  }
  return test.attachments || [];
}

/**
 * Get failures array from the current attempt
 *
 * @param test - Test document
 * @returns Failures from current attempt or legacy field
 */
export function getTestFailures(test: Test): any[] {
  if (test.attempts && test.retryIndex !== undefined) {
    const currentAttempt = test.attempts[test.retryIndex];
    if (currentAttempt?.failures) {
      return currentAttempt.failures;
    }
  }
  return test.failures || [];
}

/**
 * Get all attempts for a test (for display in UI)
 *
 * @param test - Test document
 * @returns Array of all attempts
 */
export function getAllAttempts(test: Test): Attempt[] {
  return test.attempts || [];
}

/**
 * Get current attempt data
 *
 * @param test - Test document
 * @returns Current attempt object or undefined
 */
export function getCurrentAttempt(test: Test): Attempt | undefined {
  if (test.attempts && test.retryIndex !== undefined && test.retryIndex >= 0) {
    return test.attempts[test.retryIndex];
  }
  return undefined;
}

/**
 * Get attempt by index
 *
 * @param test - Test document
 * @param index - Attempt index (0-based)
 * @returns Attempt object or undefined
 */
export function getAttemptByIndex(
  test: Test,
  index: number
): Attempt | undefined {
  if (test.attempts && index >= 0 && index < test.attempts.length) {
    return test.attempts[index];
  }
  return undefined;
}

/**
 * Check if test has multiple attempts (retries)
 *
 * @param test - Test document
 * @returns True if test has more than one attempt
 */
export function hasMultipleAttempts(test: Test): boolean {
  return (
    (test.retryCount !== undefined && test.retryCount > 0) ||
    (test.attempts !== undefined && test.attempts.length > 1)
  );
}

/**
 * Get attempt statistics (for displaying retry summary)
 *
 * @param test - Test document
 * @returns Object with attempt counts and status breakdown
 */
export function getAttemptStatistics(test: Test): {
  total: number;
  passed: number;
  failed: number;
  other: number;
} {
  const attempts = getAllAttempts(test);

  if (attempts.length === 0) {
    return { total: 0, passed: 0, failed: 0, other: 0 };
  }

  const stats = attempts.reduce(
    (acc, attempt) => {
      if (attempt.status === "PASSED") {
        acc.passed++;
      } else if (attempt.status === "FAILED") {
        acc.failed++;
      } else {
        acc.other++;
      }
      return acc;
    },
    { passed: 0, failed: 0, other: 0 }
  );

  return {
    total: attempts.length,
    ...stats,
  };
}

/**
 * Format attempt label for display (e.g., "Attempt 1 of 3")
 *
 * @param test - Test document
 * @returns Formatted label string
 */
export function formatAttemptLabel(test: Test): string {
  if (test.retryIndex === undefined || test.retryCount === undefined) {
    return "";
  }

  const current = test.retryIndex + 1;
  const total = test.retryCount + 1;

  return `Attempt ${current} of ${total}`;
}
