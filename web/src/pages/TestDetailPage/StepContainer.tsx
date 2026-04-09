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
    (step) => step.steps && step.steps.length > 0,
  );

  return (
    <div key={test.id}>
      <div className="mb-6 p-4 bg-(--stitch-surface-low) rounded-lg">
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <div>
            <span className="text-sm text-(--stitch-on-surface-subtle)">
              Test ID
            </span>
            <p className="font-medium text-(--stitch-on-surface) truncate">
              {test.id}
            </p>
          </div>
          <div>
            <span className="text-sm text-(--stitch-on-surface-subtle)">
              Status
            </span>
            <div className="mt-1">
              <Badge status={test.status as TestStatus} />
            </div>
          </div>
          {test.startTime && (
            <div>
              <span className="text-sm text-(--stitch-on-surface-subtle)">
                Started
              </span>
              <p className="text-sm text-(--stitch-on-surface)">
                {new Date(test.startTime).toLocaleString()}
              </p>
            </div>
          )}
          {test.endTime && (
            <div>
              <span className="text-sm text-(--stitch-on-surface-subtle)">
                Finished
              </span>
              <p className="text-sm text-(--stitch-on-surface)">
                {new Date(test.endTime).toLocaleString()}
              </p>
            </div>
          )}
        </div>
        {test.errors && (
          <div className="mt-4 p-3 bg-(--status-failure-soft) border border-(--status-failure-border) rounded">
            <p className="text-sm font-medium text-(--status-failure)">Error</p>
            <p className="text-sm text-(--status-failure) mt-1">
              {test.errors?.[0] || "Unknown error"}
            </p>
          </div>
        )}
      </div>
      {hasStepsWithChildren && (
        <div className="mb-4 flex justify-end">
          <button
            onClick={() => setExpandAll(!expandAll)}
            className="flex items-center space-x-2 px-4 py-2 text-sm font-medium text-(--stitch-on-surface-muted) bg-(--stitch-surface-card) border border-(--stitch-outline) rounded-md hover:bg-(--stitch-surface-low) transition-colors"
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
  stepId: string | null = null,
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
