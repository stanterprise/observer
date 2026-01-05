import { useState } from "react";
import { ChevronsDown, ChevronsRight } from "lucide-react";
import type { Test, Step as StepType } from "@/types/testCase";
import type { TestStatus } from "@/types/common";
import { Badge } from "@/components/Badge";
import { Step } from "./Step";

type StepContainerProps = {
  test: Test;
};

export default ({ test }: StepContainerProps) => {
  const [expandAll, setExpandAll] = useState(false);
  const steps = buildStepHierarchies(test.steps!);
  const hasStepsWithChildren = steps.some(
    (step) => step.steps && step.steps.length > 0
  );

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
          {test.startTime && (
            <div>
              <span className="text-sm text-gray-500">Started</span>
              <p className="text-sm text-gray-900">
                {new Date(test.startTime).toLocaleString()}
              </p>
            </div>
          )}
          {test.endTime && (
            <div>
              <span className="text-sm text-gray-500">Finished</span>
              <p className="text-sm text-gray-900">
                {new Date(test.endTime).toLocaleString()}
              </p>
            </div>
          )}
        </div>
        {test.errors && (
          <div className="mt-4 p-3 bg-red-50 border border-red-200 rounded">
            <p className="text-sm font-medium text-red-800">Error</p>
            <p className="text-sm text-red-700 mt-1">
              {test.errors?.[0] || "Unknown error"}
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
      const aTime = a.startTime || "";
      const bTime = b.startTime || "";
      return aTime.localeCompare(bTime);
    });
}
