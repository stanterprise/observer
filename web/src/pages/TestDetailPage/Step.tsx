import { Card, CardContent } from "@/components/Card";
import { TagList } from "@/components/TagList";
import { useState, useEffect } from "react";
import { ChevronRight, ChevronDown } from "lucide-react";
import type { Step as StepType } from "@/types/testCase";
import { Badge } from "@/components/Badge";
import { ansiToHtml } from "@/utils/ansi";

type StepProps = {
  step: StepType;
  globalExpandAll?: boolean;
};

export const Step = ({ step, globalExpandAll }: StepProps) => {
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

  // Update local state when global state changes
  useEffect(() => {
    setIsExpanded(globalExpandAll ?? false);
  }, [globalExpandAll]);

  return (
    <div>
      <Card className="mb-4">
        <CardContent className="py-4">
          <div className="flex items-center justify-between">
            <div className="flex-1 min-w-0">
              <div className="flex items-center space-x-3 mb-2">
                {hasChildren && (
                  <button
                    onClick={() => setIsExpanded(!isExpanded)}
                    className="p-0 hover:bg-(--stitch-surface-low) rounded transition-colors"
                    aria-label={
                      isExpanded ? "Collapse substeps" : "Expand substeps"
                    }
                  >
                    {isExpanded ? (
                      <ChevronDown className="w-5 h-5 text-(--stitch-on-surface-muted)" />
                    ) : (
                      <ChevronRight className="w-5 h-5 text-(--stitch-on-surface-muted)" />
                    )}
                  </button>
                )}
                <Badge status={step.status || "UNKNOWN"} />
                <h3 className="text-base font-medium text-(--stitch-on-surface) truncate">
                  {step.title || step.id}
                </h3>
              </div>
              {step.tags && step.tags.length > 0 && (
                <div className="mt-2 ml-8">
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
        <div className="pl-6 border-l-2 border-(--stitch-outline)">
          {step.steps?.map((subStep) => (
            <Step
              key={subStep.id}
              step={subStep}
              globalExpandAll={globalExpandAll}
            />
          ))}
        </div>
      )}
    </div>
  );
};
