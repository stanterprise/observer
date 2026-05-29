import { useEffect, useState } from "react";
import { useParams, Link } from "react-router-dom";
import { apiUrl } from "@/lib/config";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/Card";
import { ArrowLeft, BarChart3, AlertCircle, TrendingUp } from "lucide-react";
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
  AreaChart,
  Area,
} from "recharts";
import { humanizeDuration } from "@/utils/duration";

interface RunStat {
  runId: string;
  name: string;
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
  flaky: number;
}

interface MarkerStatsResponse {
  marker: string;
  runs: RunStat[];
}

type AggregateStats = {
  totalRuns: number;
  totalTests: number;
  avgTests: number;
  passed: number;
  failed: number;
  skipped: number;
  broken: number;
  timedout: number;
  interrupted: number;
  unknown: number;
  avgPassed: number;
  avgFailed: number;
};

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
      const response = await fetch(
        apiUrl(`/marker/${encodeURIComponent(marker)}/stats`),
      );
      if (!response.ok) {
        throw new Error(`Failed to fetch marker stats: ${response.statusText}`);
      }
      const data = await response.json();
      setStatsData(data);
      setError(null);
    } catch (err) {
      console.error("Error fetching marker stats:", err);
      setError(
        err instanceof Error
          ? err.message
          : "Failed to fetch marker statistics",
      );
    } finally {
      setLoading(false);
    }
  };

  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    return date.toLocaleDateString() + " " + date.toLocaleTimeString();
  };

  const formatDurationValue = (duration?: number) => {
    if (!Number.isFinite(duration)) {
      return "0s";
    }
    return humanizeDuration(duration as number, 1_000);
  };

  const calculateAggregateStats = (runs: RunStat[]) => {
    const stats: AggregateStats = {
      totalRuns: runs.length,
      totalTests: 0,
      avgTests: 0,
      passed: 0,
      failed: 0,
      skipped: 0,
      broken: 0,
      timedout: 0,
      interrupted: 0,
      unknown: 0,
      avgPassed: 0,
      avgFailed: 0,
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

    stats.avgTests =
      stats.totalRuns > 0 ? stats.totalTests / stats.totalRuns : 0;
    stats.avgPassed = stats.totalRuns > 0 ? stats.passed / stats.totalRuns : 0;
    stats.avgFailed = stats.totalRuns > 0 ? stats.failed / stats.totalRuns : 0;

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
      .sort(
        (a, b) =>
          new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime(),
      )
      .map((run, index) => {
        const date = new Date(run.createdAt);
        const passRate =
          run.total > 0 ? ((run.passed / run.total) * 100).toFixed(1) : 0;
        return {
          name: run.name || `Run ${index + 1}`,
          runId: run.runId.substring(0, 8),
          passed: run.passed,
          flaky: run.flaky,
          failed: run.failed,
          skipped: run.skipped,
          duration: run.duration,
          total: run.total,
          passRate: parseFloat(passRate as string),
          date: date.toLocaleDateString(),
          fullDate: date.toLocaleString(),
        };
      });
  };

  const CustomTooltip = ({ active, payload }: any) => {
    if (active && payload && payload.length) {
      const data = payload[0].payload;
      return (
        <div className="bg-(--stitch-surface-card) p-3 border border-(--stitch-outline) rounded-lg shadow-lg">
          <p className="font-semibold text-(--stitch-on-surface)">
            {data.name}
          </p>
          <p className="text-sm text-(--stitch-on-surface-muted)">
            Run ID: {data.runId}
          </p>
          <p className="text-sm text-(--status-success)">
            Passed: {data.passed}
          </p>
          <p className="text-sm text-(--status-warning)">Flaky: {data.flaky}</p>
          <p className="text-sm text-(--status-failure)">
            Failed: {data.failed}
          </p>
          <p className="text-sm text-(--stitch-on-surface-muted)">
            Skipped: {data.skipped}
          </p>
          <p className="text-sm text-(--stitch-primary)">Total: {data.total}</p>
          {data.duration !== undefined && (
            <p className="text-sm text-(--stitch-on-surface-muted)">
              Duration: {formatDurationValue(data.duration)}
            </p>
          )}
          <p className="text-sm text-(--stitch-on-surface-muted)">
            Pass Rate: {data.passRate}%
          </p>
          <p className="text-xs text-(--stitch-on-surface-muted) mt-1">
            {data.fullDate}
          </p>
        </div>
      );
    }
    return null;
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-(--stitch-on-surface-muted)">
          Loading marker statistics...
        </div>
      </div>
    );
  }

  if (error || !statsData) {
    return (
      <div className="space-y-4">
        <Link
          to="/"
          className="inline-flex items-center text-(--stitch-primary) hover:text-(--stitch-primary) focus:outline-none focus:ring-2 focus:ring-(--stitch-primary) focus:ring-offset-2 rounded-md px-2 py-1"
        >
          <ArrowLeft className="h-4 w-4 mr-2" />
          Back to Dashboard
        </Link>
        <Card>
          <CardContent className="py-12">
            <div className="flex flex-col items-center justify-center space-y-4">
              <AlertCircle className="h-16 w-16 text-(--status-failure)" />
              <div className="text-(--status-failure) text-center">
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
            className="inline-flex items-center text-(--stitch-primary) hover:text-(--stitch-primary) focus:outline-none focus:ring-2 focus:ring-(--stitch-primary) focus:ring-offset-2 rounded-md p-1"
            aria-label="Back to dashboard"
          >
            <ArrowLeft className="h-5 w-5" />
          </Link>
          <div>
            <div className="flex items-center space-x-2">
              <BarChart3 className="h-6 w-6 text-(--stitch-primary)" />
              <h1 className="text-3xl font-bold text-(--stitch-on-surface)">
                Marker Statistics
              </h1>
            </div>
            <p className="text-sm text-(--stitch-on-surface-muted) mt-1">
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
          <p className="font-mono text-sm text-(--stitch-on-surface) break-all">
            {markerValue}
          </p>
        </CardContent>
      </Card>

      {/* Aggregate Statistics Summary */}
      <div className="grid grid-cols-1 md:grid-cols-5 gap-4">
        <Card>
          <CardContent className="py-6">
            <div className="text-center">
              <p className="text-sm text-(--stitch-on-surface-muted) mb-1">
                Total Runs
              </p>
              <p className="text-3xl font-bold text-(--stitch-on-surface)">
                {aggregateStats.totalRuns}
              </p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="py-6">
            <div className="text-center">
              <p className="text-sm text-(--stitch-on-surface-muted) mb-1">
                Avg. Tests
              </p>
              <p className="text-3xl font-bold text-(--stitch-on-surface)">
                {aggregateStats.avgTests.toFixed(2)}
              </p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="py-6">
            <div className="text-center">
              <p className="text-sm text-(--stitch-on-surface-muted) mb-1">
                Pass Rate
              </p>
              <p className="text-3xl font-bold text-(--status-success)">
                {passRate}%
              </p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="py-6">
            <div className="text-center">
              <p className="text-sm text-(--stitch-on-surface-muted) mb-1">
                Avg. Passed
              </p>
              <p className="text-3xl font-bold text-(--status-success)">
                {aggregateStats.avgPassed.toFixed(2)}
              </p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="py-6">
            <div className="text-center">
              <p className="text-sm text-(--stitch-on-surface-muted) mb-1">
                Avg. Failed
              </p>
              <p className="text-3xl font-bold text-(--status-failure)">
                {aggregateStats.avgFailed.toFixed(2)}
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
              <TrendingUp className="h-5 w-5 text-(--stitch-primary)" />
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
                  <CartesianGrid
                    strokeDasharray="3 3"
                    className="stroke-gray-200"
                  />
                  <XAxis
                    dataKey="name"
                    tick={{ fontSize: 12 }}
                    className="text-(--stitch-on-surface-muted)"
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
                    className="text-(--stitch-on-surface-muted)"
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

      {/* Status Distribution Over Time */}
      {runs.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center space-x-2">
              <TrendingUp className="h-5 w-5 text-(--status-timedout)" />
              <span>Status Distribution Over Time</span>
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="w-full h-80">
              <ResponsiveContainer width="100%" height="100%">
                <AreaChart
                  data={prepareTimelineData(runs)}
                  margin={{ top: 5, right: 30, left: 20, bottom: 5 }}
                >
                  <CartesianGrid
                    strokeDasharray="3 3"
                    className="stroke-gray-200"
                  />
                  <XAxis
                    dataKey="name"
                    tick={{ fontSize: 12 }}
                    className="text-(--stitch-on-surface-muted)"
                  />
                  <YAxis
                    label={{
                      value: "Number of Tests",
                      angle: -90,
                      position: "insideLeft",
                      style: { fontSize: 12 },
                    }}
                    tick={{ fontSize: 12 }}
                    className="text-(--stitch-on-surface-muted)"
                  />
                  <Tooltip content={<CustomTooltip />} />
                  <Legend />
                  <Area
                    type="monotone"
                    dataKey="passed"
                    stackId={1}
                    stroke="#10b981"
                    fill="#10b981"
                    strokeWidth={2}
                    dot={{ fill: "#10b981", r: 4 }}
                    activeDot={{ r: 6 }}
                    name="Passed"
                  />
                  <Area
                    type="monotone"
                    dataKey="failed"
                    stackId={2}
                    stroke="#ef4444"
                    fill="#ef4444"
                    strokeWidth={2}
                    dot={{ fill: "#ef4444", r: 4 }}
                    activeDot={{ r: 6 }}
                    name="Failed"
                  />
                  <Area
                    type="monotone"
                    dataKey="flaky"
                    stackId={3}
                    stroke="#f59e0b"
                    fill="#f59e0b"
                    strokeWidth={2}
                    dot={{ fill: "#f59e0b", r: 4 }}
                    activeDot={{ r: 6 }}
                    name="Flaky"
                  />
                  <Area
                    type="monotone"
                    dataKey="skipped"
                    stackId={4}
                    stroke="#6b7280"
                    fill="#6b7280"
                    strokeWidth={2}
                    dot={{ fill: "#6b7280", r: 4 }}
                    activeDot={{ r: 6 }}
                    name="Skipped"
                  />
                </AreaChart>
              </ResponsiveContainer>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Test Duration graph */}
      {runs.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center space-x-2">
              <TrendingUp className="h-5 w-5 text-(--status-timedout)" />
              <span>Test Duration Over Time</span>
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="w-full h-80">
              <ResponsiveContainer width="100%" height="100%">
                <LineChart
                  data={prepareTimelineData(runs)}
                  margin={{ top: 5, right: 30, left: 20, bottom: 5 }}
                >
                  <CartesianGrid
                    strokeDasharray="3 3"
                    className="stroke-gray-200"
                  />
                  <XAxis
                    dataKey="name"
                    tick={{ fontSize: 12 }}
                    className="text-(--stitch-on-surface-muted)"
                  />
                  <YAxis
                    label={{
                      value: "Duration",
                      angle: -90,
                      position: "insideLeft",
                      style: { fontSize: 12 },
                    }}
                    tickFormatter={(value) =>
                      formatDurationValue(Number(value))
                    }
                    tick={{ fontSize: 12 }}
                    width={96}
                    className="text-(--stitch-on-surface-muted)"
                  />
                  <Tooltip content={<CustomTooltip />} />
                  <Legend />
                  <Line
                    type="monotone"
                    dataKey="duration"
                    stroke="#10b981"
                    strokeWidth={2}
                    dot={{ fill: "#10b981", r: 4 }}
                    activeDot={{ r: 6 }}
                    name="Duration"
                  />
                </LineChart>
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
            <div className="text-center py-8 text-(--stitch-on-surface-muted)">
              No runs found with MARKER = "{markerValue}"
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-(--stitch-outline)">
                <thead className="bg-(--stitch-surface-card)">
                  <tr>
                    <th className="px-6 py-3 text-left text-xs font-medium text-(--stitch-on-surface-muted) uppercase tracking-wider">
                      Name
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-(--stitch-on-surface-muted) uppercase tracking-wider">
                      Stats
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-(--stitch-on-surface-muted) uppercase tracking-wider">
                      Total
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-(--stitch-on-surface-muted) uppercase tracking-wider">
                      Duration
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-(--stitch-on-surface-muted) uppercase tracking-wider">
                      Created At
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-(--stitch-on-surface-muted) uppercase tracking-wider">
                      Actions
                    </th>
                  </tr>
                </thead>
                <tbody className="bg-(--stitch-surface-card) divide-y divide-(--stitch-outline)">
                  {runs.map((run) => {
                    const runPassRate =
                      run.total > 0
                        ? ((run.passed / run.total) * 100).toFixed(1)
                        : "0.0";
                    return (
                      <tr
                        key={run.runId}
                        className="hover:bg-(--stitch-surface-card)"
                      >
                        <td className="px-6 py-4 whitespace-nowrap">
                          <Link
                            to={`/runs/${run.runId}`}
                            className="font-mono text-sm text-(--stitch-primary) hover:underline focus:outline-none focus:ring-2 focus:ring-(--stitch-primary) focus:ring-offset-2 rounded"
                          >
                            {run.name || run.runId.substring(0, 12) + "..."}
                          </Link>
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap">
                          <div className="text-sm text-(--stitch-on-surface)">
                            <span className="text-(--status-success) font-semibold">
                              {run.passed}
                            </span>
                            {" / "}
                            <span className="text-(--status-warning) font-semibold">
                              {run.flaky}
                            </span>
                            {" / "}
                            <span className="text-(--status-failure) font-semibold">
                              {run.failed}
                            </span>
                            {" / "}
                            <span className="text-(--stitch-on-surface-muted) font-semibold">
                              {run.skipped}
                            </span>
                          </div>
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-(--stitch-on-surface)">
                          <span className="text-(--stitch-primary) font-semibold">
                            {run.total}
                          </span>
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-(--stitch-on-surface)">
                          {runPassRate}%
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-(--stitch-on-surface)">
                          {humanizeDuration(run.duration!, 1_000)}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-(--stitch-on-surface-muted)">
                          {formatDate(run.createdAt)}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm">
                          <Link
                            to={`/runs/${run.runId}`}
                            className="text-(--stitch-primary) hover:text-(--stitch-primary) hover:underline focus:outline-none focus:ring-2 focus:ring-(--stitch-primary) focus:ring-offset-2 rounded px-2 py-1"
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
