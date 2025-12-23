import { useEffect, useState } from "react";
import { useParams, Link } from "react-router-dom";
import { apiUrl } from "../lib/config";
import { Card, CardHeader, CardTitle, CardContent } from "./Card";
import { Badge } from "./Badge";
import type {
  WebSocketEvent,
  TestStatus,
  WebSocketTestData,
  WebSocketStepData,
} from "../types";
import { ArrowLeft, Clock, AlertCircle, CheckCircle2 } from "lucide-react";

interface TestCaseRunDetailPageProps {
  onWebSocketEvent?: WebSocketEvent | null;
}

interface Step {
  ID: string;
  RunID: string;
  TestCaseRunID: string;
  ParentStepID?: string;
  Status: string;
  Category: string;
  Title: string;
  CreatedAt: string;
  UpdatedAt: string;
  Steps?: Step[]; // Nested steps for hierarchical structure
}

interface TestCaseDetail {
  ID: string;
  RunID: string;
  Title: string;
  Status: string;
  Metadata?: Record<string, unknown>;
  Duration?: number;
  RetryCount?: number;
  RetryIndex?: number;
  Timeout?: number;
  CreatedAt: string;
  UpdatedAt: string;
}

interface TestDetailResponse {
  test: TestCaseDetail;
  steps: Step[];
}

export function TestCaseRunDetailPage({
  onWebSocketEvent,
}: TestCaseRunDetailPageProps) {
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
      const eventTestId = eventData.test_case?.id || eventData.id;
      if (eventTestId === testId) {
        setTestDetail((prevDetail) => {
          if (!prevDetail || !prevDetail.test) return prevDetail;

          try {
            // Safely extract status - handle both string and non-string values
            const rawStatus = eventData.test_case?.status || eventData.status;
            const status =
              typeof rawStatus === "string"
                ? rawStatus.toUpperCase()
                : "RUNNING";

            return {
              ...prevDetail,
              test: {
                ...prevDetail.test,
                Status: status,
                UpdatedAt: new Date().toISOString(),
              },
            };
          } catch (error) {
            console.error("Error updating test detail from WebSocket:", error);
            return prevDetail;
          }
        });
      }
    } else if (type === "step.end" || type === "step.begin") {
      const eventData = data as WebSocketStepData;
      const eventTestCaseRunId = eventData.test_case_run_id;
      if (eventTestCaseRunId === testId) {
        setTestDetail((prevDetail) => {
          if (!prevDetail || !prevDetail.steps) return prevDetail;

          try {
            const stepId = eventData.id;
            // Safely extract status - handle both string and non-string values
            const rawStatus = eventData.status;
            const status =
              typeof rawStatus === "string"
                ? rawStatus.toUpperCase()
                : "RUNNING";
            const updatedSteps = [...prevDetail.steps];
            const stepIndex = updatedSteps.findIndex((s) => s.ID === stepId);

            if (type === "step.begin") {
              if (stepIndex === -1) {
                // Add new step
                updatedSteps.push({
                  ID: stepId || "",
                  RunID: prevDetail.test.RunID,
                  TestCaseRunID: testId || "",
                  ParentStepID: eventData.parent_step_id,
                  Status: "RUNNING",
                  Category: eventData.category || "",
                  Title: eventData.title || "",
                  CreatedAt: new Date().toISOString(),
                  UpdatedAt: new Date().toISOString(),
                });
              } else {
                updatedSteps[stepIndex] = {
                  ...updatedSteps[stepIndex],
                  Status: "RUNNING",
                  UpdatedAt: new Date().toISOString(),
                };
              }
            } else if (type === "step.end") {
              if (stepIndex >= 0) {
                updatedSteps[stepIndex] = {
                  ...updatedSteps[stepIndex],
                  Status: status,
                  UpdatedAt: new Date().toISOString(),
                };
              }
            }

            return {
              ...prevDetail,
              steps: updatedSteps,
            };
          } catch (error) {
            console.error("Error updating steps from WebSocket:", error);
            return prevDetail;
          }
        });
      }
    }
  }, [onWebSocketEvent, testDetail, testId]);

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

  const { test, steps } = testDetail;
  const testStatus = getTestStatus(test.Status);
  const safeSteps = steps || [];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-4">
          <Link
            to={`/suite_runs/${test.RunID}`}
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
              <CardTitle className="text-xl mb-2 break-words">
                {test.Title || test.ID}
              </CardTitle>
              <p className="text-sm text-gray-500 font-mono">{test.ID}</p>
            </div>
            <Badge
              status={testStatus}
              className="text-lg px-4 py-2 flex-shrink-0 ml-4"
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
                    {test.ID}
                  </dd>
                </div>
                <div className="flex justify-between items-start">
                  <dt className="text-gray-600 font-medium">Run ID:</dt>
                  <dd className="text-right ml-4">
                    <Link
                      to={`/suite_runs/${test.RunID}`}
                      className="font-mono text-blue-600 hover:underline focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 rounded"
                    >
                      {test.RunID}
                    </Link>
                  </dd>
                </div>
                <div className="flex justify-between items-start">
                  <dt className="text-gray-600 font-medium">Duration:</dt>
                  <dd className="text-gray-900 font-semibold text-right ml-4">
                    {formatDuration(test.Duration)}
                  </dd>
                </div>
                {test.RetryCount !== undefined && test.RetryCount > 0 && (
                  <div className="flex justify-between items-start">
                    <dt className="text-gray-600 font-medium">Retries:</dt>
                    <dd className="text-gray-900 text-right ml-4">
                      {test.RetryIndex !== undefined
                        ? `${test.RetryIndex} / ${test.RetryCount}`
                        : test.RetryCount}
                    </dd>
                  </div>
                )}
                {test.Timeout !== undefined && (
                  <div className="flex justify-between items-start">
                    <dt className="text-gray-600 font-medium">Timeout:</dt>
                    <dd className="text-gray-900 text-right ml-4">
                      {test.Timeout}ms
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
                    {new Date(test.CreatedAt).toLocaleString()}
                  </dd>
                </div>
                <div className="flex justify-between items-start">
                  <dt className="text-gray-600 font-medium">Last Updated:</dt>
                  <dd className="text-gray-900 text-right ml-4">
                    {new Date(test.UpdatedAt).toLocaleString()}
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
          {test.Metadata && Object.keys(test.Metadata).length > 0 && (
            <div className="mt-6 pt-6 border-t border-gray-200">
              <h3 className="text-sm font-semibold text-gray-700 mb-3 uppercase tracking-wide">
                Metadata
              </h3>
              <div className="bg-gray-50 rounded-lg p-4 border border-gray-200">
                <pre className="text-xs text-gray-800 overflow-x-auto whitespace-pre-wrap break-words">
                  {JSON.stringify(test.Metadata, null, 2)}
                </pre>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Steps Section */}
      <div>
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-xl font-semibold text-gray-900">
            Execution Steps
          </h2>
          <span className="text-sm text-gray-600 bg-gray-100 px-3 py-1 rounded-full">
            {safeSteps.length} {safeSteps.length === 1 ? "step" : "steps"}
          </span>
        </div>
        {safeSteps.length === 0 ? (
          <Card>
            <CardContent>
              <div className="text-center py-8 text-gray-500">
                <Clock className="mx-auto h-8 w-8 mb-2 text-gray-400" />
                <p>No steps recorded for this test case.</p>
                <p className="text-sm mt-1">
                  Steps will appear here as the test executes.
                </p>
              </div>
            </CardContent>
          </Card>
        ) : (
          <div className="space-y-3">
            {renderStepHierarchy(safeSteps, null, 0, getTestStatus)}
          </div>
        )}
      </div>
    </div>
  );
}

// Helper function to render steps hierarchically
function renderStepHierarchy(
  allSteps: Step[],
  parentId: string | null,
  level: number,
  getTestStatus: (status: string) => TestStatus
): React.ReactNode[] {
  // Find all steps that have the given parent
  const childSteps = allSteps.filter(
    (step) => (step.ParentStepID || null) === parentId
  );

  return childSteps.map((step, index) => {
    const stepStatus = getTestStatus(step.Status);
    const hasChildren = allSteps.some((s) => s.ParentStepID === step.ID);
    const isTopLevel = level === 0;

    return (
      <div key={step.ID}>
        <div style={{ marginLeft: level > 0 ? `${level * 2}rem` : "0" }}>
          <Card
            className={`transition-all duration-200 ${
              stepStatus === "failed"
                ? "border-red-300 bg-red-50"
                : stepStatus === "passed"
                ? "border-green-200 bg-green-50/30"
                : ""
            }`}
          >
            <CardContent className="py-4">
              <div className="flex items-start space-x-4">
                <div
                  className={`flex-shrink-0 w-8 h-8 rounded-full flex items-center justify-center text-sm font-medium ${
                    stepStatus === "passed"
                      ? "bg-green-100 text-green-700"
                      : stepStatus === "failed"
                      ? "bg-red-100 text-red-700"
                      : stepStatus === "running"
                      ? "bg-blue-100 text-blue-700"
                      : "bg-gray-100 text-gray-700"
                  }`}
                  aria-hidden="true"
                >
                  {isTopLevel ? index + 1 : "↳"}
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center justify-between mb-2 flex-wrap gap-2">
                    <div className="flex items-center space-x-3">
                      <Badge status={stepStatus} showIcon={true} />
                      <span className="text-sm font-medium text-gray-900">
                        {step.Title || step.Category || "Step"}
                      </span>
                    </div>
                    <div className="flex items-center space-x-2">
                      {stepStatus === "passed" && (
                        <CheckCircle2 className="h-5 w-5 text-green-600" />
                      )}
                      {stepStatus === "failed" && (
                        <AlertCircle className="h-5 w-5 text-red-600" />
                      )}
                    </div>
                  </div>
                  {step.Category && step.Title && (
                    <div className="text-xs text-gray-600 mb-2 flex items-center">
                      <span className="font-medium mr-1">Category:</span>
                      <span className="font-mono bg-gray-100 px-2 py-0.5 rounded">
                        {step.Category}
                      </span>
                    </div>
                  )}
                  <div className="flex items-center flex-wrap gap-x-4 gap-y-1 text-xs text-gray-500">
                    <div className="flex items-center">
                      <Clock className="h-3 w-3 mr-1" />
                      <span className="font-medium">Started:</span>
                      <span className="ml-1">
                        {new Date(step.CreatedAt).toLocaleString()}
                      </span>
                    </div>
                    {step.UpdatedAt !== step.CreatedAt && (
                      <div className="flex items-center">
                        <span className="font-medium">Completed:</span>
                        <span className="ml-1">
                          {new Date(step.UpdatedAt).toLocaleString()}
                        </span>
                      </div>
                    )}
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
        {/* Recursively render child steps */}
        {hasChildren && (
          <div className="space-y-3 mt-3">
            {renderStepHierarchy(allSteps, step.ID, level + 1, getTestStatus)}
          </div>
        )}
      </div>
    );
  });
}
