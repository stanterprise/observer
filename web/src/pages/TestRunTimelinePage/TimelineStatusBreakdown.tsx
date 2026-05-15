import { Card, CardContent } from "@/components/Card";
import type { TestStatus } from "@/types/common";
import { metricValue, statusStyles } from "./timelineFormatting";

type TimelineStatusBreakdownProps = {
  orderedStatuses: TestStatus[];
  statusCounts: Partial<Record<TestStatus, number>>;
  approximateCount: number;
  syntheticCount: number;
};

export function TimelineStatusBreakdown({
  orderedStatuses,
  statusCounts,
  approximateCount,
  syntheticCount,
}: TimelineStatusBreakdownProps) {
  return (
    <Card className="border border-(--stitch-outline) shadow-sm">
      <CardContent className="space-y-4 py-5">
        <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
          <div className="space-y-1">
            <h2 className="text-lg font-semibold text-(--stitch-on-surface)">
              Attempt Status Breakdown
            </h2>
            <p className="text-sm text-(--stitch-on-surface-muted)">
              Click any bar in the chart to open the underlying test attempt
              details.
            </p>
          </div>

          {(approximateCount > 0 || syntheticCount > 0) && (
            <div className="rounded-xl border border-(--status-warning-border) bg-(--status-warning-soft) px-4 py-3 text-sm text-(--status-warning)">
              {approximateCount > 0 && (
                <p>
                  {metricValue(approximateCount)} attempt
                  {approximateCount === 1 ? " used" : "s used"} inferred start
                  or end bounds.
                </p>
              )}
              {syntheticCount > 0 && (
                <p>
                  {metricValue(syntheticCount)} legacy test
                  {syntheticCount === 1 ? " was" : "s were"} plotted from
                  aggregated test timing because attempt rows were missing.
                </p>
              )}
            </div>
          )}
        </div>

        <div className="flex flex-wrap gap-2">
          {orderedStatuses.map((status) => (
            <span
              key={status}
              className="inline-flex items-center gap-2 rounded-full border px-3 py-1.5 text-sm font-medium"
              style={statusStyles(status)}
            >
              <span className="capitalize">
                {status.toLowerCase().replace("_", " ")}
              </span>
              <span className="rounded-full bg-black/5 px-2 py-0.5 text-xs font-semibold text-current">
                {metricValue(statusCounts[status] ?? 0)}
              </span>
            </span>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}
