import { Badge } from "@/components/Badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/Card";
import type { TestStatus } from "@/types";
import { CheckCircle, CircleDashed, CircleOff, XCircle } from "lucide-react";
import type { RunDetail } from "./types";

export type ProgressBarProps = {
  runDetail: RunDetail;
  overallStatus?: TestStatus;
};

export const ProgressBar = ({ runDetail, overallStatus }: ProgressBarProps) => {
  const runningPendingCount =
    runDetail.statistics.total -
    runDetail.statistics.passed -
    runDetail.statistics.failed -
    runDetail.statistics.skipped;
  return (
    <Card>
      {/* Progress Bar */}
      <div className="h-8 bg-gray-200 rounded-t-lg overflow-hidden flex">
        {runDetail.statistics.passed > 0 && (
          <div
            className="bg-green-500 transition-all duration-300"
            style={{
              width: `${
                (runDetail.statistics.passed / runDetail.statistics.total) * 100
              }%`,
            }}
            title={`${runDetail.statistics.passed} passed`}
          />
        )}
        {runDetail.statistics.failed > 0 && (
          <div
            className="bg-red-500 transition-all duration-300"
            style={{
              width: `${
                (runDetail.statistics.failed / runDetail.statistics.total) * 100
              }%`,
            }}
            title={`${runDetail.statistics.failed} failed`}
          />
        )}
        {runDetail.statistics.skipped > 0 && (
          <div
            className="bg-gray-400 transition-all duration-300"
            style={{
              width: `${
                (runDetail.statistics.skipped / runDetail.statistics.total) *
                100
              }%`,
            }}
            title={`${runDetail.statistics.skipped} skipped`}
          />
        )}
        {runDetail.statistics.total > 0 && runningPendingCount > 0 && (
          <div
            className="bg-blue-300 transition-all duration-300 animate-pulse"
            style={{
              width: `${
                (runningPendingCount / runDetail.statistics.total) * 100
              }%`,
            }}
            title={`${runningPendingCount} running/pending`}
          />
        )}
      </div>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle className="text-xl mb-2">
              {runDetail.name ?? runDetail.id}
            </CardTitle>
            <div className="text-sm text-gray-500">
              Total Tests: {runDetail.tests.length}
            </div>
          </div>
          <Badge status={overallStatus!} className="text-lg px-4 py-2" />
        </div>
      </CardHeader>
      <CardContent>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-6">
          <div className="flex flex-col items-center p-4 bg-green-50 rounded-lg">
            <CheckCircle className="h-8 w-8 text-green-600 mb-2" />
            <div className="text-2xl font-bold text-green-600">
              {runDetail.statistics.passed}
            </div>
            <div className="text-sm text-gray-600">Passed</div>
          </div>
          <div className="flex flex-col items-center p-4 bg-red-50 rounded-lg">
            <XCircle className="h-8 w-8 text-red-600 mb-2" />
            <div className="text-2xl font-bold text-red-600">
              {runDetail.statistics.failed}
            </div>
            <div className="text-sm text-gray-600">Failed</div>
          </div>
          <div className="flex flex-col items-center p-4 bg-gray-50 rounded-lg">
            <CircleDashed className="h-8 w-8 text-gray-600 mb-2" />
            <div className="text-2xl font-bold text-gray-600">
              {runDetail.statistics.skipped}
            </div>
            <div className="text-sm text-gray-600">Skipped</div>
          </div>
          <div className="flex flex-col items-center p-4 bg-gray-50 rounded-lg">
            <CircleOff className="h-8 w-8 text-gray-600 mb-2" />
            <div className="text-2xl font-bold text-gray-600">
              {runDetail.statistics.unknown}
            </div>
            <div className="text-sm text-gray-600">Unknown</div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
};
