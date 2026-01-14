import { useEffect, useState } from "react";
import { useParams, Link } from "react-router-dom";
import { apiUrl } from "@/lib/config";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/Card";
import { Badge } from "@/components/Badge";
import { ArrowLeft, BarChart3, AlertCircle, TrendingUp } from "lucide-react";
import type { TestStatus } from "@/types/common";
import {
  LineChart,
  Line,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from "recharts";

interface RunStat {
  runId: string;
  runName?: string;
  status?: string;
  metadata?: Record<string, any>;
  startTime?: string;
  endTime?: string;
  duration?: number;
  createdAt: string;
  updatedAt: string;
  total: number;
  passed: number;
  failed: number;
  skipped: number;
  running?: number;
  broken?: number;
  timedout?: number;
  interrupted?: number;
  unknown?: number;
}

interface MarkerStatsResponse {
  marker: string;
  runs: RunStat[];
  total: number;
  count: number;
}

export function MarkerStatsPage() {
  const { markerValue } = useParams<{ markerValue: string }>();
  const [statsData, setStatsData] = useState<MarkerStatsResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (markerValue) {
      fetchMarkerStats(markerValue);
    }
  }, [markerValue]);

  const fetchMarkerStats = async (marker: string) => {
    try {
      setLoading(true);
      const response = await fetch(apiUrl(`/marker/${encodeURIComponent(marker)}/stats?limit=100`));
      if (!response.ok) {
        throw new Error(`Failed to fetch marker stats: ${response.statusText}`);
      }
      const data = await response.json();
      setStatsData(data);
      setError(null);
    } catch (err) {
      console.error("Error fetching marker stats:", err);
      setError(
        err instanceof Error ? err.message : "Failed to fetch marker statistics"
      );
    } finally {
      setLoading(false);
    }
  };

  const getRunStatus = (status?: string): TestStatus => {
    if (!status) return "UNKNOWN";
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
    return (statusMap[status.toUpperCase()] || "UNKNOWN") as TestStatus;
  };

  const formatDuration = (nanoseconds?: number) => {
    if (!nanoseconds) return "N/A";
    const milliseconds = nanoseconds / 1000000;
    if (milliseconds < 1000) return `${milliseconds.toFixed(0)}ms`;
    const seconds = milliseconds / 1000;
    if (seconds < 60) return `${seconds.toFixed(1)}s`;
    const minutes = seconds / 60;
    return `${minutes.toFixed(1)}m`;
  };

  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    return date.toLocaleDateString() + " " + date.toLocaleTimeString();
  };

  const calculateAggregateStats = (runs: RunStat[]) => {
    const stats = {
      totalRuns: runs.length,
      totalTests: 0,
      passed: 0,
      failed: 0,
      skipped: 0,
      broken: 0,
      timedout: 0,
      interrupted: 0,
      unknown: 0,
    };

    runs.forEach((run) => {
      stats.totalTests += run.total;
      stats.passed += run.passed;
      stats.failed += run.failed;
      stats.skipped += run.skipped;
      stats.broken += run.broken || 0;
      stats.timedout += run.timedout || 0;
      stats.interrupted += run.interrupted || 0;
      stats.unknown += run.unknown || 0;
    });

    return stats;
  };

  const calculatePassRate = (runs: RunStat[]) => {
    const totalTests = runs.reduce((sum, run) => sum + run.total, 0);
    if (totalTests === 0) return 0;
    const totalPassed = runs.reduce((sum, run) => sum + run.passed, 0);
    return ((totalPassed / totalTests) * 100).toFixed(1);
  };

  const prepareTimelineData = (runs: RunStat[]) => {
    // Sort by createdAt (oldest to newest for chronological order)
    return [...runs]
      .sort((a, b) => new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime())
      .map((run, index) => {
        const date = new Date(run.createdAt);
        const passRate = run.total > 0 ? ((run.passed / run.total) * 100).toFixed(1) : 0;
        return {
          name: `Run ${index + 1}`,
          runId: run.runId.substring(0, 8),
          passed: run.passed,
          failed: run.failed,
          skipped: run.skipped,
          total: run.total,
          passRate: parseFloat(passRate as string),
          date: date.toLocaleDateString(),
          fullDate: date.toLocaleString(),
        };
      });
  };

  const prepareStatusDistribution = (runs: RunStat[]) => {
    const aggregateStats = calculateAggregateStats(runs);
    return [
      { name: "Passed", value: aggregateStats.passed, fill: "#10b981" },
      { name: "Failed", value: aggregateStats.failed, fill: "#ef4444" },
      { name: "Skipped", value: aggregateStats.skipped, fill: "#f59e0b" },
      { name: "Broken", value: aggregateStats.broken, fill: "#f97316" },
      { name: "Timedout", value: aggregateStats.timedout, fill: "#ec4899" },
      { name: "Interrupted", value: aggregateStats.interrupted, fill: "#8b5cf6" },
      { name: "Unknown", value: aggregateStats.unknown, fill: "#6b7280" },
    ].filter((item) => item.value > 0);
  };

  const CustomTooltip = ({ active, payload }: any) => {
    if (active && payload && payload.length) {
      const data = payload[0].payload;
      return (
        <div className="bg-white p-3 border border-gray-200 rounded-lg shadow-lg">
          <p className="font-semibold text-gray-900">{data.name}</p>
          <p className="text-sm text-gray-600">Run ID: {data.runId}</p>
          <p className="text-sm text-gray-600">Passed: {data.passed}</p>
          <p className="text-sm text-gray-600">Failed: {data.failed}</p>
          <p className="text-sm text-gray-600">Total: {data.total}</p>
          <p className="text-sm text-gray-600">Pass Rate: {data.passRate}%</p>
          <p className="text-xs text-gray-500 mt-1">{data.fullDate}</p>
        </div>
      );
    }
    return null;
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-gray-600">Loading marker statistics...</div>
      </div>
    );
  }

  if (error || !statsData) {
    return (
      <div className="space-y-4">
        <Link
          to="/"
          className="inline-flex items-center text-blue-600 hover:text-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 rounded-md px-2 py-1"
        >
          <ArrowLeft className="h-4 w-4 mr-2" />
          Back to Dashboard
        </Link>
        <Card>
          <CardContent className="py-12">
            <div className="flex flex-col items-center justify-center space-y-4">
              <AlertCircle className="h-16 w-16 text-red-500" />
              <div className="text-red-600 text-center">
                <p className="font-semibold">
                  Error: {error || "Marker statistics not found"}
                </p>
                <p className="text-sm mt-1">
                  Unable to fetch historical data for this marker.
                </p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  const runs = statsData.runs || [];
  const aggregateStats = calculateAggregateStats(runs);
  const passRate = calculatePassRate(runs);

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-4">
          <Link
            to="/"
            className="inline-flex items-center text-blue-600 hover:text-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 rounded-md p-1"
            aria-label="Back to dashboard"
          >
            <ArrowLeft className="h-5 w-5" />
          </Link>
          <div>
            <div className="flex items-center space-x-2">
              <BarChart3 className="h-6 w-6 text-blue-600" />
              <h1 className="text-3xl font-bold text-gray-900">
                Marker Statistics
              </h1>
            </div>
            <p className="text-sm text-gray-600 mt-1">
              Historical performance data for runs with MARKER = "{markerValue}"
            </p>
          </div>
        </div>
      </div>

      {/* Marker Value Card */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">Marker Value</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="font-mono text-sm text-gray-700 break-all">
            {markerValue}
          </p>
        </CardContent>
      </Card>

      {/* Aggregate Statistics Summary */}
      <div className="grid grid-cols-1 md:grid-cols-5 gap-4">
        <Card>
          <CardContent className="py-6">
            <div className="text-center">
              <p className="text-sm text-gray-600 mb-1">Total Runs</p>
              <p className="text-3xl font-bold text-gray-900">
                {aggregateStats.totalRuns}
              </p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="py-6">
            <div className="text-center">
              <p className="text-sm text-gray-600 mb-1">Total Tests</p>
              <p className="text-3xl font-bold text-gray-900">
                {aggregateStats.totalTests}
              </p>
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
                {aggregateStats.passed}
              </p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="py-6">
            <div className="text-center">
              <p className="text-sm text-gray-600 mb-1">Failed</p>
              <p className="text-3xl font-bold text-red-600">
                {aggregateStats.failed}
              </p>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Pass Rate Timeline */}
      {runs.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center space-x-2">
              <TrendingUp className="h-5 w-5 text-blue-600" />
              <span>Pass Rate Over Time</span>
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="w-full h-80">
              <ResponsiveContainer width="100%" height="100%">
                <LineChart
                  data={prepareTimelineData(runs)}
                  margin={{ top: 5, right: 30, left: 20, bottom: 5 }}
                >
                  <CartesianGrid strokeDasharray="3 3" className="stroke-gray-200" />
                  <XAxis
                    dataKey="name"
                    tick={{ fontSize: 12 }}
                    className="text-gray-600"
                  />
                  <YAxis
                    label={{
                      value: "Pass Rate (%)",
                      angle: -90,
                      position: "insideLeft",
                      style: { fontSize: 12 },
                    }}
                    domain={[0, 100]}
                    tick={{ fontSize: 12 }}
                    className="text-gray-600"
                  />
                  <Tooltip content={<CustomTooltip />} />
                  <Legend />
                  <Line
                    type="monotone"
                    dataKey="passRate"
                    stroke="#10b981"
                    strokeWidth={2}
                    dot={{ fill: "#10b981", r: 4 }}
                    activeDot={{ r: 6 }}
                    name="Pass Rate (%)"
                  />
                </LineChart>
              </ResponsiveContainer>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Test Status Distribution */}
      {runs.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>Test Status Distribution</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="w-full h-80">
              <ResponsiveContainer width="100%" height="100%">
                <BarChart
                  data={prepareStatusDistribution(runs)}
                  margin={{ top: 5, right: 30, left: 20, bottom: 5 }}
                >
                  <CartesianGrid strokeDasharray="3 3" className="stroke-gray-200" />
                  <XAxis
                    dataKey="name"
                    tick={{ fontSize: 12 }}
                    className="text-gray-600"
                  />
                  <YAxis
                    label={{
                      value: "Count",
                      angle: -90,
                      position: "insideLeft",
                      style: { fontSize: 12 },
                    }}
                    tick={{ fontSize: 12 }}
                    className="text-gray-600"
                  />
                  <Tooltip />
                  <Legend />
                  <Bar dataKey="value" name="Tests" />
                </BarChart>
              </ResponsiveContainer>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Runs Table */}
      <Card>
        <CardHeader>
          <CardTitle>Run History</CardTitle>
        </CardHeader>
        <CardContent>
          {runs.length === 0 ? (
            <div className="text-center py-8 text-gray-500">
              No runs found with MARKER = "{markerValue}"
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
                      Name
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Status
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Tests
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Pass Rate
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Duration
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Created At
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Actions
                    </th>
                  </tr>
                </thead>
                <tbody className="bg-white divide-y divide-gray-200">
                  {runs.map((run) => {
                    const runPassRate = run.total > 0 
                      ? ((run.passed / run.total) * 100).toFixed(1)
                      : "0.0";
                    return (
                      <tr key={run.runId} className="hover:bg-gray-50">
                        <td className="px-6 py-4 whitespace-nowrap">
                          <Link
                            to={`/suite_runs/${run.runId}`}
                            className="font-mono text-sm text-blue-600 hover:underline focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 rounded"
                          >
                            {run.runId.substring(0, 12)}...
                          </Link>
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                          {run.runName || "N/A"}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap">
                          <Badge status={getRunStatus(run.status)} />
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap">
                          <div className="text-sm text-gray-900">
                            <span className="text-green-600 font-semibold">
                              {run.passed}
                            </span>
                            {" / "}
                            <span className="text-red-600 font-semibold">
                              {run.failed}
                            </span>
                            {" / "}
                            {run.total}
                          </div>
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                          {runPassRate}%
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                          {formatDuration(run.duration)}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                          {formatDate(run.createdAt)}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm">
                          <Link
                            to={`/suite_runs/${run.runId}`}
                            className="text-blue-600 hover:text-blue-700 hover:underline focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 rounded px-2 py-1"
                          >
                            View Details
                          </Link>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
