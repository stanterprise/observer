import { Clock } from "lucide-react";
import { Badge } from "../../components/Badge";
import { Link } from "react-router-dom";
import { Card, CardContent } from "@/components/Card";

import type { Test } from "@/types/testCase";
import { getTestStatus, formatDuration } from "./utils";

type TestRecordProps = {
  test: Test;
  runId: string;
};

export default ({ test, runId }: TestRecordProps) => {
  return (
    <Link key={test.id} to={`/suite_runs/${runId}/tests/${test.id}`}>
      <Card className="hover:shadow-lg transition-all duration-200 cursor-pointer hover:border-blue-400 hover:scale-[1.01] group">
        <CardContent className="py-4">
          <div className="flex items-center justify-between">
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-3 mb-2">
                <Badge status={getTestStatus(test.status)} />
                <h3 className="text-base font-medium text-gray-900 truncate group-hover:text-blue-600 transition-colors">
                  {test.title || test.id}
                </h3>
              </div>
              <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-gray-500">
                <div className="flex items-center">
                  <Clock className="h-4 w-4 mr-1 text-gray-400" />
                  <span className="font-medium">Duration:</span>
                  <span className="ml-1 font-semibold text-gray-700">
                    {formatDuration(test.duration)}
                  </span>
                </div>
                {test.retryCount !== undefined && test.retryCount > 0 && (
                  <div className="flex items-center">
                    <span className="font-medium">Retries:</span>
                    <span className="ml-1 font-semibold text-gray-700">
                      {test.retryCount}
                    </span>
                  </div>
                )}
                {test.createdAt && (
                  <div className="flex items-center">
                    <span className="font-medium">Started:</span>
                    <span className="ml-1 text-gray-600">
                      {new Date(test.createdAt).toLocaleString()}
                    </span>
                  </div>
                )}
              </div>
            </div>
            <div className="shrink-0 ml-4">
              <svg
                className="h-5 w-5 text-gray-400 group-hover:text-blue-600 group-hover:translate-x-1 transition-all"
                fill="none"
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth="2"
                viewBox="0 0 24 24"
                stroke="currentColor"
                aria-hidden="true"
              >
                <path d="M9 5l7 7-7 7" />
              </svg>
            </div>
          </div>
        </CardContent>
      </Card>
    </Link>
  );
};
