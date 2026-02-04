import { useEffect, useState } from "react";
import { useParams, Link } from "react-router-dom";
import { apiUrl } from "@/lib/config";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/Card";
import { Badge } from "@/components/Badge";
import { TagList } from "@/components/TagList";
import type { TestStatus } from "@/types/common";
import {
  ArrowLeft,
  AlertCircle,
  TrendingUp,
  ChevronDown,
  ChevronRight,
} from "lucide-react";
import StepContainer from "./StepContainer";
import type { Attempt, Test } from "@/types/testCase";
import { getTestAttachments } from "@/lib/attemptUtils";

interface TestDetailResponse {
  runId: string;
  tests: Test[];
}

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

// Helper component for rendering attempts in an accordion
function AttemptsAccordion({
  test,
  attempts,
}: {
  test: Test;
  attempts: Attempt[];
}) {
  const [openAttempt, setOpenAttempt] = useState<number>(
    test.retryIndex ?? attempts.length - 1,
  );

  const getAttemptStatus = (attempt: Attempt): TestStatus => {
    if (!attempt.status) return "PENDING";
    return getTestStatus(attempt.status);
  };

  const toggleAttempt = (retryIndex: number) => {
    setOpenAttempt(openAttempt === retryIndex ? -1 : retryIndex);
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-lg">Test Execution Attempts</CardTitle>
        <p className="text-sm text-gray-600 mt-1">
          {attempts.length} attempt{attempts.length > 1 ? "s" : ""} recorded
        </p>
      </CardHeader>
      <CardContent>
        <div className="space-y-3">
          {attempts.map((attempt) => {
            const isOpen = openAttempt === attempt.retryIndex;
            const attemptStatus = getAttemptStatus(attempt);
            const attemptSteps = attempt.steps || [];

            return (
              <div
                key={attempt.retryIndex}
                className="border border-gray-200 rounded-lg overflow-hidden"
              >
                {/* Accordion Header */}
                <button
                  onClick={() => toggleAttempt(attempt.retryIndex)}
                  className="w-full flex items-center justify-between p-4 bg-gray-50 hover:bg-gray-100 transition-colors"
                >
                  <div className="flex items-center gap-3">
                    {isOpen ? (
                      <ChevronDown className="h-5 w-5 text-gray-600" />
                    ) : (
                      <ChevronRight className="h-5 w-5 text-gray-600" />
                    )}
                    <div className="text-left">
                      <div className="flex items-center gap-2">
                        <span className="font-semibold text-gray-900">
                          Attempt {attempt.retryIndex + 1}
                        </span>
                        {attempt.retryIndex === test.retryIndex && (
                          <span className="text-xs px-2 py-0.5 bg-blue-100 text-blue-700 rounded-full font-medium">
                            Current
                          </span>
                        )}
                      </div>
                      <div className="text-sm text-gray-600 mt-1">
                        {attemptSteps.length} step
                        {attemptSteps.length !== 1 ? "s" : ""}
                        {attempt.startTime && (
                          <span className="ml-2">
                            • Started{" "}
                            {new Date(attempt.startTime).toLocaleString()}
                          </span>
                        )}
                      </div>
                    </div>
                  </div>
                  <Badge status={attemptStatus} />
                </button>

                {/* Accordion Body */}
                {isOpen && (
                  <div className="p-4 bg-white border-t border-gray-200">
                    {/* Attempt Info */}
                    <div className="mb-4 p-3 bg-gray-50 rounded-lg">
                      <dl className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
                        <div>
                          <dt className="text-gray-600 font-medium">Status</dt>
                          <dd className="mt-1">
                            <Badge status={attemptStatus} />
                          </dd>
                        </div>
                        {attempt.startTime && (
                          <div>
                            <dt className="text-gray-600 font-medium">
                              Started
                            </dt>
                            <dd className="text-gray-900 mt-1">
                              {new Date(attempt.startTime).toLocaleString()}
                            </dd>
                          </div>
                        )}
                        {attempt.endTime && (
                          <div>
                            <dt className="text-gray-600 font-medium">
                              Finished
                            </dt>
                            <dd className="text-gray-900 mt-1">
                              {new Date(attempt.endTime).toLocaleString()}
                            </dd>
                          </div>
                        )}
                        {attempt.duration !== undefined && (
                          <div>
                            <dt className="text-gray-600 font-medium">
                              Duration
                            </dt>
                            <dd className="text-gray-900 mt-1 font-semibold">
                              {formatDuration(attempt.duration)}
                            </dd>
                          </div>
                        )}
                      </dl>
                    </div>

                    {/* Error Display */}
                    {(attempt.errorMessage ||
                      (attempt.errors && attempt.errors.length > 0)) && (
                      <div className="mb-4 p-4 bg-red-50 border border-red-200 rounded-lg">
                        <h4 className="text-sm font-semibold text-red-800 mb-2">
                          Error Details
                        </h4>
                        {attempt.errorMessage && (
                          <p className="text-sm text-red-700 mb-2">
                            {attempt.errorMessage}
                          </p>
                        )}
                        {attempt.stackTrace && (
                          <pre className="text-xs text-red-600 bg-red-100 p-2 rounded overflow-x-auto whitespace-pre-wrap">
                            {attempt.stackTrace}
                          </pre>
                        )}
                        {attempt.errors && attempt.errors.length > 0 && (
                          <ul className="text-sm text-red-700 list-disc list-inside space-y-1">
                            {attempt.errors.map((err, idx) => (
                              <li key={idx}>
                                {typeof err === "string"
                                  ? err
                                  : JSON.stringify(err)}
                              </li>
                            ))}
                          </ul>
                        )}
                      </div>
                    )}

                    {/* Steps */}
                    {attemptSteps.length > 0 ? (
                      <StepContainer
                        test={{
                          id: test.id,
                          runId: test.runId,
                          title: `Attempt ${attempt.retryIndex + 1}`,
                          status: attemptStatus,
                          steps: attemptSteps.map((step) => ({
                            id: step.id,
                            runId: step.runId || test.runId,
                            testCaseRunId: step.testCaseRunId,
                            parentStepId:
                              step.parentStepId && step.parentStepId !== ""
                                ? step.parentStepId
                                : undefined,
                            status: getTestStatus(step.status),
                            category: step.category,
                            title: step.title,
                            startedAt: step.startTime || step.createdAt,
                            finishedAt: step.updatedAt,
                            error: (step as any).error,
                            errors: (step as any).errors,
                            metadata: (step as any).metadata,
                            duration: (step as any).duration,
                            location: (step as any).location,
                          })),
                        }}
                      />
                    ) : (
                      <div className="text-center py-8 text-gray-500">
                        <p className="text-sm">
                          No steps recorded for this attempt
                        </p>
                      </div>
                    )}
                  </div>
                )}
              </div>
            );
          })}
        </div>
      </CardContent>
    </Card>
  );
}

export function TestDetailPage() {
  const pollIntervalMs = 10_000;
  const { runId, testId } = useParams<{ runId: string; testId: string }>();
  const [testDetail, setTestDetail] = useState<TestDetailResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [activeAttachment, setActiveAttachment] = useState<{
    attachment: Record<string, any>;
    url?: string;
    inlineUrl?: string;
    isImage: boolean;
    isVideo: boolean;
    isAudio: boolean;
    contentText: string;
  } | null>(null);

  const fetchTestDetail = async (
    id: string,
    options?: { silent?: boolean },
  ) => {
    const silent = options?.silent ?? false;
    try {
      if (!silent) {
        setLoading(true);
      }
      if (!runId) {
        throw new Error("Run ID is required");
      }
      const response = await fetch(apiUrl(`/runs/${runId}/tests/${id}`));
      if (!response.ok) {
        throw new Error(`Failed to fetch test details: ${response.statusText}`);
      }
      const data = await response.json();
      setTestDetail(data);
      setError(null);
    } catch (err) {
      console.error("Error fetching test details:", err);
      setError(
        err instanceof Error ? err.message : "Failed to fetch test details",
      );
    } finally {
      if (!silent) {
        setLoading(false);
      }
    }
  };

  useEffect(() => {
    if (testId) {
      fetchTestDetail(testId);
    }
  }, [testId]);

  useEffect(() => {
    if (!testId) return;
    const intervalId = window.setInterval(() => {
      fetchTestDetail(testId, { silent: true });
    }, pollIntervalMs);

    return () => {
      window.clearInterval(intervalId);
    };
  }, [testId, pollIntervalMs, runId]);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-gray-600">Loading test details...</div>
      </div>
    );
  }

  if (error || !testDetail) {
    return (
      <div className="space-y-4">
        <Link
          to="/suite_runs"
          className="inline-flex items-center text-blue-600 hover:text-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 rounded-md px-2 py-1"
        >
          <ArrowLeft className="h-4 w-4 mr-2" />
          Back to Test Runs
        </Link>
        <Card>
          <CardContent className="py-12">
            <div className="flex flex-col items-center justify-center space-y-4">
              <AlertCircle className="h-16 w-16 text-red-500" />
              <div className="text-red-600 text-center">
                <p className="font-semibold">
                  Error: {error || "Test not found"}
                </p>
                <p className="text-sm mt-1">
                  The test case you're looking for doesn't exist or couldn't be
                  loaded.
                </p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  const getTestStatus = (status: string): TestStatus => {
    const statusMap: Record<string, TestStatus> = {
      PASSED: "PASSED",
      FAILED: "FAILED",
      SKIPPED: "SKIPPED",
      RUNNING: "RUNNING",
      UNKNOWN: "UNKNOWN",
      BROKEN: "BROKEN",
      TIMEDOUT: "TIMEDOUT",
      INTERRUPTED: "INTERRUPTED",
    };
    return (statusMap[status] || "UNKNOWN") as TestStatus;
  };

  // Extract test from the tests array (API returns array with single element)
  const test = testDetail.tests[0];
  if (!test) {
    return (
      <div className="space-y-4">
        <Link
          to="/suite_runs"
          className="inline-flex items-center text-blue-600 hover:text-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 rounded-md px-2 py-1"
        >
          <ArrowLeft className="h-4 w-4 mr-2" />
          Back to Test Runs
        </Link>
        <Card>
          <CardContent className="py-12">
            <div className="flex flex-col items-center justify-center space-y-4">
              <AlertCircle className="h-16 w-16 text-red-500" />
              <div className="text-red-600 text-center">
                <p className="font-semibold">Test data not found</p>
                <p className="text-sm mt-1">
                  The test case data is missing or invalid.
                </p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  const testStatus = getTestStatus(test.status);

  // Use attempts if available, fallback to legacy steps
  const hasAttempts = test.attempts && test.attempts.length > 0;
  const attempts = test.attempts || [];
  const legacySteps = test.steps || [];
  const attachments = getTestAttachments(test);

  console.log(
    "Rendering TestDetailPage for test:",
    test.id,
    "hasAttempts:",
    hasAttempts,
    "attempts:",
    attempts,
    "legacySteps:",
    legacySteps,
  );
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-4">
          <Link
            to={`/suite_runs/${test.runId}`}
            className="inline-flex items-center text-blue-600 hover:text-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 rounded-md p-1"
            aria-label="Back to test run"
          >
            <ArrowLeft className="h-5 w-5" />
          </Link>
          <div>
            <h1 className="text-3xl font-bold text-gray-900">Test Case</h1>
            <p className="text-sm text-gray-600 mt-1">
              Detailed view of test execution and steps
            </p>
          </div>
        </div>
        <Link
          to={`/tests/${test.id}/trends`}
          className="inline-flex items-center px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 transition-colors"
        >
          <TrendingUp className="h-4 w-4 mr-2" />
          View Trends
        </Link>
      </div>

      {/* Test Case Summary Card */}
      <Card>
        <CardHeader>
          <div className="flex items-start justify-between">
            <div className="flex-1 min-w-0">
              <CardTitle className="text-xl mb-2 wrap-break-word">
                {test.title || test.id}
              </CardTitle>
              <p className="text-sm text-gray-500 font-mono">{test.id}</p>
            </div>
            <Badge
              status={testStatus}
              className="text-lg px-4 py-2 shrink-0 ml-4"
            />
          </div>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div>
              <h3 className="text-sm font-semibold text-gray-700 mb-3 uppercase tracking-wide">
                Test Information
              </h3>
              <dl className="space-y-3 text-sm">
                <div className="flex justify-between items-start">
                  <dt className="text-gray-600 font-medium">Test ID:</dt>
                  <dd className="font-mono text-gray-900 text-right break-all ml-4">
                    {test.id}
                  </dd>
                </div>
                <div className="flex justify-between items-start">
                  <dt className="text-gray-600 font-medium">Run ID:</dt>
                  <dd className="text-right ml-4">
                    <Link
                      to={`/suite_runs/${test.runId}`}
                      className="font-mono text-blue-600 hover:underline focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 rounded"
                    >
                      {test.runId}
                    </Link>
                  </dd>
                </div>
                <div className="flex justify-between items-start">
                  <dt className="text-gray-600 font-medium">Duration:</dt>
                  <dd className="text-gray-900 font-semibold text-right ml-4">
                    {formatDuration(test.duration)}
                  </dd>
                </div>
                {test.retryCount !== undefined && test.retryCount > 0 && (
                  <div className="flex justify-between items-start">
                    <dt className="text-gray-600 font-medium">Retries:</dt>
                    <dd className="text-gray-900 text-right ml-4">
                      {test.retryIndex !== undefined
                        ? `${test.retryIndex} / ${test.retryCount}`
                        : test.retryCount}
                    </dd>
                  </div>
                )}
                {test.timeout !== undefined && (
                  <div className="flex justify-between items-start">
                    <dt className="text-gray-600 font-medium">Timeout:</dt>
                    <dd className="text-gray-900 text-right ml-4">
                      {test.timeout}ms
                    </dd>
                  </div>
                )}
              </dl>
            </div>
            <div>
              <h3 className="text-sm font-semibold text-gray-700 mb-3 uppercase tracking-wide">
                Execution Timeline
              </h3>
              <dl className="space-y-3 text-sm">
                <div className="flex justify-between items-start">
                  <dt className="text-gray-600 font-medium">Started:</dt>
                  <dd className="text-gray-900 text-right ml-4">
                    {new Date(test.createdAt!).toLocaleString()}
                  </dd>
                </div>
                <div className="flex justify-between items-start">
                  <dt className="text-gray-600 font-medium">Last Updated:</dt>
                  <dd className="text-gray-900 text-right ml-4">
                    {new Date(test.updatedAt!).toLocaleString()}
                  </dd>
                </div>
                <div className="flex justify-between items-start">
                  <dt className="text-gray-600 font-medium">Total Steps:</dt>
                  <dd className="text-gray-900 font-semibold text-right ml-4">
                    {hasAttempts
                      ? attempts.reduce(
                          (sum, attempt) => sum + (attempt.steps?.length || 0),
                          0,
                        )
                      : legacySteps.length}
                  </dd>
                </div>
                {hasAttempts && attempts.length > 1 && (
                  <div className="flex justify-between items-start">
                    <dt className="text-gray-600 font-medium">
                      Total Attempts:
                    </dt>
                    <dd className="text-gray-900 font-semibold text-right ml-4">
                      {attempts.length}
                    </dd>
                  </div>
                )}
              </dl>
            </div>
          </div>

          {/* Metadata Section */}
          {test.metadata && Object.keys(test.metadata).length > 0 && (
            <div className="mt-6 pt-6 border-t border-gray-200">
              <h3 className="text-sm font-semibold text-gray-700 mb-3 uppercase tracking-wide">
                Metadata
              </h3>
              <div className="bg-gray-50 rounded-lg p-4 border border-gray-200">
                <pre className="text-xs text-gray-800 overflow-x-auto whitespace-pre-wrap wrap-break-word">
                  {JSON.stringify(test.metadata, null, 2)}
                </pre>
              </div>
            </div>
          )}

          {/* Tags Section */}
          {test.tags && test.tags.length > 0 && (
            <div className="mt-6 pt-6 border-t border-gray-200">
              <h3 className="text-sm font-semibold text-gray-700 mb-3 uppercase tracking-wide">
                Tags
              </h3>
              <TagList tags={test.tags} />
            </div>
          )}
        </CardContent>
      </Card>

      {attachments.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">Attachments</CardTitle>
            <p className="text-sm text-gray-600 mt-1">
              {attachments.length} attachment
              {attachments.length > 1 ? "s" : ""} associated with this test
            </p>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              {attachments.map((attachment, index) => {
                const url = getAttachmentUrl(attachment);
                const storageType = attachment.storage || "inline";
                const content = decodeInlineContent(attachment);
                const previewLimit = 400;
                const preview = content ? content.slice(0, previewLimit) : "";
                const isImage =
                  typeof attachment.mime_type === "string" &&
                  attachment.mime_type.startsWith("image/");
                const isVideo =
                  typeof attachment.mime_type === "string" &&
                  attachment.mime_type.startsWith("video/");
                const isAudio =
                  typeof attachment.mime_type === "string" &&
                  attachment.mime_type.startsWith("audio/");
                const inlineMediaUrl = getInlineMediaUrl(attachment);
                const mediaUrl = url || inlineMediaUrl;
                const canPreview = isImage || isVideo || isAudio || content;
                const handleAttachmentClick = () => {
                  if (!canPreview) return;
                  setActiveAttachment({
                    attachment,
                    url,
                    inlineUrl: inlineMediaUrl,
                    isImage,
                    isVideo,
                    isAudio,
                    contentText: content,
                  });
                };

                return (
                  <div
                    key={`${attachment.storage_key || attachment.uri || attachment.name || "attachment"}-${index}`}
                    className="border border-gray-200 rounded-lg p-4 bg-white"
                  >
                    <div className="flex items-start justify-between gap-4">
                      <div className="min-w-0">
                        <p className="font-medium text-gray-900 truncate">
                          {attachment.name || "Attachment"}
                        </p>
                        <p className="text-xs text-gray-500 mt-1">
                          {attachment.mime_type || "unknown"} •{" "}
                          {formatBytes(attachment.size)} • {storageType}
                        </p>
                      </div>
                      <div className="flex items-center gap-2">
                        {url && (
                          <a
                            href={url}
                            target="_blank"
                            rel="noreferrer"
                            className="inline-flex items-center px-3 py-1.5 text-sm font-medium text-blue-600 border border-blue-200 rounded-md hover:bg-blue-50 transition-colors"
                          >
                            Open
                          </a>
                        )}
                        {canPreview && (
                          <button
                            type="button"
                            onClick={handleAttachmentClick}
                            className="inline-flex items-center px-3 py-1.5 text-sm font-medium text-gray-900 border border-gray-200 rounded-md hover:bg-gray-50 transition-colors"
                          >
                            View
                          </button>
                        )}
                      </div>
                    </div>
                    {isImage && mediaUrl && (
                      <button
                        type="button"
                        onClick={handleAttachmentClick}
                        className="mt-3 block"
                      >
                        <img
                          src={mediaUrl}
                          alt={attachment.name || "Attachment"}
                          className="max-h-64 rounded-md border border-gray-200 bg-gray-50"
                        />
                      </button>
                    )}
                    {storageType === "inline" && !isImage && preview && (
                      <div className="mt-3 bg-gray-50 border border-gray-200 rounded-md p-3">
                        <pre className="text-xs text-gray-700 whitespace-pre-wrap wrap-break-word">
                          {preview}
                        </pre>
                        {content.length > previewLimit && (
                          <p className="text-xs text-gray-500 mt-2">
                            Preview truncated
                          </p>
                        )}
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          </CardContent>
        </Card>
      )}

      {activeAttachment && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 p-4"
          onClick={() => setActiveAttachment(null)}
          role="dialog"
          aria-modal="true"
        >
          <div className="max-h-full w-full max-w-5xl">
            {activeAttachment.isImage &&
              (activeAttachment.url || activeAttachment.inlineUrl) && (
                <img
                  src={activeAttachment.url || activeAttachment.inlineUrl}
                  alt={activeAttachment.attachment.name || "Attachment"}
                  className="max-h-[80vh] w-full object-contain rounded-lg"
                />
              )}
            {activeAttachment.isVideo &&
              (activeAttachment.url || activeAttachment.inlineUrl) && (
                <video
                  src={activeAttachment.url || activeAttachment.inlineUrl}
                  controls
                  className="max-h-[80vh] w-full rounded-lg"
                />
              )}
            {activeAttachment.isAudio &&
              (activeAttachment.url || activeAttachment.inlineUrl) && (
                <div className="bg-white rounded-lg p-6">
                  <audio
                    src={activeAttachment.url || activeAttachment.inlineUrl}
                    controls
                    className="w-full"
                  />
                </div>
              )}
            {!activeAttachment.isImage &&
              !activeAttachment.isVideo &&
              !activeAttachment.isAudio && (
                <div className="bg-white rounded-lg p-6">
                  <h3 className="text-lg font-semibold text-gray-900 mb-3">
                    {activeAttachment.attachment.name || "Attachment"}
                  </h3>
                  {activeAttachment.contentText ? (
                    <pre className="text-sm text-gray-700 whitespace-pre-wrap wrap-break-word max-h-[70vh] overflow-auto">
                      {activeAttachment.contentText}
                    </pre>
                  ) : (
                    <p className="text-sm text-gray-600">
                      Preview not available.
                    </p>
                  )}
                </div>
              )}
          </div>
        </div>
      )}

      {/* Test Execution Steps - Attempts Accordion */}
      {hasAttempts ? (
        <AttemptsAccordion test={test} attempts={attempts} />
      ) : (
        <StepContainer
          test={{
            id: test.id,
            runId: test.runId,
            title: test.title || test.id,
            status: testStatus,
            steps: legacySteps.map((step) => ({
              id: step.id,
              runId: step.runId || test.runId,
              testCaseRunId: step.testCaseRunId,
              parentStepId:
                step.parentStepId && step.parentStepId !== ""
                  ? step.parentStepId
                  : undefined,
              status: step.status,
              category: step.category,
              title: step.title,
              startedAt: step.startTime || step.createdAt,
              finishedAt: step.updatedAt,
              error: step.error,
              errors: step.errors,
              metadata: step.metadata,
              duration: step.duration,
              location: step.location,
            })),
          }}
        />
      )}
    </div>
  );
}
