import { useEffect, useState } from "react";
import { useParams, Link } from "react-router-dom";
import { apiUrl } from "@/lib/config";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/Card";
import { Badge } from "@/components/Badge";
import type {
  WebSocketEvent,
  WebSocketTestData,
  WebSocketStepData,
} from "@/types";
import type { TestStatus } from "@/types/common";
import { ArrowLeft, AlertCircle } from "lucide-react";
import StepContainer from "./StepContainer";

interface TestDetailPageProps {
  onWebSocketEvent?: WebSocketEvent | null;
}

interface StepDetail {
  id: string;
  runId?: string;
  testCaseRunId: string;
  parentStepId?: string;
  status: string;
  category: string;
  title: string;
  startTime?: string;
  createdAt: string;
  updatedAt: string;
}

interface TestDetail {
  id: string;
  runId: string;
  title: string;
  name?: string;
  status: string;
  metadata?: Record<string, unknown>;
  duration?: number;
  retryCount?: number;
  retryIndex?: number;
  timeout?: number;
  createdAt: string;
  updatedAt: string;
  steps?: StepDetail[];
}

interface TestDetailResponse {
  runId: string;
  tests: TestDetail[];
}

export function TestDetailPage({ onWebSocketEvent }: TestDetailPageProps) {
  const { runId, testId } = useParams<{ runId: string; testId: string }>();
  const [testDetail, setTestDetail] = useState<TestDetailResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (testId) {
      fetchTestDetail(testId);
    }
  }, [testId]);

  const fetchTestDetail = async (id: string) => {
    try {
      setLoading(true);
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
        err instanceof Error ? err.message : "Failed to fetch test details"
      );
    } finally {
      setLoading(false);
    }
  };

  // Handle WebSocket events to update test and step statuses locally
  useEffect(() => {
    if (!onWebSocketEvent || !testDetail) return;

    const { type, data } = onWebSocketEvent;

    if (type === "test.end" || type === "test.begin") {
      const eventData = data as WebSocketTestData;
      const eventTestId = eventData.testCase?.id || eventData.id;
      if (eventTestId === testId) {
        setTestDetail((prevDetail) => {
          if (!prevDetail || !prevDetail.tests || prevDetail.tests.length === 0)
            return prevDetail;

          try {
            // Safely extract status - handle both string and numeric values (protobuf enums)
            const rawStatus = eventData.testCase?.status || eventData.status;
            let status = "RUNNING";
            if (type === "test.end") {
              if (typeof rawStatus === "number") {
                // Protobuf enum mapping: 0=UNKNOWN, 1=PASSED, 2=FAILED, 3=SKIPPED, etc.
                const statusMap: Record<number, string> = {
                  0: "UNKNOWN",
                  1: "PASSED",
                  2: "FAILED",
                  3: "SKIPPED",
                  4: "BROKEN",
                  5: "TIMEDOUT",
                  6: "INTERRUPTED",
                };
                status = statusMap[rawStatus] || "UNKNOWN";
              } else if (typeof rawStatus === "string") {
                status = rawStatus.toUpperCase();
              }
            }

            return {
              ...prevDetail,
              tests: prevDetail.tests.map((t) =>
                t.id === testId
                  ? {
                      ...t,
                      status: status,
                      updatedAt: new Date().toISOString(),
                    }
                  : t
              ),
            };
          } catch (error) {
            console.error("Error updating test detail from WebSocket:", error);
            return prevDetail;
          }
        });
      }
    } else if (type === "step.end" || type === "step.begin") {
      const eventData = data as WebSocketStepData;
      const eventTestCaseRunId = eventData.testCaseRunId;
      if (eventTestCaseRunId === testId && testDetail.tests.length > 0) {
        setTestDetail((prevDetail) => {
          if (!prevDetail || !prevDetail.tests || prevDetail.tests.length === 0)
            return prevDetail;

          try {
            const stepId = eventData.id;
            if (!stepId) return prevDetail;

            // Safely extract status - handle both string and numeric values (protobuf enums)
            const rawStatus = eventData.status;
            let status = "RUNNING";
            if (type === "step.end") {
              if (typeof rawStatus === "number") {
                // Protobuf enum mapping: 0=UNKNOWN, 1=PASSED, 2=FAILED, 3=SKIPPED, etc.
                const statusMap: Record<number, string> = {
                  0: "UNKNOWN",
                  1: "PASSED",
                  2: "FAILED",
                  3: "SKIPPED",
                  4: "BROKEN",
                  5: "TIMEDOUT",
                  6: "INTERRUPTED",
                };
                status = statusMap[rawStatus] || "UNKNOWN";
              } else if (typeof rawStatus === "string") {
                status = rawStatus.toUpperCase();
              }
            }

            const currentTest = prevDetail.tests[0];
            const updatedSteps = [...(currentTest.steps || [])];
            const stepIndex = updatedSteps.findIndex((s) => s.id === stepId);

            if (type === "step.begin") {
              if (stepIndex === -1) {
                // Add new step
                updatedSteps.push({
                  id: stepId,
                  runId: currentTest.runId,
                  testCaseRunId: testId || "",
                  parentStepId: eventData.parentStepId,
                  status: "RUNNING",
                  category: eventData.category || "",
                  title: eventData.title || "",
                  createdAt: new Date().toISOString(),
                  updatedAt: new Date().toISOString(),
                });
              } else {
                updatedSteps[stepIndex] = {
                  ...updatedSteps[stepIndex],
                  status: "RUNNING",
                  updatedAt: new Date().toISOString(),
                };
              }
            } else if (type === "step.end") {
              if (stepIndex >= 0) {
                updatedSteps[stepIndex] = {
                  ...updatedSteps[stepIndex],
                  status: status,
                  updatedAt: new Date().toISOString(),
                };
              }
            }

            return {
              ...prevDetail,
              tests: [
                {
                  ...currentTest,
                  steps: updatedSteps,
                  updatedAt: new Date().toISOString(),
                },
              ],
            };
          } catch (error) {
            console.error("Error updating steps from WebSocket:", error);
            return prevDetail;
          }
        });
      }
    }
  }, [onWebSocketEvent, testId, testDetail]);

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
      PASSED: "passed",
      FAILED: "failed",
      SKIPPED: "skipped",
      RUNNING: "running",
      UNKNOWN: "unknown",
      BROKEN: "broken",
      TIMEDOUT: "timedout",
      INTERRUPTED: "interrupted",
    };
    return (statusMap[status] || "unknown") as TestStatus;
  };

  const formatDuration = (nanoseconds?: number) => {
    if (!nanoseconds) return "N/A";
    const milliseconds = nanoseconds / 1000000;
    if (milliseconds < 1000) return `${milliseconds.toFixed(0)}ms`;
    return `${(milliseconds / 1000).toFixed(2)}s`;
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
  const safeSteps = test.steps || [];
  console.log(
    "Rendering TestDetailPage for test:",
    test.id,
    "with steps:",
    safeSteps
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
                    {new Date(test.createdAt).toLocaleString()}
                  </dd>
                </div>
                <div className="flex justify-between items-start">
                  <dt className="text-gray-600 font-medium">Last Updated:</dt>
                  <dd className="text-gray-900 text-right ml-4">
                    {new Date(test.updatedAt).toLocaleString()}
                  </dd>
                </div>
                <div className="flex justify-between items-start">
                  <dt className="text-gray-600 font-medium">Total Steps:</dt>
                  <dd className="text-gray-900 font-semibold text-right ml-4">
                    {safeSteps.length}
                  </dd>
                </div>
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
        </CardContent>
      </Card>

      <StepContainer
        test={{
          id: test.id,
          runId: test.runId,
          title: test.title || test.name || test.id,
          status: testStatus,
          steps: safeSteps.map((step) => ({
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
          })),
        }}
      />
    </div>
  );
}
