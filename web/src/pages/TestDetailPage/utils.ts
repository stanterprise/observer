import { apiUrl } from "@/lib/config";
import type { TestStatus } from "@/types/common";
import type { Step } from "@/types/testCase";

// Utility function to format duration
const formatDuration = (nanoseconds?: number) => {
  if (!nanoseconds) return "N/A";
  const milliseconds = nanoseconds / 1000000;
  if (milliseconds < 1000) return `${milliseconds.toFixed(0)}ms`;
  return `${(milliseconds / 1000).toFixed(2)}s`;
};

const formatBytes = (bytes?: number) => {
  if (!bytes && bytes !== 0) return "Unknown size";
  if (bytes < 1024) return `${bytes} B`;
  const kb = bytes / 1024;
  if (kb < 1024) return `${kb.toFixed(1)} KB`;
  const mb = kb / 1024;
  if (mb < 1024) return `${mb.toFixed(2)} MB`;
  const gb = mb / 1024;
  return `${gb.toFixed(2)} GB`;
};

const getAttachmentUrl = (attachment: Record<string, any>) => {
  if (attachment.storage_key) {
    return apiUrl(`/attachments/${encodeURIComponent(attachment.storage_key)}`);
  }
  if (attachment.storage_uri) {
    return attachment.storage_uri as string;
  }
  if (attachment.uri) {
    return attachment.uri as string;
  }
  return undefined;
};

const getInlineMediaUrl = (attachment: Record<string, any>) => {
  const mimeType = attachment.mime_type || "application/octet-stream";
  const content = attachment.content;
  if (!content || typeof content !== "string") return undefined;
  if (attachment.content_encoding === "base64") {
    return `data:${mimeType};base64,${content}`;
  }
  return undefined;
};

const decodeBase64ToUtf8 = (value: string) => {
  try {
    const binary = atob(value.replace(/\s+/g, ""));
    const bytes = Uint8Array.from(binary, (char) => char.charCodeAt(0));
    return new TextDecoder("utf-8").decode(bytes);
  } catch {
    return value;
  }
};

const isLikelyBase64 = (value: string) => {
  if (value.length < 16 || value.length % 4 !== 0) return false;
  return /^[A-Za-z0-9+/=\s]+$/.test(value);
};

const decodeInlineContent = (attachment: Record<string, any>) => {
  const content = attachment.content;
  if (!content || typeof content !== "string") return "";
  if (attachment.content_encoding === "base64") {
    return decodeBase64ToUtf8(content);
  }
  const mimeType = attachment.mime_type || "";
  const isTextual =
    mimeType.startsWith("text/") ||
    mimeType === "application/json" ||
    mimeType === "text/csv";
  if (isTextual && isLikelyBase64(content)) {
    return decodeBase64ToUtf8(content);
  }
  return content;
};

// Utility function to convert status to TestStatus
function getTestStatus(status: string | number | undefined): TestStatus {
  if (!status) return "PENDING";
  if (typeof status === "number") {
    const statusMap: Record<number, TestStatus> = {
      0: "UNKNOWN",
      1: "PASSED",
      2: "FAILED",
      3: "SKIPPED",
      4: "BROKEN",
      5: "TIMEDOUT",
      6: "INTERRUPTED",
    };
    return statusMap[status] || "UNKNOWN";
  }
  const upperStatus = status.toUpperCase();
  if (upperStatus === "PASSED") return "PASSED";
  if (upperStatus === "FAILED") return "FAILED";
  if (upperStatus === "RUNNING") return "RUNNING";
  if (upperStatus === "SKIPPED") return "SKIPPED";
  if (upperStatus === "BROKEN") return "BROKEN";
  if (upperStatus === "TIMEDOUT") return "TIMEDOUT";
  if (upperStatus === "INTERRUPTED") return "INTERRUPTED";
  if (upperStatus === "PENDING") return "PENDING";
  return upperStatus as TestStatus;
}

const countNestedSteps = (steps?: Step[]): number => {
  return (steps || []).reduce(
    (sum, step) => sum + 1 + countNestedSteps(step.steps),
    0,
  );
};

const countExpandableSteps = (steps?: Step[]): number => {
  return (steps || []).reduce((sum, step) => {
    const childCount = step.steps && step.steps.length > 0 ? 1 : 0;
    return sum + childCount + countExpandableSteps(step.steps);
  }, 0);
};

const formatStepLocation = (location?: string) => {
  if (!location) return undefined;
  const normalized = location.replace(/\\/g, "/");
  const lastSlash = normalized.lastIndexOf("/");
  return lastSlash >= 0 ? normalized.slice(lastSlash + 1) : normalized;
};

export {
  countExpandableSteps,
  countNestedSteps,
  formatStepLocation,
  formatDuration,
  formatBytes,
  getAttachmentUrl,
  getInlineMediaUrl,
  decodeInlineContent,
  getTestStatus,
};
