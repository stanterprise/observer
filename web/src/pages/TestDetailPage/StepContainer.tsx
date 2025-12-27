import { Card, CardContent } from "@/components/Card";
import { useState } from "react";
import {
  CheckCircle2,
  AlertCircle,
  Clock,
  ChevronRight,
  ChevronDown,
  ChevronsDown,
  ChevronsRight,
} from "lucide-react";
import type { Test, Step as StepType, TestStatus } from "@/types";
import { Badge } from "@/components/Badge";

type StepContainerProps = {
  test: Test;
};

type StepProps = {
  step: StepType;
  globalExpandAll?: boolean;
};
export default ({ test }: StepContainerProps) => {
  const [expandAll, setExpandAll] = useState(false);
  const steps = buildStepHierarchies(test.steps!);
  const hasStepsWithChildren = steps.some(
    (step) => step.steps && step.steps.length > 0
  );

  console.log("Rendering steps for test:", test.id, steps);
  return (
    <div key={test.id}>
      <div className="mb-6 p-4 bg-gray-50 rounded-lg">
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <div>
            <span className="text-sm text-gray-500">Test ID</span>
            <p className="font-medium text-gray-900 truncate">{test.id}</p>
          </div>
          <div>
            <span className="text-sm text-gray-500">Status</span>
            <div className="mt-1">
              <Badge status={test.status as TestStatus} />
            </div>
          </div>
          {test.startedAt && (
            <div>
              <span className="text-sm text-gray-500">Started</span>
              <p className="text-sm text-gray-900">
                {new Date(test.startedAt).toLocaleString()}
              </p>
            </div>
          )}
          {test.finishedAt && (
            <div>
              <span className="text-sm text-gray-500">Finished</span>
              <p className="text-sm text-gray-900">
                {new Date(test.finishedAt).toLocaleString()}
              </p>
            </div>
          )}
        </div>
        {test.error && (
          <div className="mt-4 p-3 bg-red-50 border border-red-200 rounded">
            <p className="text-sm font-medium text-red-800">Error</p>
            <p className="text-sm text-red-700 mt-1">
              {test.error?.message || "Unknown error"}
            </p>
          </div>
        )}
      </div>
      {hasStepsWithChildren && (
        <div className="mb-4 flex justify-end">
          <button
            onClick={() => setExpandAll(!expandAll)}
            className="flex items-center space-x-2 px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 transition-colors"
          >
            {expandAll ? (
              <>
                <ChevronsRight className="w-4 h-4" />
                <span>Collapse All</span>
              </>
            ) : (
              <>
                <ChevronsDown className="w-4 h-4" />
                <span>Expand All</span>
              </>
            )}
          </button>
        </div>
      )}
      {steps.map((step) => (
        <Step key={step.id} step={step} globalExpandAll={expandAll} />
      ))}
    </div>
  );
};

export const Step = ({ step, globalExpandAll }: StepProps) => {
  const [localExpanded, setLocalExpanded] = useState(false);
  const hasChildren = step.steps && step.steps.length > 0;
  const isExpanded =
    globalExpandAll !== undefined ? globalExpandAll : localExpanded;

  console.log("Rendering step:", step);

  return (
    <div>
      <Card className="mb-4">
        <CardContent className="py-4">
          <div className="flex items-center justify-between">
            <div className="flex-1 min-w-0">
              <div className="flex items-center space-x-3 mb-2">
                {hasChildren && (
                  <button
                    onClick={() => setLocalExpanded(!localExpanded)}
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
          {step.steps.map((subStep) => (
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

function buildStepHierarchies(
  steps: StepType[],
  stepId: string | null = null
): StepType[] {
  const filteredSteps = steps.filter((s) => {
    const parentId = s.parentStepId || null;
    return parentId === stepId;
  });
  return filteredSteps
    .map((step) => ({
      ...step,
      steps: buildStepHierarchies(steps, step.id),
    }))
    .sort((a, b) => {
      const aTime = a.startedAt || "";
      const bTime = b.startedAt || "";
      return aTime.localeCompare(bTime);
    });
}
