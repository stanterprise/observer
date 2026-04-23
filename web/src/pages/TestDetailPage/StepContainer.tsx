import { useState } from "react";
import { ChevronsDown, ChevronsRight } from "lucide-react";
import type { Test, Step as StepType } from "@/types/testCase";
import type { TestStatus } from "@/types/common";
import { Badge } from "@/components/Badge";
import { Step } from "./Step";
import { countExpandableSteps, countNestedSteps, getTestStatus } from "./utils";

type StepContainerProps = {
  test: Test;
};

export default ({ test }: StepContainerProps) => {
  const [expandAll, setExpandAll] = useState(false);
  const steps = buildStepHierarchies(test.steps ?? [], test.runId);
  const totalStepCount = countNestedSteps(steps);
  const expandableStepCount = countExpandableSteps(steps);
  const topLevelStepCount = steps.length;
  const hasStepsWithChildren = steps.some(
    (step) => step.steps && step.steps.length > 0,
  );

  return (
    <div key={test.id}>
      <div className="mb-4 rounded-lg bg-(--stitch-surface-low) p-3">
        <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
          <div className="space-y-1.5">
            <div className="flex flex-wrap items-center gap-1.5">
              <span className="text-xs font-semibold uppercase tracking-[0.16em] text-(--stitch-on-surface-subtle)">
                Step Explorer
              </span>
              <Badge
                status={test.status as TestStatus}
                showIcon={false}
                className="px-2 py-0 text-[11px]"
              />
            </div>
            <div className="flex flex-wrap items-center gap-1.5 text-xs text-(--stitch-on-surface-muted)">
              <span className="rounded-full bg-(--stitch-surface-card) px-2.5 py-0.5 font-semibold text-(--stitch-on-surface)">
                {totalStepCount} total
              </span>
              {totalStepCount !== topLevelStepCount && (
                <span className="rounded-full bg-(--stitch-surface-card) px-2.5 py-0.5 font-medium text-(--stitch-on-surface-muted)">
                  {topLevelStepCount} top-level
                </span>
              )}
              {expandableStepCount > 0 && (
                <span className="rounded-full bg-(--stitch-surface-card) px-2.5 py-0.5 font-medium text-(--stitch-on-surface-muted)">
                  {expandableStepCount} expandable
                </span>
              )}
            </div>
            <p className="text-xs text-(--stitch-on-surface-subtle)">
              {test.title && test.title !== test.id ? `${test.title} · ` : ""}
              Nested execution order with hooks, steps, and assertions.
            </p>
          </div>
          {hasStepsWithChildren && (
            <button
              onClick={() => setExpandAll(!expandAll)}
              className="inline-flex items-center justify-center gap-1.5 rounded-md bg-(--stitch-surface-card) px-3 py-1.5 text-xs font-medium text-(--stitch-on-surface-muted) transition-colors hover:bg-(--stitch-surface-highest)"
            >
              {expandAll ? (
                <>
                  <ChevronsRight className="h-3.5 w-3.5" />
                  <span>Collapse All</span>
                </>
              ) : (
                <>
                  <ChevronsDown className="h-3.5 w-3.5" />
                  <span>Expand All</span>
                </>
              )}
            </button>
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
      {steps.map((step) => (
        <Step key={step.id} step={step} globalExpandAll={expandAll} depth={0} />
      ))}
    </div>
  );
};

function buildStepHierarchies(
  steps: StepType[],
  fallbackRunId: string,
): StepType[] {
  const flattenedSteps = flattenSteps(steps, fallbackRunId);
  const stepsById = new Map(
    flattenedSteps.map((step) => [
      step.id,
      { ...step, steps: [] as StepType[] },
    ]),
  );
  const rootSteps: StepType[] = [];

  flattenedSteps.forEach((step) => {
    const normalizedStep = stepsById.get(step.id);
    if (!normalizedStep) {
      return;
    }

    const parentId = step.parentStepId || null;
    if (parentId) {
      const parent = stepsById.get(parentId);
      if (parent) {
        parent.steps = [...(parent.steps || []), normalizedStep];
        return;
      }
    }

    rootSteps.push(normalizedStep);
  });

  return sortStepTree(rootSteps);
}

function flattenSteps(
  steps: StepType[],
  fallbackRunId: string,
  parentStepId?: string,
  seen = new Set<string>(),
): StepType[] {
  return steps.flatMap((step) => {
    if (!step?.id || seen.has(step.id)) {
      return [];
    }

    seen.add(step.id);

    const normalizedStep: StepType = {
      ...step,
      runId: step.runId || fallbackRunId,
      title: step.title || step.id,
      status: getTestStatus(step.status),
      category:
        step.category ||
        (typeof step.metadata?.category === "string"
          ? step.metadata.category
          : undefined),
      parentStepId:
        step.parentStepId && step.parentStepId !== ""
          ? step.parentStepId
          : parentStepId,
      startTime: step.startTime || step.createdAt,
      steps: [],
    };

    const childSteps = flattenSteps(
      step.steps || [],
      normalizedStep.runId,
      normalizedStep.id,
      seen,
    );

    return [normalizedStep, ...childSteps];
  });
}

function sortStepTree(steps: StepType[]): StepType[] {
  return [...steps]
    .sort((a, b) => {
      const aTime = a.startTime || a.createdAt || "";
      const bTime = b.startTime || b.createdAt || "";
      return aTime.localeCompare(bTime);
    })
    .map((step) => ({
      ...step,
      steps: step.steps ? sortStepTree(step.steps) : [],
    }));
}
