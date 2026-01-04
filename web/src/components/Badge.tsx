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
  status: TestStatus;
  className?: string;
  showIcon?: boolean;
}

export function Badge({ status, className, showIcon = true }: BadgeProps) {
  const statusConfig: Record<
    TestStatus,
    { color: string; icon: typeof CheckCircle; label: string }
  > = {
    PASSED: {
      color: "bg-green-100 text-green-800 border-green-200",
      icon: CheckCircle,
      label: "passed",
    },
    FAILED: {
      color: "bg-red-100 text-red-800 border-red-200",
      icon: XCircle,
      label: "failed",
    },
    SKIPPED: {
      color: "bg-gray-100 text-gray-800 border-gray-200",
      icon: MinusCircle,
      label: "skipped",
    },
    RUNNING: {
      color: "bg-blue-100 text-blue-800 border-blue-200",
      icon: Play,
      label: "running",
    },
    PENDING: {
      color: "bg-yellow-100 text-yellow-800 border-yellow-200",
      icon: Clock,
      label: "pending",
    },
    UNKNOWN: {
      color: "bg-gray-100 text-gray-800 border-gray-200",
      icon: Circle,
      label: "unknown",
    },
    BROKEN: {
      color: "bg-orange-100 text-orange-800 border-orange-200",
      icon: AlertTriangle,
      label: "broken",
    },
    TIMEDOUT: {
      color: "bg-purple-100 text-purple-800 border-purple-200",
      icon: Clock,
      label: "timed out",
    },
    INTERRUPTED: {
      color: "bg-pink-100 text-pink-800 border-pink-200",
      icon: Ban,
      label: "interrupted",
    },
    NOT_RUN: {
      color: "bg-gray-100 text-gray-800 border-gray-200",
      icon: MinusCircle,
      label: "not run",
    },
  };

  const config = statusConfig[status] || statusConfig.UNKNOWN;
  const Icon = config.icon;

  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium border",
        config.color,
        className
      )}
      role="status"
      aria-label={`Test status: ${config.label}`}
    >
      {showIcon && <Icon className="h-3.5 w-3.5" aria-hidden="true" />}
      <span className="capitalize">{config.label}</span>
    </span>
  );
}
