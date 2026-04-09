import { Clock } from "lucide-react";
import { Badge } from "../../components/Badge";
import { TagList } from "../../components/TagList";
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
    <Link
      key={test.id}
      to={`/suite_runs/${runId}/tests/${test.id}`}
      className="block rounded-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-(--stitch-primary) focus-visible:ring-offset-2 focus-visible:ring-offset-(--stitch-background)"
    >
      <Card className="group cursor-pointer transition-colors duration-200 hover:border-(--status-running-border) hover:bg-(--stitch-surface-low)">
        <CardContent className="py-4">
          <div className="flex items-center justify-between">
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-3 mb-2">
                <Badge status={getTestStatus(test.status)} />
                <h3 className="truncate text-base font-medium text-(--stitch-on-surface) transition-colors group-hover:text-(--stitch-primary)">
                  {test.title || test.id}
                </h3>
              </div>
              <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-(--stitch-on-surface-muted)">
                <div className="flex items-center">
                  <Clock className="mr-1 h-4 w-4 text-(--stitch-on-surface-subtle)" />
                  <span className="font-medium">Duration:</span>
                  <span className="ml-1 font-semibold text-(--stitch-on-surface)">
                    {formatDuration(test.duration)}
                  </span>
                </div>
                {(test.attempts?.length ?? 0) > 1 && (
                  <div className="flex items-center">
                    <span className="font-medium">Retries:</span>
                    <span className="ml-1 font-semibold text-(--stitch-on-surface)">
                      {(test.attempts?.length ?? 0) - 1}
                    </span>
                  </div>
                )}
                {test.createdAt && (
                  <div className="flex items-center">
                    <span className="font-medium">Started:</span>
                    <span className="ml-1 text-(--stitch-on-surface-muted)">
                      {new Date(test.createdAt).toLocaleString()}
                    </span>
                  </div>
                )}
              </div>
              {test.tags && test.tags.length > 0 && (
                <div className="mt-2">
                  <TagList tags={test.tags} />
                </div>
              )}
            </div>
            <div className="shrink-0 ml-4">
              <svg
                className="h-5 w-5 text-(--stitch-on-surface-subtle) transition-all group-hover:translate-x-1 group-hover:text-(--stitch-primary)"
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
