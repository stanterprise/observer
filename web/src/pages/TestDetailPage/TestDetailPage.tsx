import { useEffect, useState } from "react";
import { useParams, Link } from "react-router-dom";
import { apiUrl, config } from "@/lib/config";
import { Card, CardContent } from "@/components/Card";
import { useRefresh } from "@/lib/refresh";
import type { TestStatus } from "@/types/common";
import { ArrowLeft, AlertCircle, TrendingUp } from "lucide-react";
import StepContainer from "./StepContainer";
import type { Test } from "@/types/testCase";
import { getTestAttachments } from "@/lib/attemptUtils";
import AttemptsAccordion from "./AttemptsAccordion";
import AttachmentsCard from "./AttachmentsCard";
import SummaryCard from "./SummaryCard";

interface TestDetailResponse {
  runId: string;
  tests: Test[];
}

export function TestDetailPage() {
  const pollIntervalMs = config.pollingIntervalMs;
  const { autoRefreshEnabled } = useRefresh();
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
    if (!testId || !autoRefreshEnabled) return;
    const intervalId = window.setInterval(() => {
      fetchTestDetail(testId, { silent: true });
    }, pollIntervalMs);

    return () => {
      window.clearInterval(intervalId);
    };
  }, [testId, autoRefreshEnabled, pollIntervalMs, runId]);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-(--stitch-on-surface-muted)">
          Loading test details...
        </div>
      </div>
    );
  }

  if (error || !testDetail) {
    return (
      <div className="space-y-4">
        <Link
          to="/suite_runs"
          className="inline-flex items-center text-(--stitch-primary) hover:text-(--stitch-primary) focus:outline-none focus:ring-2 focus:ring-(--stitch-primary) focus:ring-offset-2 rounded-md px-2 py-1"
        >
          <ArrowLeft className="h-4 w-4 mr-2" />
          Back to Test Runs
        </Link>
        <Card>
          <CardContent className="py-12">
            <div className="flex flex-col items-center justify-center space-y-4">
              <AlertCircle className="h-16 w-16 text-(--status-failure)" />
              <div className="text-(--status-failure) text-center">
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
          className="inline-flex items-center text-(--stitch-primary) hover:text-(--stitch-primary) focus:outline-none focus:ring-2 focus:ring-(--stitch-primary) focus:ring-offset-2 rounded-md px-2 py-1"
        >
          <ArrowLeft className="h-4 w-4 mr-2" />
          Back to Test Runs
        </Link>
        <Card>
          <CardContent className="py-12">
            <div className="flex flex-col items-center justify-center space-y-4">
              <AlertCircle className="h-16 w-16 text-(--status-failure)" />
              <div className="text-(--status-failure) text-center">
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
            className="inline-flex items-center text-(--stitch-primary) hover:text-(--stitch-primary) focus:outline-none focus:ring-2 focus:ring-(--stitch-primary) focus:ring-offset-2 rounded-md p-1"
            aria-label="Back to test run"
          >
            <ArrowLeft className="h-5 w-5" />
          </Link>
          <div>
            <h1 className="text-3xl font-bold text-(--stitch-on-surface)">
              Test Case
            </h1>
            <p className="text-sm text-(--stitch-on-surface-muted) mt-1">
              Detailed view of test execution and steps
            </p>
          </div>
        </div>
        <Link
          to={`/tests/${test.id}/trends`}
          className="inline-flex items-center rounded-md px-4 py-2 text-white transition-opacity hover:opacity-90 focus:outline-none focus:ring-2 focus:ring-(--stitch-primary) focus:ring-offset-2"
          style={{
            background:
              "linear-gradient(135deg, var(--stitch-primary), var(--stitch-primary-end))",
          }}
        >
          <TrendingUp className="h-4 w-4 mr-2" />
          View Trends
        </Link>
      </div>

      {/* Test Case Summary Card */}
      <SummaryCard
        test={test}
        testStatus={testStatus}
        hasAttempts={hasAttempts}
        attempts={attempts}
        legacySteps={legacySteps}
      />

      {attachments.length > 0 && (
        <AttachmentsCard
          attachments={attachments}
          setActiveAttachment={setActiveAttachment}
        />
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
                <div className="bg-(--stitch-surface-card) rounded-lg p-6">
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
                <div className="bg-(--stitch-surface-card) rounded-lg p-6">
                  <h3 className="text-lg font-semibold text-(--stitch-on-surface) mb-3">
                    {activeAttachment.attachment.name || "Attachment"}
                  </h3>
                  {activeAttachment.contentText ? (
                    <pre className="text-sm text-(--stitch-on-surface-muted) whitespace-pre-wrap wrap-break-word max-h-[70vh] overflow-auto">
                      {activeAttachment.contentText}
                    </pre>
                  ) : (
                    <p className="text-sm text-(--stitch-on-surface-muted)">
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
