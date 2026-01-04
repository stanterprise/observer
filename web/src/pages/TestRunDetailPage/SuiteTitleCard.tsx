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
    stats.total - stats.passed - stats.failed - stats.skipped;

  return (
    <Card className="overflow-hidden shadow-lg border-gray-200">
      {/* Enhanced Progress Bar */}
      <div className="h-2 bg-gray-100 flex relative group">
        {stats.passed > 0 && (
          <div
            className="bg-linear-to-r from-green-500 to-green-600 transition-all duration-500 ease-out relative overflow-hidden"
            style={{
              width: `${(stats.passed / stats.total) * 100}%`,
            }}
            title={`${stats.passed} passed (${Math.round(
              (stats.passed / stats.total) * 100
            )}%)`}
          >
            <div className="absolute inset-0 bg-linear-to-r from-transparent via-white/20 to-transparent animate-shimmer" />
          </div>
        )}
        {stats.failed > 0 && (
          <div
            className="bg-linear-to-r from-red-500 to-red-600 transition-all duration-500 ease-out"
            style={{
              width: `${(stats.failed / stats.total) * 100}%`,
            }}
            title={`${stats.failed} failed (${Math.round(
              (stats.failed / stats.total) * 100
            )}%)`}
          />
        )}
        {stats.skipped > 0 && (
          <div
            className="bg-linear-to-r from-gray-400 to-gray-500 transition-all duration-500 ease-out"
            style={{
              width: `${(stats.skipped / stats.total) * 100}%`,
            }}
            title={`${stats.skipped} skipped (${Math.round(
              (stats.skipped / stats.total) * 100
            )}%)`}
          />
        )}
        {stats.total > 0 && runningPendingCount > 0 && (
          <div
            className="bg-linear-to-r from-blue-400 to-blue-500 transition-all duration-500 ease-out animate-pulse"
            style={{
              width: `${(runningPendingCount / stats.total) * 100}%`,
            }}
            title={`${runningPendingCount} running/pending (${Math.round(
              (runningPendingCount / stats.total) * 100
            )}%)`}
          />
        )}
      </div>
      <CardHeader className="bg-linear-to-r from-gray-50 to-white">
        <div className="flex items-center justify-between flex-wrap gap-4">
          <div className="flex-1 min-w-0">
            <CardTitle className="text-xl md:text-2xl mb-2 truncate">
              {runDetail.name ?? runDetail.id}
            </CardTitle>
            <div className="flex items-center gap-4 text-sm text-gray-600">
              <span className="font-medium">Total Tests:</span>
              <span className="font-bold text-gray-900">
                {runDetail.tests!.length}
              </span>
              {runDetail.createdAt && (
                <>
                  <span className="text-gray-300">•</span>
                  <span className="text-gray-500">
                    {new Date(runDetail.createdAt).toLocaleDateString(
                      undefined,
                      {
                        month: "short",
                        day: "numeric",
                        year: "numeric",
                        hour: "2-digit",
                        minute: "2-digit",
                      }
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
      <CardContent className="bg-white">
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 md:gap-6">
          <div className="group flex flex-col items-center p-4 md:p-6 bg-linear-to-br from-green-50 to-green-100/50 rounded-xl border border-green-200 transition-all duration-200 hover:shadow-md hover:scale-105 cursor-default">
            <CheckCircle className="h-8 w-8 md:h-10 md:w-10 text-green-600 mb-3 group-hover:scale-110 transition-transform" />
            <div className="text-3xl md:text-4xl font-bold text-green-700 mb-1">
              {runDetail.statistics!.passed}
            </div>
            <div className="text-xs md:text-sm text-gray-700 font-medium uppercase tracking-wide">
              Passed
            </div>
            {stats.total > 0 && (
              <div className="text-xs text-green-600 font-semibold mt-1">
                {Math.round((stats.passed / stats.total) * 100)}%
              </div>
            )}
          </div>
          <div className="group flex flex-col items-center p-4 md:p-6 bg-linear-to-br from-red-50 to-red-100/50 rounded-xl border border-red-200 transition-all duration-200 hover:shadow-md hover:scale-105 cursor-default">
            <XCircle className="h-8 w-8 md:h-10 md:w-10 text-red-600 mb-3 group-hover:scale-110 transition-transform" />
            <div className="text-3xl md:text-4xl font-bold text-red-700 mb-1">
              {runDetail.statistics!.failed}
            </div>
            <div className="text-xs md:text-sm text-gray-700 font-medium uppercase tracking-wide">
              Failed
            </div>
            {stats.total > 0 && (
              <div className="text-xs text-red-600 font-semibold mt-1">
                {Math.round((stats.failed / stats.total) * 100)}%
              </div>
            )}
          </div>
          <div className="group flex flex-col items-center p-4 md:p-6 bg-linear-to-br from-gray-50 to-gray-100/50 rounded-xl border border-gray-200 transition-all duration-200 hover:shadow-md hover:scale-105 cursor-default">
            <CircleDashed className="h-8 w-8 md:h-10 md:w-10 text-gray-600 mb-3 group-hover:scale-110 transition-transform" />
            <div className="text-3xl md:text-4xl font-bold text-gray-700 mb-1">
              {runDetail.statistics!.skipped}
            </div>
            <div className="text-xs md:text-sm text-gray-700 font-medium uppercase tracking-wide">
              Skipped
            </div>
            {stats.total > 0 && (
              <div className="text-xs text-gray-600 font-semibold mt-1">
                {Math.round((stats.skipped / stats.total) * 100)}%
              </div>
            )}
          </div>
          <div className="group flex flex-col items-center p-4 md:p-6 bg-linear-to-br from-gray-50 to-gray-100/50 rounded-xl border border-gray-200 transition-all duration-200 hover:shadow-md hover:scale-105 cursor-default">
            <CircleOff className="h-8 w-8 md:h-10 md:w-10 text-gray-600 mb-3 group-hover:scale-110 transition-transform" />
            <div className="text-3xl md:text-4xl font-bold text-gray-700 mb-1">
              {runningPendingCount}
            </div>
            <div className="text-xs md:text-sm text-gray-700 font-medium uppercase tracking-wide">
              Pending
            </div>
            {stats.total > 0 && (
              <div className="text-xs text-gray-600 font-semibold mt-1">
                {Math.round((runningPendingCount / stats.total) * 100)}%
              </div>
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  );
};
