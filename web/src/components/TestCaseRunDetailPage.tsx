import { useEffect, useState } from "react";
import { useParams, Link } from "react-router-dom";
import { apiUrl } from "../lib/config";
import { Card, CardHeader, CardTitle, CardContent } from "./Card";
import { Badge } from "./Badge";
import type { WebSocketEvent, TestStatus, WebSocketTestData, WebSocketStepData } from "../types";
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
  const { testId } = useParams<{ testId: string }>();
  const [testDetail, setTestDetail] = useState<TestDetailResponse | null>(
    null
  );
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
      const response = await fetch(apiUrl(`/tests/${id}`));
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

  // Handle WebSocket events to update test and step statuses
  useEffect(() => {
    if (!onWebSocketEvent || !testDetail) return;

    const { type, data } = onWebSocketEvent;

    if (type === "test.end" || type === "test.begin") {
      const eventData = data as WebSocketTestData;
      const eventTestId = eventData.test_case?.id || eventData.id;
      if (eventTestId === testId) {
        // Refetch test details
        if (testId) {
          fetchTestDetail(testId);
        }
      }
    } else if (type === "step.end" || type === "step.begin") {
      const eventData = data as WebSocketStepData;
      const eventTestCaseRunId = eventData.test_case_run_id;
      if (eventTestCaseRunId === testId) {
        // Refetch test details to get updated steps
        if (testId) {
          fetchTestDetail(testId);
        }
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
          to="/"
          className="inline-flex items-center text-blue-600 hover:text-blue-700"
        >
          <ArrowLeft className="h-4 w-4 mr-2" />
          Back to Tests
        </Link>
        <div className="flex items-center justify-center h-64">
          <div className="text-red-600">
            Error: {error || "Test not found"}
          </div>
        </div>
      </div>
    );
  }

  const getTestStatus = (status: string): TestStatus => {
    const statusMap: Record<string, TestStatus> = {
      PASSED: "passed",
      FAILED: "failed",
      SKIPPED: "skipped",
      RUNNING: "running",
    };
    return (statusMap[status] || "pending") as TestStatus;
  };

  const formatDuration = (nanoseconds?: number) => {
    if (!nanoseconds) return "N/A";
    const milliseconds = nanoseconds / 1000000;
    if (milliseconds < 1000) return `${milliseconds.toFixed(0)}ms`;
    return `${(milliseconds / 1000).toFixed(2)}s`;
  };

  const { test, steps } = testDetail;
  const testStatus = getTestStatus(test.Status);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-4">
          <Link
            to={`/runs/${test.RunID}`}
            className="inline-flex items-center text-blue-600 hover:text-blue-700"
          >
            <ArrowLeft className="h-5 w-5" />
          </Link>
          <h1 className="text-3xl font-bold text-gray-900">Test Case Detail</h1>
        </div>
      </div>

      {/* Test Case Summary Card */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle className="text-xl">{test.Title || test.ID}</CardTitle>
            <Badge status={testStatus} className="text-lg px-4 py-2" />
          </div>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div>
              <h3 className="text-sm font-medium text-gray-700 mb-3">
                Test Information
              </h3>
              <dl className="space-y-2 text-sm">
                <div className="flex justify-between">
                  <dt className="text-gray-600">Test ID:</dt>
                  <dd className="font-mono text-gray-900">{test.ID}</dd>
                </div>
                <div className="flex justify-between">
                  <dt className="text-gray-600">Run ID:</dt>
                  <dd className="font-mono text-gray-900">
                    <Link
                      to={`/runs/${test.RunID}`}
                      className="text-blue-600 hover:underline"
                    >
                      {test.RunID}
                    </Link>
                  </dd>
                </div>
                <div className="flex justify-between">
                  <dt className="text-gray-600">Duration:</dt>
                  <dd className="text-gray-900">
                    {formatDuration(test.Duration)}
                  </dd>
                </div>
                {test.RetryCount !== undefined && test.RetryCount > 0 && (
                  <div className="flex justify-between">
                    <dt className="text-gray-600">Retries:</dt>
                    <dd className="text-gray-900">
                      {test.RetryIndex !== undefined
                        ? `${test.RetryIndex} / ${test.RetryCount}`
                        : test.RetryCount}
                    </dd>
                  </div>
                )}
                {test.Timeout !== undefined && (
                  <div className="flex justify-between">
                    <dt className="text-gray-600">Timeout:</dt>
                    <dd className="text-gray-900">{test.Timeout}ms</dd>
                  </div>
                )}
              </dl>
            </div>
            <div>
              <h3 className="text-sm font-medium text-gray-700 mb-3">
                Execution Timeline
              </h3>
              <dl className="space-y-2 text-sm">
                <div className="flex justify-between">
                  <dt className="text-gray-600">Started:</dt>
                  <dd className="text-gray-900">
                    {new Date(test.CreatedAt).toLocaleString()}
                  </dd>
                </div>
                <div className="flex justify-between">
                  <dt className="text-gray-600">Last Updated:</dt>
                  <dd className="text-gray-900">
                    {new Date(test.UpdatedAt).toLocaleString()}
                  </dd>
                </div>
                <div className="flex justify-between">
                  <dt className="text-gray-600">Total Steps:</dt>
                  <dd className="text-gray-900">{steps.length}</dd>
                </div>
              </dl>
            </div>
          </div>

          {/* Metadata Section */}
          {test.Metadata && Object.keys(test.Metadata).length > 0 && (
            <div className="mt-6 pt-6 border-t border-gray-200">
              <h3 className="text-sm font-medium text-gray-700 mb-3">
                Metadata
              </h3>
              <div className="bg-gray-50 rounded-lg p-4">
                <pre className="text-xs text-gray-800 overflow-x-auto">
                  {JSON.stringify(test.Metadata, null, 2)}
                </pre>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Steps Section */}
      <div>
        <h2 className="text-xl font-semibold text-gray-900 mb-4">
          Execution Steps ({steps.length})
        </h2>
        {steps.length === 0 ? (
          <Card>
            <CardContent>
              <div className="text-center py-8 text-gray-500">
                No steps recorded for this test case.
              </div>
            </CardContent>
          </Card>
        ) : (
          <div className="space-y-3">
            {renderStepHierarchy(steps, null, 0, getTestStatus)}
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

    return (
      <div key={step.ID}>
        <div style={{ marginLeft: level > 0 ? `${level * 2}rem` : "0" }}>
          <Card
            className={`${
              stepStatus === "failed" ? "border-red-200 bg-red-50" : ""
            }`}
          >
            <CardContent className="py-4">
              <div className="flex items-start space-x-4">
                <div className="flex-shrink-0 w-8 h-8 rounded-full bg-gray-200 flex items-center justify-center text-sm font-medium text-gray-700">
                  {level > 0 ? "↳" : index + 1}
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center justify-between mb-2">
                    <div className="flex items-center space-x-3">
                      <Badge status={stepStatus} />
                      <span className="text-sm font-medium text-gray-700">
                        {step.Title || step.Category || "Step"}
                      </span>
                    </div>
                    {stepStatus === "passed" && (
                      <CheckCircle2 className="h-5 w-5 text-green-600" />
                    )}
                    {stepStatus === "failed" && (
                      <AlertCircle className="h-5 w-5 text-red-600" />
                    )}
                  </div>
                  {step.Category && step.Title && (
                    <div className="text-xs text-gray-500 mb-1">
                      Category: {step.Category}
                    </div>
                  )}
                  <div className="flex items-center space-x-4 text-xs text-gray-500">
                    <div className="flex items-center">
                      <Clock className="h-3 w-3 mr-1" />
                      Started: {new Date(step.CreatedAt).toLocaleString()}
                    </div>
                    {step.UpdatedAt !== step.CreatedAt && (
                      <div>
                        Completed: {new Date(step.UpdatedAt).toLocaleString()}
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
