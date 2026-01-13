import { useEffect, useState } from "react";
import { useParams, Link } from "react-router-dom";
import { apiUrl } from "@/lib/config";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/Card";
import { Badge } from "@/components/Badge";
import { ArrowLeft, TrendingUp, Clock, AlertCircle } from "lucide-react";
import type { TestStatus } from "@/types/common";

interface TestTrendItem {
  testId: string;
  runId: string;
  status: string;
  duration?: number;
  startTime?: string;
  endTime?: string;
  createdAt: string;
}

interface TestTrendsResponse {
  testId: string;
  trends: TestTrendItem[];
  count: number;
}

export function TestTrendsPage() {
  const { testId } = useParams<{ testId: string }>();
  const [trendsData, setTrendsData] = useState<TestTrendsResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (testId) {
      fetchTestTrends(testId);
    }
  }, [testId]);

  const fetchTestTrends = async (id: string) => {
    try {
      setLoading(true);
      const response = await fetch(apiUrl(`/tests/${id}/trends?limit=50`));
      if (!response.ok) {
        throw new Error(`Failed to fetch test trends: ${response.statusText}`);
      }
      const data = await response.json();
      setTrendsData(data);
      setError(null);
    } catch (err) {
      console.error("Error fetching test trends:", err);
      setError(
        err instanceof Error ? err.message : "Failed to fetch test trends"
      );
    } finally {
      setLoading(false);
    }
  };

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

  const formatDuration = (nanoseconds?: number) => {
    if (!nanoseconds) return "N/A";
    const milliseconds = nanoseconds / 1000000;
    if (milliseconds < 1000) return `${milliseconds.toFixed(0)}ms`;
    return `${(milliseconds / 1000).toFixed(2)}s`;
  };

  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    return date.toLocaleString();
  };

  const calculateStats = (trends: TestTrendItem[]) => {
    const stats = {
      total: trends.length,
      passed: 0,
      failed: 0,
      skipped: 0,
      broken: 0,
      timedout: 0,
      interrupted: 0,
      unknown: 0,
    };

    trends.forEach((trend) => {
      const status = trend.status.toUpperCase();
      switch (status) {
        case "PASSED":
          stats.passed++;
          break;
        case "FAILED":
          stats.failed++;
          break;
        case "SKIPPED":
          stats.skipped++;
          break;
        case "BROKEN":
          stats.broken++;
          break;
        case "TIMEDOUT":
          stats.timedout++;
          break;
        case "INTERRUPTED":
          stats.interrupted++;
          break;
        default:
          stats.unknown++;
      }
    });

    return stats;
  };

  const calculatePassRate = (trends: TestTrendItem[]) => {
    if (trends.length === 0) return 0;
    const passed = trends.filter(
      (t) => t.status.toUpperCase() === "PASSED"
    ).length;
    return ((passed / trends.length) * 100).toFixed(1);
  };

  const getAverageDuration = (trends: TestTrendItem[]) => {
    const validDurations = trends
      .filter((t) => t.duration !== undefined && t.duration !== null)
      .map((t) => t.duration!);

    if (validDurations.length === 0) return null;

    const sum = validDurations.reduce((acc, d) => acc + d, 0);
    return sum / validDurations.length;
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-gray-600">Loading test trends...</div>
      </div>
    );
  }

  if (error || !trendsData) {
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
                  Error: {error || "Test trends not found"}
                </p>
                <p className="text-sm mt-1">
                  Unable to fetch historical data for this test.
                </p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  const trends = trendsData.trends || [];
  const stats = calculateStats(trends);
  const passRate = calculatePassRate(trends);
  const avgDuration = getAverageDuration(trends);

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-4">
          <Link
            to="/suite_runs"
            className="inline-flex items-center text-blue-600 hover:text-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 rounded-md p-1"
            aria-label="Back to test runs"
          >
            <ArrowLeft className="h-5 w-5" />
          </Link>
          <div>
            <div className="flex items-center space-x-2">
              <TrendingUp className="h-6 w-6 text-blue-600" />
              <h1 className="text-3xl font-bold text-gray-900">Test Trends</h1>
            </div>
            <p className="text-sm text-gray-600 mt-1">
              Historical performance data across multiple runs
            </p>
          </div>
        </div>
      </div>

      {/* Test ID Card */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">Test ID</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="font-mono text-sm text-gray-700 break-all">{testId}</p>
        </CardContent>
      </Card>

      {/* Statistics Summary */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card>
          <CardContent className="py-6">
            <div className="text-center">
              <p className="text-sm text-gray-600 mb-1">Total Runs</p>
              <p className="text-3xl font-bold text-gray-900">{stats.total}</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="py-6">
            <div className="text-center">
              <p className="text-sm text-gray-600 mb-1">Pass Rate</p>
              <p className="text-3xl font-bold text-green-600">{passRate}%</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="py-6">
            <div className="text-center">
              <p className="text-sm text-gray-600 mb-1">Passed</p>
              <p className="text-3xl font-bold text-green-600">
                {stats.passed}
              </p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="py-6">
            <div className="text-center">
              <p className="text-sm text-gray-600 mb-1">Failed</p>
              <p className="text-3xl font-bold text-red-600">{stats.failed}</p>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Average Duration */}
      {avgDuration !== null && (
        <Card>
          <CardContent className="py-6">
            <div className="flex items-center justify-center space-x-3">
              <Clock className="h-5 w-5 text-blue-600" />
              <div className="text-center">
                <p className="text-sm text-gray-600">Average Duration</p>
                <p className="text-xl font-semibold text-gray-900">
                  {formatDuration(avgDuration)}
                </p>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Trends Table */}
      <Card>
        <CardHeader>
          <CardTitle>Test Execution History</CardTitle>
        </CardHeader>
        <CardContent>
          {trends.length === 0 ? (
            <div className="text-center py-8 text-gray-500">
              No test execution history found for this test.
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-gray-200">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Run ID
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Status
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Duration
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Executed At
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Actions
                    </th>
                  </tr>
                </thead>
                <tbody className="bg-white divide-y divide-gray-200">
                  {trends.map((trend, index) => (
                    <tr key={`${trend.runId}-${index}`} className="hover:bg-gray-50">
                      <td className="px-6 py-4 whitespace-nowrap">
                        <Link
                          to={`/suite_runs/${trend.runId}`}
                          className="font-mono text-sm text-blue-600 hover:underline focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 rounded"
                        >
                          {trend.runId.substring(0, 12)}...
                        </Link>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <Badge status={getTestStatus(trend.status)} />
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                        {formatDuration(trend.duration)}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                        {formatDate(trend.createdAt)}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm">
                        <Link
                          to={`/suite_runs/${trend.runId}/tests/${trend.testId}`}
                          className="text-blue-600 hover:text-blue-700 hover:underline focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 rounded px-2 py-1"
                        >
                          View Details
                        </Link>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
