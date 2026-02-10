import { Link } from "react-router-dom";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/Card";
import { Badge } from "@/components/Badge";
import { TagList } from "@/components/TagList";
import type { TestStatus } from "@/types/common";
import type { Attempt, Test } from "@/types/testCase";
import { formatDuration } from "./utils";

type SummaryCardProps = {
  test: Test;
  testStatus: TestStatus;
  hasAttempts?: boolean;
  attempts: Attempt[];
  legacySteps: any[];
};

export default function SummaryCard({
  test,
  testStatus,
  hasAttempts,
  attempts,
  legacySteps,
}: SummaryCardProps) {
  return (
    <Card>
      <CardHeader>
        <div className="flex items-start justify-between">
          <div className="flex-1 min-w-0">
            <CardTitle className="text-xl mb-2 wrap-break-word">
              {test.title || test.id}
            </CardTitle>
            <p className="text-sm text-gray-500 font-mono">{test.id}</p>
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
            <h3 className="text-sm font-semibold text-gray-700 mb-3 uppercase tracking-wide">
              Test Information
            </h3>
            <dl className="space-y-3 text-sm">
              <div className="flex justify-between items-start">
                <dt className="text-gray-600 font-medium">Test ID:</dt>
                <dd className="font-mono text-gray-900 text-right break-all ml-4">
                  {test.id}
                </dd>
              </div>
              <div className="flex justify-between items-start">
                <dt className="text-gray-600 font-medium">Run ID:</dt>
                <dd className="text-right ml-4">
                  <Link
                    to={`/suite_runs/${test.runId}`}
                    className="font-mono text-blue-600 hover:underline focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 rounded"
                  >
                    {test.runId}
                  </Link>
                </dd>
              </div>
              <div className="flex justify-between items-start">
                <dt className="text-gray-600 font-medium">Duration:</dt>
                <dd className="text-gray-900 font-semibold text-right ml-4">
                  {formatDuration(test.duration)}
                </dd>
              </div>
              {test.retryCount !== undefined && test.retryCount > 0 && (
                <div className="flex justify-between items-start">
                  <dt className="text-gray-600 font-medium">Retries:</dt>
                  <dd className="text-gray-900 text-right ml-4">
                    {test.retryIndex !== undefined
                      ? `${test.retryIndex} / ${test.retryCount}`
                      : test.retryCount}
                  </dd>
                </div>
              )}
              {test.timeout !== undefined && (
                <div className="flex justify-between items-start">
                  <dt className="text-gray-600 font-medium">Timeout:</dt>
                  <dd className="text-gray-900 text-right ml-4">
                    {test.timeout}ms
                  </dd>
                </div>
              )}
            </dl>
          </div>
          <div>
            <h3 className="text-sm font-semibold text-gray-700 mb-3 uppercase tracking-wide">
              Execution Timeline
            </h3>
            <dl className="space-y-3 text-sm">
              <div className="flex justify-between items-start">
                <dt className="text-gray-600 font-medium">Started:</dt>
                <dd className="text-gray-900 text-right ml-4">
                  {new Date(test.createdAt!).toLocaleString()}
                </dd>
              </div>
              <div className="flex justify-between items-start">
                <dt className="text-gray-600 font-medium">Last Updated:</dt>
                <dd className="text-gray-900 text-right ml-4">
                  {new Date(test.updatedAt!).toLocaleString()}
                </dd>
              </div>
              <div className="flex justify-between items-start">
                <dt className="text-gray-600 font-medium">Total Steps:</dt>
                <dd className="text-gray-900 font-semibold text-right ml-4">
                  {hasAttempts
                    ? attempts.reduce(
                        (sum, attempt) => sum + (attempt.steps?.length || 0),
                        0,
                      )
                    : legacySteps.length}
                </dd>
              </div>
              {hasAttempts && attempts.length > 1 && (
                <div className="flex justify-between items-start">
                  <dt className="text-gray-600 font-medium">Total Attempts:</dt>
                  <dd className="text-gray-900 font-semibold text-right ml-4">
                    {attempts.length}
                  </dd>
                </div>
              )}
            </dl>
          </div>
        </div>

        {/* Metadata Section */}
        {test.metadata && Object.keys(test.metadata).length > 0 && (
          <div className="mt-6 pt-6 border-t border-gray-200">
            <h3 className="text-sm font-semibold text-gray-700 mb-3 uppercase tracking-wide">
              Metadata
            </h3>
            <div className="bg-gray-50 rounded-lg p-4 border border-gray-200">
              <pre className="text-xs text-gray-800 overflow-x-auto whitespace-pre-wrap wrap-break-word">
                {JSON.stringify(test.metadata, null, 2)}
              </pre>
            </div>
          </div>
        )}

        {/* Tags Section */}
        {test.tags && test.tags.length > 0 && (
          <div className="mt-6 pt-6 border-t border-gray-200">
            <h3 className="text-sm font-semibold text-gray-700 mb-3 uppercase tracking-wide">
              Tags
            </h3>
            <TagList tags={test.tags} />
          </div>
        )}
      </CardContent>
    </Card>
  );
}
