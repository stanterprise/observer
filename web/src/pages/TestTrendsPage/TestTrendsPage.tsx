import { useEffect, useState } from "react";
import { useParams, Link } from "react-router-dom";
import { apiUrl } from "@/lib/config";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/Card";
import { Badge } from "@/components/Badge";
import { ArrowLeft, TrendingUp, Clock, AlertCircle } from "lucide-react";
import type { TestStatus } from "@/types/common";
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from "recharts";

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

  const prepareChartData = (trends: TestTrendItem[]) => {
    // Reverse to show chronological order (oldest to newest)
    return [...trends]
      .reverse()
      .map((trend, index) => {
        const durationMs = trend.duration ? trend.duration / 1000000 : 0;
        const date = new Date(trend.createdAt);
        return {
          name: `Run ${index + 1}`,
          duration: parseFloat(durationMs.toFixed(2)),
          fullDate: date.toLocaleString(),
          runId: trend.runId,
          status: trend.status,
        };
      })
      .filter((item) => item.duration > 0); // Only include items with valid duration
  };

  const CustomTooltip = ({ active, payload }: any) => {
    if (active && payload && payload.length) {
      const data = payload[0].payload;
      return (
        <div className="bg-(--stitch-surface-card) p-3 border border-(--stitch-outline) rounded-lg shadow-lg">
          <p className="font-semibold text-(--stitch-on-surface)">{data.name}</p>
          <p className="text-sm text-(--stitch-on-surface-muted)">Duration: {data.duration}ms</p>
          <p className="text-sm text-(--stitch-on-surface-muted)">Status: {data.status}</p>
          <p className="text-xs text-(--stitch-on-surface-subtle) mt-1">{data.fullDate}</p>
        </div>
      );
    }
    return null;
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-(--stitch-on-surface-muted)">Loading test trends...</div>
      </div>
    );
  }

  if (error || !trendsData) {
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
            className="inline-flex items-center text-(--stitch-primary) hover:text-(--stitch-primary) focus:outline-none focus:ring-2 focus:ring-(--stitch-primary) focus:ring-offset-2 rounded-md p-1"
            aria-label="Back to test runs"
          >
            <ArrowLeft className="h-5 w-5" />
          </Link>
          <div>
            <div className="flex items-center space-x-2">
              <TrendingUp className="h-6 w-6 text-(--stitch-primary)" />
              <h1 className="text-3xl font-bold text-(--stitch-on-surface)">Test Trends</h1>
            </div>
            <p className="text-sm text-(--stitch-on-surface-muted) mt-1">
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
          <p className="font-mono text-sm text-(--stitch-on-surface-muted) break-all">{testId}</p>
        </CardContent>
      </Card>

      {/* Statistics Summary */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card>
          <CardContent className="py-6">
            <div className="text-center">
              <p className="text-sm text-(--stitch-on-surface-muted) mb-1">Total Runs</p>
              <p className="text-3xl font-bold text-(--stitch-on-surface)">{stats.total}</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="py-6">
            <div className="text-center">
              <p className="text-sm text-(--stitch-on-surface-muted) mb-1">Pass Rate</p>
              <p className="text-3xl font-bold text-(--status-success)">{passRate}%</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="py-6">
            <div className="text-center">
              <p className="text-sm text-(--stitch-on-surface-muted) mb-1">Passed</p>
              <p className="text-3xl font-bold text-(--status-success)">
                {stats.passed}
              </p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="py-6">
            <div className="text-center">
              <p className="text-sm text-(--stitch-on-surface-muted) mb-1">Failed</p>
              <p className="text-3xl font-bold text-(--status-failure)">{stats.failed}</p>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Average Duration */}
      {avgDuration !== null && (
        <Card>
          <CardContent className="py-6">
            <div className="flex items-center justify-center space-x-3">
              <Clock className="h-5 w-5 text-(--stitch-primary)" />
              <div className="text-center">
                <p className="text-sm text-(--stitch-on-surface-muted)">Average Duration</p>
                <p className="text-xl font-semibold text-(--stitch-on-surface)">
                  {formatDuration(avgDuration)}
                </p>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Execution Time Graph */}
      {trends.length > 0 && prepareChartData(trends).length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>Execution Time Trend</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="w-full h-80">
              <ResponsiveContainer width="100%" height="100%">
                <LineChart
                  data={prepareChartData(trends)}
                  margin={{ top: 5, right: 30, left: 20, bottom: 5 }}
                >
                  <CartesianGrid strokeDasharray="3 3" className="stroke-gray-200" />
                  <XAxis
                    dataKey="name"
                    tick={{ fontSize: 12 }}
                    className="text-(--stitch-on-surface-muted)"
                  />
                  <YAxis
                    label={{
                      value: "Duration (ms)",
                      angle: -90,
                      position: "insideLeft",
                      style: { fontSize: 12 },
                    }}
                    tick={{ fontSize: 12 }}
                    className="text-(--stitch-on-surface-muted)"
                  />
                  <Tooltip content={<CustomTooltip />} />
                  <Legend />
                  <Line
                    type="monotone"
                    dataKey="duration"
                    stroke="#2563eb"
                    strokeWidth={2}
                    dot={{ fill: "#2563eb", r: 4 }}
                    activeDot={{ r: 6 }}
                    name="Duration (ms)"
                  />
                </LineChart>
              </ResponsiveContainer>
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
            <div className="text-center py-8 text-(--stitch-on-surface-subtle)">
              No test execution history found for this test.
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-(--stitch-outline)">
                <thead className="bg-(--stitch-surface-low)">
                  <tr>
                    <th className="px-6 py-3 text-left text-xs font-medium text-(--stitch-on-surface-subtle) uppercase tracking-wider">
                      Run ID
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-(--stitch-on-surface-subtle) uppercase tracking-wider">
                      Status
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-(--stitch-on-surface-subtle) uppercase tracking-wider">
                      Duration
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-(--stitch-on-surface-subtle) uppercase tracking-wider">
                      Executed At
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-(--stitch-on-surface-subtle) uppercase tracking-wider">
                      Actions
                    </th>
                  </tr>
                </thead>
                <tbody className="bg-(--stitch-surface-card) divide-y divide-(--stitch-outline)">
                  {trends.map((trend, index) => (
                    <tr key={`${trend.runId}-${trend.testId}-${index}`} className="hover:bg-(--stitch-surface-low)">
                      <td className="px-6 py-4 whitespace-nowrap">
                        <Link
                          to={`/suite_runs/${trend.runId}`}
                          className="font-mono text-sm text-(--stitch-primary) hover:underline focus:outline-none focus:ring-2 focus:ring-(--stitch-primary) focus:ring-offset-2 rounded"
                        >
                          {trend.runId.substring(0, 12)}...
                        </Link>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <Badge status={getTestStatus(trend.status)} />
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-(--stitch-on-surface)">
                        {formatDuration(trend.duration)}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-(--stitch-on-surface-subtle)">
                        {formatDate(trend.createdAt)}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm">
                        <Link
                          to={`/suite_runs/${trend.runId}/tests/${trend.testId}`}
                          className="text-(--stitch-primary) hover:text-(--stitch-primary) hover:underline focus:outline-none focus:ring-2 focus:ring-(--stitch-primary) focus:ring-offset-2 rounded px-2 py-1"
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
