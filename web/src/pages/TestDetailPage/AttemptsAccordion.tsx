import { useState } from "react";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/Card";
import { Badge } from "@/components/Badge";
import type { TestStatus } from "@/types/common";
import { ChevronDown, ChevronRight } from "lucide-react";
import StepContainer from "./StepContainer";
import type { Attempt, Test } from "@/types/testCase";
import { getTestStatus, formatDuration } from "./utils";

// Helper component for rendering attempts in an accordion
export default function AttemptsAccordion({
  test,
  attempts,
}: {
  test: Test;
  attempts: Attempt[];
}) {
  const [openAttempt, setOpenAttempt] = useState<number>(
    test.retryIndex ?? attempts.length - 1,
  );

  const getAttemptStatus = (attempt: Attempt): TestStatus => {
    if (!attempt.status) return "PENDING";
    return getTestStatus(attempt.status);
  };

  const toggleAttempt = (retryIndex: number) => {
    setOpenAttempt(openAttempt === retryIndex ? -1 : retryIndex);
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-lg">Test Execution Attempts</CardTitle>
        <p className="text-sm text-gray-600 mt-1">
          {attempts.length} attempt{attempts.length > 1 ? "s" : ""} recorded
        </p>
      </CardHeader>
      <CardContent>
        <div className="space-y-3">
          {attempts.map((attempt) => {
            const isOpen = openAttempt === attempt.retryIndex;
            const attemptStatus = getAttemptStatus(attempt);
            const attemptSteps = attempt.steps || [];

            return (
              <div
                key={attempt.retryIndex}
                className="border border-gray-200 rounded-lg overflow-hidden"
              >
                {/* Accordion Header */}
                <button
                  onClick={() => toggleAttempt(attempt.retryIndex)}
                  className="w-full flex items-center justify-between p-4 bg-gray-50 hover:bg-gray-100 transition-colors"
                >
                  <div className="flex items-center gap-3">
                    {isOpen ? (
                      <ChevronDown className="h-5 w-5 text-gray-600" />
                    ) : (
                      <ChevronRight className="h-5 w-5 text-gray-600" />
                    )}
                    <div className="text-left">
                      <div className="flex items-center gap-2">
                        <span className="font-semibold text-gray-900">
                          Attempt {attempt.retryIndex + 1}
                        </span>
                        {attempt.retryIndex === test.retryIndex && (
                          <span className="text-xs px-2 py-0.5 bg-blue-100 text-blue-700 rounded-full font-medium">
                            Current
                          </span>
                        )}
                      </div>
                      <div className="text-sm text-gray-600 mt-1">
                        {attemptSteps.length} step
                        {attemptSteps.length !== 1 ? "s" : ""}
                        {attempt.startTime && (
                          <span className="ml-2">
                            • Started{" "}
                            {new Date(attempt.startTime).toLocaleString()}
                          </span>
                        )}
                      </div>
                    </div>
                  </div>
                  <Badge status={attemptStatus} />
                </button>

                {/* Accordion Body */}
                {isOpen && (
                  <div className="p-4 bg-white border-t border-gray-200">
                    {/* Attempt Info */}
                    <div className="mb-4 p-3 bg-gray-50 rounded-lg">
                      <dl className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
                        <div>
                          <dt className="text-gray-600 font-medium">Status</dt>
                          <dd className="mt-1">
                            <Badge status={attemptStatus} />
                          </dd>
                        </div>
                        {attempt.startTime && (
                          <div>
                            <dt className="text-gray-600 font-medium">
                              Started
                            </dt>
                            <dd className="text-gray-900 mt-1">
                              {new Date(attempt.startTime).toLocaleString()}
                            </dd>
                          </div>
                        )}
                        {attempt.endTime && (
                          <div>
                            <dt className="text-gray-600 font-medium">
                              Finished
                            </dt>
                            <dd className="text-gray-900 mt-1">
                              {new Date(attempt.endTime).toLocaleString()}
                            </dd>
                          </div>
                        )}
                        {attempt.duration !== undefined && (
                          <div>
                            <dt className="text-gray-600 font-medium">
                              Duration
                            </dt>
                            <dd className="text-gray-900 mt-1 font-semibold">
                              {formatDuration(attempt.duration)}
                            </dd>
                          </div>
                        )}
                      </dl>
                    </div>

                    {/* Error Display */}
                    {(attempt.errorMessage ||
                      (attempt.errors && attempt.errors.length > 0)) && (
                      <div className="mb-4 p-4 bg-red-50 border border-red-200 rounded-lg">
                        <h4 className="text-sm font-semibold text-red-800 mb-2">
                          Error Details
                        </h4>
                        {attempt.errorMessage && (
                          <p className="text-sm text-red-700 mb-2">
                            {attempt.errorMessage}
                          </p>
                        )}
                        {attempt.stackTrace && (
                          <pre className="text-xs text-red-600 bg-red-100 p-2 rounded overflow-x-auto whitespace-pre-wrap">
                            {attempt.stackTrace}
                          </pre>
                        )}
                        {attempt.errors && attempt.errors.length > 0 && (
                          <ul className="text-sm text-red-700 list-disc list-inside space-y-1">
                            {attempt.errors.map((err, idx) => (
                              <li key={idx}>
                                {typeof err === "string"
                                  ? err
                                  : JSON.stringify(err)}
                              </li>
                            ))}
                          </ul>
                        )}
                      </div>
                    )}

                    {/* Steps */}
                    {attemptSteps.length > 0 ? (
                      <StepContainer
                        test={{
                          id: test.id,
                          runId: test.runId,
                          title: `Attempt ${attempt.retryIndex + 1}`,
                          status: attemptStatus,
                          steps: attemptSteps.map((step) => ({
                            id: step.id,
                            runId: step.runId || test.runId,
                            testCaseRunId: step.testCaseRunId,
                            parentStepId:
                              step.parentStepId && step.parentStepId !== ""
                                ? step.parentStepId
                                : undefined,
                            status: getTestStatus(step.status),
                            category: step.category,
                            title: step.title,
                            startedAt: step.startTime || step.createdAt,
                            finishedAt: step.updatedAt,
                            error: (step as any).error,
                            errors: (step as any).errors,
                            metadata: (step as any).metadata,
                            duration: (step as any).duration,
                            location: (step as any).location,
                          })),
                        }}
                      />
                    ) : (
                      <div className="text-center py-8 text-gray-500">
                        <p className="text-sm">
                          No steps recorded for this attempt
                        </p>
                      </div>
                    )}
                  </div>
                )}
              </div>
            );
          })}
        </div>
      </CardContent>
    </Card>
  );
}
