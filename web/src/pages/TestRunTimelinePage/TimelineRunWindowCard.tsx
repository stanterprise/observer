import { Card, CardContent } from "@/components/Card";
import { humanizeDuration, humanizeMilliseconds } from "@/utils/duration";
import { metricValue } from "./timelineFormatting";

type TimelineRunWindowCardProps = {
  spanMs: number;
  totalAttempts: number;
  runDuration?: number;
};

export function TimelineRunWindowCard({
  spanMs,
  totalAttempts,
  runDuration,
}: TimelineRunWindowCardProps) {
  return (
    <Card className="border border-(--stitch-outline) shadow-sm">
      <CardContent className="py-5">
        <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <h2 className="text-lg font-semibold text-(--stitch-on-surface)">
              Run Window
            </h2>
            <p className="mt-1 text-sm text-(--stitch-on-surface-muted)">
              The overall run currently spans {humanizeMilliseconds(spanMs)}
              across {metricValue(totalAttempts)} attempt windows.
            </p>
          </div>
          {typeof runDuration === "number" && runDuration > 0 && (
            <div className="rounded-xl border border-(--stitch-outline) bg-(--stitch-surface-low) px-4 py-3 text-sm text-(--stitch-on-surface-muted)">
              Run duration field:{" "}
              <span className="font-semibold text-(--stitch-on-surface)">
                {humanizeDuration(runDuration, 1_000_000_000)}
              </span>
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  );
}
