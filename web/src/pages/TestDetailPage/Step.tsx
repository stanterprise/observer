import { Card, CardContent } from "@/components/Card";
import { TagList } from "@/components/TagList";
import { useState, useEffect } from "react";
import {
  ChevronRight,
  ChevronDown,
} from "lucide-react";
import type { Step as StepType } from "@/types/testCase";
import { Badge } from "@/components/Badge";

type StepProps = {
  step: StepType;
  globalExpandAll?: boolean;
};

export const Step = ({ step, globalExpandAll }: StepProps) => {
  const [isExpanded, setIsExpanded] = useState(globalExpandAll ?? false);
  const hasChildren = step.steps && step.steps.length > 0;
  const hasError = step.error || (step.errors && step.errors.length > 0);
  const shouldShowError = hasError && (step.status === "FAILED" || step.status === "BROKEN" || step.status === "TIMEDOUT");

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
                    className="p-0 hover:bg-gray-100 rounded transition-colors"
                    aria-label={
                      isExpanded ? "Collapse substeps" : "Expand substeps"
                    }
                  >
                    {isExpanded ? (
                      <ChevronDown className="w-5 h-5 text-gray-600" />
                    ) : (
                      <ChevronRight className="w-5 h-5 text-gray-600" />
                    )}
                  </button>
                )}
                <Badge status={step.status || "UNKNOWN"} />
                <h3 className="text-base font-medium text-gray-900 truncate">
                  {step.title || step.id}
                </h3>
              </div>
              {step.tags && step.tags.length > 0 && (
                <div className="mt-2 ml-8">
                  <TagList tags={step.tags} />
                </div>
              )}
              {shouldShowError && (
                <div className="mt-4 p-3 bg-red-50 border border-red-200 rounded">
                  <p className="text-sm font-medium text-red-800">Error</p>
                  <p className="text-sm text-red-700 mt-1 whitespace-pre-wrap break-words">
                    {step.error || step.errors?.[0] || "Unknown error"}
                  </p>
                </div>
              )}
            </div>
          </div>
        </CardContent>
      </Card>
      {hasChildren && isExpanded && (
        <div className="pl-6 border-l-2 border-gray-200">
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
