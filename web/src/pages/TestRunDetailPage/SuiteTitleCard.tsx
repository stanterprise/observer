import { Badge } from "@/components/Badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/Card";
import type { TestStatus } from "@/types/common";
import { CheckCircle, CircleDashed, CircleOff, XCircle } from "lucide-react";

import type { TestRun } from "@/types/testRun";

export type ProgressBarProps = {
  runDetail: TestRun;
  overallStatus?: TestStatus;
};

export const SuiteTitleCard = ({
  runDetail,
  overallStatus,
}: ProgressBarProps) => {
  const stats = runDetail.statistics!;
  const runningPendingCount =
    stats.total -
    stats.passed -
    stats.failed -
    stats.skipped -
    stats.broken! -
    stats.timedout! -
    stats.interrupted!;

  const totalFailedTestsCounts =
    stats.failed + stats.broken! + stats.timedout! + stats.interrupted!;

  return (
    <Card className="overflow-hidden border-(--stitch-outline)">
      {/* Enhanced Progress Bar */}
      <div className="relative flex h-8 bg-(--stitch-surface-low)">
        {stats.passed > 0 && (
          <div
            className="relative overflow-hidden bg-(--status-success) transition-all duration-500 ease-out"
            style={{
              width: `${(stats.passed / runDetail.totalTests!) * 100}%`,
            }}
            title={`${stats.passed} passed (${Math.round(
              (stats.passed / runDetail.totalTests!) * 100,
            )}%)`}
          >
            <div className="absolute inset-0 bg-linear-to-r from-transparent via-white/20 to-transparent animate-shimmer" />
          </div>
        )}
        {totalFailedTestsCounts > 0 && (
          <div
            className="bg-(--status-failure) transition-all duration-500 ease-out"
            style={{
              width: `${(totalFailedTestsCounts / runDetail.totalTests!) * 100}%`,
            }}
            title={`${totalFailedTestsCounts} failed (${Math.round(
              (totalFailedTestsCounts / runDetail.totalTests!) * 100,
            )}%)`}
          />
        )}
        {stats.skipped > 0 && (
          <div
            className="bg-(--status-neutral) transition-all duration-500 ease-out"
            style={{
              width: `${(stats.skipped / runDetail.totalTests!) * 100}%`,
            }}
            title={`${stats.skipped} skipped (${Math.round(
              (stats.skipped / runDetail.totalTests!) * 100,
            )}%)`}
          />
        )}
        {stats.total > 0 && runningPendingCount > 0 && (
          <div
            className="animate-pulse bg-(--status-running) transition-all duration-500 ease-out"
            style={{
              width: `${(runningPendingCount / runDetail.totalTests!) * 100}%`,
            }}
            title={`${runningPendingCount} running/pending (${Math.round(
              (runningPendingCount / runDetail.totalTests!) * 100,
            )}%)`}
          />
        )}
      </div>
      <CardHeader className="bg-(--stitch-surface-low)">
        <div className="flex items-center justify-between flex-wrap gap-4">
          <div className="flex-1 min-w-0">
            <CardTitle className="text-xl md:text-2xl mb-2 truncate">
              {runDetail.name ?? runDetail.id}
            </CardTitle>
            <div className="flex items-center gap-4 text-sm text-(--stitch-on-surface-muted)">
              <span className="font-medium">Total Tests:</span>
              <span className="font-bold text-(--stitch-on-surface)">
                {runDetail.totalTests}
              </span>
              {runDetail.createdAt && (
                <>
                  <span className="text-(--stitch-on-surface-subtle)">•</span>
                  <span className="text-(--stitch-on-surface-muted)">
                    {new Date(runDetail.createdAt).toLocaleDateString(
                      undefined,
                      {
                        month: "short",
                        day: "numeric",
                        year: "numeric",
                        hour: "2-digit",
                        minute: "2-digit",
                      },
                    )}
                  </span>
                </>
              )}
            </div>
          </div>
          <Badge
            status={overallStatus!}
            className="text-base px-4 py-2 shadow-sm"
          />
        </div>
      </CardHeader>
      <CardContent className="bg-(--stitch-surface-card)">
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 md:gap-6">
          <div className="group flex cursor-default flex-col items-center rounded-xl border p-4 transition-all duration-200 hover:scale-105 md:p-6"
            style={{ backgroundColor: "var(--status-success-soft)", borderColor: "var(--status-success-border)" }}>
            <CheckCircle className="mb-3 h-8 w-8 text-(--status-success) transition-transform group-hover:scale-110 md:h-10 md:w-10" />
            <div className="mb-1 text-3xl font-bold text-(--status-success) md:text-4xl">
              {runDetail.statistics!.passed}
            </div>
            <div className="text-xs font-medium uppercase tracking-wide text-(--stitch-on-surface) md:text-sm">
              Passed
            </div>
            {stats.total > 0 && (
              <div className="mt-1 text-xs font-semibold text-(--status-success)">
                {Math.round((stats.passed / stats.total) * 100)}%
              </div>
            )}
          </div>
          <div className="group flex cursor-default flex-col items-center rounded-xl border p-4 transition-all duration-200 hover:scale-105 md:p-6"
            style={{ backgroundColor: "var(--status-failure-soft)", borderColor: "var(--status-failure-border)" }}>
            <XCircle className="mb-3 h-8 w-8 text-(--status-failure) transition-transform group-hover:scale-110 md:h-10 md:w-10" />
            <div className="mb-1 text-3xl font-bold text-(--status-failure) md:text-4xl">
              {totalFailedTestsCounts}
            </div>
            <div className="text-xs font-medium uppercase tracking-wide text-(--stitch-on-surface) md:text-sm">
              Failed
            </div>
            {stats.total > 0 && (
              <div className="mt-1 text-xs font-semibold text-(--status-failure)">
                {Math.round((totalFailedTestsCounts / stats.total) * 100)}%
              </div>
            )}
          </div>
          <div className="group flex cursor-default flex-col items-center rounded-xl border p-4 transition-all duration-200 hover:scale-105 md:p-6"
            style={{ backgroundColor: "var(--status-neutral-soft)", borderColor: "var(--status-neutral-border)" }}>
            <CircleDashed className="mb-3 h-8 w-8 text-(--status-neutral) transition-transform group-hover:scale-110 md:h-10 md:w-10" />
            <div className="mb-1 text-3xl font-bold text-(--status-neutral) md:text-4xl">
              {runDetail.statistics!.skipped}
            </div>
            <div className="text-xs font-medium uppercase tracking-wide text-(--stitch-on-surface) md:text-sm">
              Skipped
            </div>
            {stats.total > 0 && (
              <div className="mt-1 text-xs font-semibold text-(--status-neutral)">
                {Math.round((stats.skipped / stats.total) * 100)}%
              </div>
            )}
          </div>
          <div className="group flex cursor-default flex-col items-center rounded-xl border p-4 transition-all duration-200 hover:scale-105 md:p-6"
            style={{ backgroundColor: "var(--status-running-soft)", borderColor: "var(--status-running-border)" }}>
            <CircleOff className="mb-3 h-8 w-8 text-(--status-running) transition-transform group-hover:scale-110 md:h-10 md:w-10" />
            <div className="mb-1 text-3xl font-bold text-(--status-running) md:text-4xl">
              {runningPendingCount}
            </div>
            <div className="text-xs font-medium uppercase tracking-wide text-(--stitch-on-surface) md:text-sm">
              Pending
            </div>
            {stats.total > 0 && (
              <div className="mt-1 text-xs font-semibold text-(--status-running)">
                {Math.round((runningPendingCount / stats.total) * 100)}%
              </div>
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  );
};
