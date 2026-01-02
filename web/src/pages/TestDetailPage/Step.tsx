import { Card, CardContent } from "@/components/Card";
import { useState, useEffect } from "react";
import {
  CheckCircle2,
  AlertCircle,
  Clock,
  ChevronRight,
  ChevronDown,
} from "lucide-react";
import type { Step as StepType } from "@/types/testCase";
import type { TestStatus } from "@/types/common";
import { Badge } from "@/components/Badge";

type StepProps = {
  step: StepType;
  globalExpandAll?: boolean;
};

export const Step = ({ step, globalExpandAll }: StepProps) => {
  const [isExpanded, setIsExpanded] = useState(globalExpandAll ?? false);
  const hasChildren = step.steps && step.steps.length > 0;

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
                {step.status === "passed" ? (
                  <>
                    <CheckCircle2 className="w-5 h-5 text-green-600" />
                    <Badge status={"success" as TestStatus} />
                  </>
                ) : step.status === "failed" ? (
                  <>
                    <AlertCircle className="w-5 h-5 text-red-600" />
                    <Badge status={"error" as TestStatus} />
                  </>
                ) : (
                  <>
                    <Clock className="w-5 h-5 text-yellow-600" />
                    <Badge status={"pending" as TestStatus} />
                  </>
                )}
                <h3 className="text-base font-medium text-gray-900 truncate">
                  {step.title || step.id}
                </h3>
              </div>
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
