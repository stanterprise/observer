import { Clock } from "lucide-react";
import { Badge } from "../../components/Badge";
import { Link } from "react-router-dom";
import { Card, CardContent } from "@/components/Card";

import type { TestCase } from "./types";
import { getTestStatus, formatDuration } from "./utils";

type TestRecordProps = {
  test: TestCase;
  runId: string;
};

export default ({ test, runId }: TestRecordProps) => {
  return (
    <Link key={test.ID} to={`/suite_runs/${runId}/tests/${test.ID}`}>
      <Card className="hover:shadow-md transition-all duration-200 cursor-pointer hover:border-blue-300">
        <CardContent className="py-4">
          <div className="flex items-center justify-between">
            <div className="flex-1 min-w-0">
              <div className="flex items-center space-x-3 mb-2">
                <Badge status={getTestStatus(test.Status)} />
                <h3 className="text-base font-medium text-gray-900 truncate">
                  {test.Title || test.ID}
                </h3>
              </div>
              <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-gray-500">
                <div className="flex items-center">
                  <Clock className="h-4 w-4 mr-1" />
                  <span className="font-medium">Duration:</span>
                  <span className="ml-1">{formatDuration(test.Duration)}</span>
                </div>
                {test.RetryCount !== undefined && test.RetryCount > 0 && (
                  <div className="flex items-center">
                    <span className="font-medium">Retries:</span>
                    <span className="ml-1">{test.RetryCount}</span>
                  </div>
                )}
                <div className="flex items-center">
                  <span className="font-medium">Started:</span>
                  <span className="ml-1">
                    {new Date(test.CreatedAt).toLocaleString()}
                  </span>
                </div>
              </div>
            </div>
            <div className="shrink-0 ml-4">
              <svg
                className="h-5 w-5 text-gray-400"
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
