import { cn } from "../lib/utils";
import type { TestStatus } from "@/types/common";
import {
  CheckCircle,
  XCircle,
  Circle,
  Play,
  Clock,
  AlertTriangle,
  MinusCircle,
  Ban,
} from "lucide-react";

interface BadgeProps {
  status: TestStatus | "COMPLETED";
  className?: string;
  showIcon?: boolean;
}

export function Badge({ status, className, showIcon = true }: BadgeProps) {
  const statusConfig: Record<
    TestStatus | "COMPLETED",
    {
      colors: {
        backgroundColor: string;
        borderColor: string;
        color: string;
      };
      icon: typeof CheckCircle;
      label: string;
    }
  > = {
    PASSED: {
      colors: {
        backgroundColor: "var(--status-success-soft)",
        borderColor: "var(--status-success-border)",
        color: "var(--status-success)",
      },
      icon: CheckCircle,
      label: "passed",
    },
    FLAKY: {
      colors: {
        backgroundColor: "var(--status-warning-soft)",
        borderColor: "var(--status-warning-border)",
        color: "var(--status-warning)",
      },
      icon: AlertTriangle,
      label: "flaky",
    },
    FAILED: {
      colors: {
        backgroundColor: "var(--status-failure-soft)",
        borderColor: "var(--status-failure-border)",
        color: "var(--status-failure)",
      },
      icon: XCircle,
      label: "failed",
    },
    SKIPPED: {
      colors: {
        backgroundColor: "var(--status-neutral-soft)",
        borderColor: "var(--status-neutral-border)",
        color: "var(--status-neutral)",
      },
      icon: MinusCircle,
      label: "skipped",
    },
    RUNNING: {
      colors: {
        backgroundColor: "var(--status-running-soft)",
        borderColor: "var(--status-running-border)",
        color: "var(--status-running)",
      },
      icon: Play,
      label: "running",
    },
    PENDING: {
      colors: {
        backgroundColor: "var(--status-warning-soft)",
        borderColor: "var(--status-warning-border)",
        color: "var(--status-warning)",
      },
      icon: Clock,
      label: "pending",
    },
    UNKNOWN: {
      colors: {
        backgroundColor: "var(--status-neutral-soft)",
        borderColor: "var(--status-neutral-border)",
        color: "var(--status-neutral)",
      },
      icon: Circle,
      label: "unknown",
    },
    BROKEN: {
      colors: {
        backgroundColor: "var(--status-broken-soft)",
        borderColor: "var(--status-broken-border)",
        color: "var(--status-broken)",
      },
      icon: AlertTriangle,
      label: "broken",
    },
    TIMEDOUT: {
      colors: {
        backgroundColor: "var(--status-timedout-soft)",
        borderColor: "var(--status-timedout-border)",
        color: "var(--status-timedout)",
      },
      icon: Clock,
      label: "timed out",
    },
    INTERRUPTED: {
      colors: {
        backgroundColor: "var(--status-interrupted-soft)",
        borderColor: "var(--status-interrupted-border)",
        color: "var(--status-interrupted)",
      },
      icon: Ban,
      label: "interrupted",
    },
    NOT_RUN: {
      colors: {
        backgroundColor: "var(--status-neutral-soft)",
        borderColor: "var(--status-neutral-border)",
        color: "var(--status-neutral)",
      },
      icon: MinusCircle,
      label: "not run",
    },
    COMPLETED: {
      colors: {
        backgroundColor: "var(--status-success-soft)",
        borderColor: "var(--status-success-border)",
        color: "var(--status-success)",
      },
      icon: CheckCircle,
      label: "completed",
    },
  };

  const config = statusConfig[status] || statusConfig.UNKNOWN;
  const Icon = config.icon;

  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium border",
        className,
      )}
      style={config.colors}
      role="status"
      aria-label={`Test status: ${config.label}`}
    >
      {showIcon && <Icon className="h-3.5 w-3.5" aria-hidden="true" />}
      <span className="capitalize">{config.label}</span>
    </span>
  );
}
