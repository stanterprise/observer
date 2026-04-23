import { Link } from "react-router-dom";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/Card";
import { Badge } from "@/components/Badge";
import { TagList } from "@/components/TagList";
import type { TestStatus } from "@/types/common";
import type { Attempt, Test, Step } from "@/types/testCase";
import { countNestedSteps, formatDuration } from "./utils";

type SummaryCardProps = {
  test: Test;
  testStatus: TestStatus;
  hasAttempts?: boolean;
  attempts: Attempt[];
  legacySteps: Step[];
};

export default function SummaryCard({
  test,
  testStatus,
  hasAttempts,
  attempts,
  legacySteps,
}: SummaryCardProps) {
  const totalSteps = hasAttempts
    ? attempts.reduce(
        (sum, attempt) => sum + countNestedSteps(attempt.steps || []),
        0,
      )
    : countNestedSteps(legacySteps);
  const topLevelSteps = hasAttempts
    ? attempts.reduce(
        (sum, attempt) =>
          sum + (attempt.stepsCount ?? attempt.steps?.length ?? 0),
        0,
      )
    : legacySteps.length;

  return (
    <Card>
      <CardHeader>
        <div className="flex items-start justify-between">
          <div className="flex-1 min-w-0">
            <CardTitle className="text-xl mb-2 wrap-break-word">
              {test.title || test.id}
            </CardTitle>
            <p className="text-sm text-(--stitch-on-surface-subtle) font-mono">
              {test.id}
            </p>
          </div>
          <Badge
            status={testStatus}
            className="text-lg px-4 py-2 shrink-0 ml-4"
          />
        </div>
      </CardHeader>
      <CardContent>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <div>
            <h3 className="text-sm font-semibold text-(--stitch-on-surface-muted) mb-3 uppercase tracking-wide">
              Test Information
            </h3>
            <dl className="space-y-3 text-sm">
              <div className="flex justify-between items-start">
                <dt className="text-(--stitch-on-surface-muted) font-medium">
                  Test ID:
                </dt>
                <dd className="font-mono text-(--stitch-on-surface) text-right break-all ml-4">
                  {test.id}
                </dd>
              </div>
              <div className="flex justify-between items-start">
                <dt className="text-(--stitch-on-surface-muted) font-medium">
                  Run ID:
                </dt>
                <dd className="text-right ml-4">
                  <Link
                    to={`/suite_runs/${test.runId}`}
                    className="font-mono text-(--stitch-primary) hover:underline focus:outline-none focus:ring-2 focus:ring-(--stitch-primary) focus:ring-offset-2 rounded"
                  >
                    {test.runId}
                  </Link>
                </dd>
              </div>
              <div className="flex justify-between items-start">
                <dt className="text-(--stitch-on-surface-muted) font-medium">
                  Duration:
                </dt>
                <dd className="text-(--stitch-on-surface) font-semibold text-right ml-4">
                  {formatDuration(test.duration)}
                </dd>
              </div>
              {test.retryCount !== undefined && test.retryCount > 0 && (
                <div className="flex justify-between items-start">
                  <dt className="text-(--stitch-on-surface-muted) font-medium">
                    Retries:
                  </dt>
                  <dd className="text-(--stitch-on-surface) text-right ml-4">
                    {test.retryIndex !== undefined
                      ? `${test.retryIndex} / ${test.retryCount}`
                      : test.retryCount}
                  </dd>
                </div>
              )}
              {test.timeout !== undefined && (
                <div className="flex justify-between items-start">
                  <dt className="text-(--stitch-on-surface-muted) font-medium">
                    Timeout:
                  </dt>
                  <dd className="text-(--stitch-on-surface) text-right ml-4">
                    {test.timeout}ms
                  </dd>
                </div>
              )}
            </dl>
          </div>
          <div>
            <h3 className="text-sm font-semibold text-(--stitch-on-surface-muted) mb-3 uppercase tracking-wide">
              Execution Timeline
            </h3>
            <dl className="space-y-3 text-sm">
              <div className="flex justify-between items-start">
                <dt className="text-(--stitch-on-surface-muted) font-medium">
                  Started:
                </dt>
                <dd className="text-(--stitch-on-surface) text-right ml-4">
                  {new Date(test.createdAt!).toLocaleString()}
                </dd>
              </div>
              <div className="flex justify-between items-start">
                <dt className="text-(--stitch-on-surface-muted) font-medium">
                  Last Updated:
                </dt>
                <dd className="text-(--stitch-on-surface) text-right ml-4">
                  {new Date(test.updatedAt!).toLocaleString()}
                </dd>
              </div>
              <div className="flex justify-between items-start">
                <dt className="text-(--stitch-on-surface-muted) font-medium">
                  Total Steps:
                </dt>
                <dd className="text-(--stitch-on-surface) font-semibold text-right ml-4">
                  {totalSteps}
                </dd>
              </div>
              {totalSteps !== topLevelSteps && (
                <div className="flex justify-between items-start">
                  <dt className="text-(--stitch-on-surface-muted) font-medium">
                    Top-level Steps:
                  </dt>
                  <dd className="text-(--stitch-on-surface) font-semibold text-right ml-4">
                    {topLevelSteps}
                  </dd>
                </div>
              )}
              {hasAttempts && attempts.length > 1 && (
                <div className="flex justify-between items-start">
                  <dt className="text-(--stitch-on-surface-muted) font-medium">
                    Total Attempts:
                  </dt>
                  <dd className="text-(--stitch-on-surface) font-semibold text-right ml-4">
                    {attempts.length}
                  </dd>
                </div>
              )}
            </dl>
          </div>
        </div>

        {/* Metadata Section */}
        {test.metadata && Object.keys(test.metadata).length > 0 && (
          <div className="mt-6 pt-6 border-t border-(--stitch-outline)">
            <h3 className="text-sm font-semibold text-(--stitch-on-surface-muted) mb-3 uppercase tracking-wide">
              Metadata
            </h3>
            <div className="bg-(--stitch-surface-low) rounded-lg p-4 border border-(--stitch-outline)">
              <pre className="text-xs text-(--stitch-on-surface) overflow-x-auto whitespace-pre-wrap wrap-break-word">
                {JSON.stringify(test.metadata, null, 2)}
              </pre>
            </div>
          </div>
        )}

        {/* Tags Section */}
        {test.tags && test.tags.length > 0 && (
          <div className="mt-6 pt-6 border-t border-(--stitch-outline)">
            <h3 className="text-sm font-semibold text-(--stitch-on-surface-muted) mb-3 uppercase tracking-wide">
              Tags
            </h3>
            <TagList tags={test.tags} />
          </div>
        )}
      </CardContent>
    </Card>
  );
}
