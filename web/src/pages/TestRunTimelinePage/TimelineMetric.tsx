import { Card, CardContent } from "@/components/Card";

type TimelineMetricProps = {
  label: string;
  value: string;
  detail: string;
};

export function TimelineMetric({ label, value, detail }: TimelineMetricProps) {
  return (
    <Card className="border border-(--stitch-outline) shadow-sm">
      <CardContent className="space-y-1 py-5">
        <p className="text-[11px] font-semibold uppercase tracking-[0.16em] text-(--stitch-on-surface-subtle)">
          {label}
        </p>
        <p className="text-2xl font-semibold text-(--stitch-on-surface)">
          {value}
        </p>
        <p className="text-sm text-(--stitch-on-surface-muted)">{detail}</p>
      </CardContent>
    </Card>
  );
}
