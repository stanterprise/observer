import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { apiUrl } from "../lib/config";
import { Card, CardHeader, CardTitle, CardContent } from "./Card";
import {
  Activity,
  CheckCircle,
  XCircle,
  Clock,
  TrendingUp,
  AlertTriangle,
} from "lucide-react";

interface RunStats {
  runId: string;
  total: number;
  passed: number;
  failed: number;
  skipped: number;
  running?: number;
  broken?: number;
  timedout?: number;
  interrupted?: number;
  unknown?: number;
  lastUpdated?: string;
}

interface DashboardStats {
  totalRuns: number;
  totalTests: number;
  passRate: number;
  recentRuns: RunStats[];
}

export default function DashboardPage() {
  const [stats, setStats] = useState<DashboardStats>({
    totalRuns: 0,
    totalTests: 0,
    passRate: 0,
    recentRuns: [],
  });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetchDashboardStats();
  }, []);

  const fetchDashboardStats = async () => {
    try {
      setLoading(true);
      const response = await fetch(apiUrl("/runs/stats"));
      if (!response.ok) {
        throw new Error(`Failed to fetch stats: ${response.statusText}`);
      }
      const data = await response.json();
      const runs = (data.runs || []) as RunStats[];

      // Calculate aggregate statistics
      const totalTests = runs.reduce((sum, run) => sum + run.total, 0);
      const totalPassed = runs.reduce((sum, run) => sum + run.passed, 0);
      const passRate =
        totalTests > 0 ? Math.round((totalPassed / totalTests) * 100) : 0;

      // Sort runs by lastUpdated (most recent first)
      const sortedRuns = [...runs].sort((a, b) => {
        const aTime = a.lastUpdated ? new Date(a.lastUpdated).getTime() : 0;
        const bTime = b.lastUpdated ? new Date(b.lastUpdated).getTime() : 0;
        return bTime - aTime;
      });

      setStats({
        totalRuns: runs.length,
        totalTests,
        passRate,
        recentRuns: sortedRuns.slice(0, 5), // Top 5 recent runs
      });
      setError(null);
    } catch (err) {
      console.error("Error fetching dashboard stats:", err);
      setError(err instanceof Error ? err.message : "Failed to fetch stats");
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-(--stitch-on-surface-muted)">
          Loading dashboard...
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center h-64 space-y-4">
        <AlertTriangle className="h-12 w-12 text-(--stitch-error)" />
        <div className="text-(--stitch-error)">Error: {error}</div>
        <button
          onClick={fetchDashboardStats}
          className="rounded-md px-4 py-2 text-white transition-colors"
          style={{
            background:
              "linear-gradient(135deg, var(--stitch-primary), var(--stitch-primary-end))",
          }}
        >
          Retry
        </button>
      </div>
    );
  }

  const hasData = stats.totalRuns > 0;

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold font-headline text-(--stitch-on-surface)">
            Dashboard
          </h1>
          <p className="mt-1 text-(--stitch-on-surface-muted)">
            Overview of your test execution health and recent activity
          </p>
        </div>
        <button
          onClick={fetchDashboardStats}
          className="rounded-md px-4 py-2 text-white transition-opacity hover:opacity-90"
          style={{
            background:
              "linear-gradient(135deg, var(--stitch-primary), var(--stitch-primary-end))",
          }}
        >
          Refresh
        </button>
      </div>

      {!hasData ? (
        /* Empty State */
        <Card>
          <CardContent className="py-12">
            <div className="text-center">
              <Activity className="mx-auto h-16 w-16 text-(--stitch-on-surface-subtle)" />
              <h3 className="mt-4 text-lg font-medium text-(--stitch-on-surface)">
                Welcome to Observer
              </h3>
              <p className="mx-auto mt-2 max-w-md text-(--stitch-on-surface-muted)">
                No test runs detected yet. Start running your tests to see
                real-time observability data and analytics here.
              </p>
              <div className="mt-6">
                <Link
                  to="/runs"
                  className="inline-flex items-center rounded-md px-4 py-2 text-white transition-opacity hover:opacity-90"
                  style={{
                    background:
                      "linear-gradient(135deg, var(--stitch-primary), var(--stitch-primary-end))",
                  }}
                >
                  View Test Runs
                </Link>
              </div>
            </div>
          </CardContent>
        </Card>
      ) : (
        <>
          {/* Statistics Cards */}
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
            {/* Total Test Runs */}
            <Card>
              <CardContent className="pt-6">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium text-(--stitch-on-surface-muted)">
                      Total Runs
                    </p>
                    <p className="mt-2 text-3xl font-bold text-(--stitch-on-surface)">
                      {stats.totalRuns}
                    </p>
                  </div>
                  <Activity className="h-12 w-12 text-(--stitch-primary) opacity-75" />
                </div>
              </CardContent>
            </Card>

            {/* Total Tests */}
            <Card>
              <CardContent className="pt-6">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium text-(--stitch-on-surface-muted)">
                      Total Tests
                    </p>
                    <p className="mt-2 text-3xl font-bold text-(--stitch-on-surface)">
                      {stats.totalTests}
                    </p>
                  </div>
                  <Clock className="h-12 w-12 text-(--stitch-tertiary) opacity-75" />
                </div>
              </CardContent>
            </Card>

            {/* Pass Rate */}
            <Card>
              <CardContent className="pt-6">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium text-(--stitch-on-surface-muted)">
                      Pass Rate
                    </p>
                    <p className="mt-2 text-3xl font-bold text-(--stitch-on-surface)">
                      {stats.passRate}%
                    </p>
                  </div>
                  <TrendingUp
                    className={`h-12 w-12 opacity-75 ${
                      stats.passRate >= 90
                        ? "text-(--status-success)"
                        : stats.passRate >= 70
                          ? "text-(--status-warning)"
                          : "text-(--status-failure)"
                    }`}
                  />
                </div>
              </CardContent>
            </Card>

            {/* Status Indicator */}
            <Card>
              <CardContent className="pt-6">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium text-(--stitch-on-surface-muted)">
                      Test Health
                    </p>
                    <p className="mt-2 text-xl font-semibold">
                      {stats.passRate >= 95 ? (
                        <span className="text-(--status-success)">
                          Excellent
                        </span>
                      ) : stats.passRate >= 80 ? (
                        <span className="text-(--stitch-primary)">Good</span>
                      ) : stats.passRate >= 60 ? (
                        <span className="text-(--status-warning)">Fair</span>
                      ) : (
                        <span className="text-(--status-failure)">
                          Needs Attention
                        </span>
                      )}
                    </p>
                  </div>
                  {stats.passRate >= 80 ? (
                    <CheckCircle className="h-12 w-12 text-(--status-success) opacity-75" />
                  ) : (
                    <XCircle className="h-12 w-12 text-(--status-failure) opacity-75" />
                  )}
                </div>
              </CardContent>
            </Card>
          </div>

          {/* Recent Test Runs */}
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle>Recent Test Runs</CardTitle>
                <Link
                  to="/runs"
                  className="text-sm font-medium text-(--stitch-primary) hover:opacity-90"
                >
                  View All →
                </Link>
              </div>
            </CardHeader>
            <CardContent className="p-0">
              {stats.recentRuns.length === 0 ? (
                <div className="py-8 text-center text-(--stitch-on-surface-muted)">
                  No recent runs available
                </div>
              ) : (
                <div className="overflow-x-auto">
                  <table className="min-w-full divide-y divide-(--stitch-outline)">
                    <thead className="bg-(--stitch-surface-low)">
                      <tr>
                        <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-(--stitch-on-surface-subtle)">
                          Run ID
                        </th>
                        <th className="px-6 py-3 text-center text-xs font-medium uppercase tracking-wider text-(--stitch-on-surface-subtle)">
                          Total
                        </th>
                        <th className="px-6 py-3 text-center text-xs font-medium uppercase tracking-wider text-(--stitch-on-surface-subtle)">
                          <div className="flex items-center justify-center">
                            <CheckCircle className="mr-1 h-4 w-4 text-(--status-success)" />
                            Passed
                          </div>
                        </th>
                        <th className="px-6 py-3 text-center text-xs font-medium uppercase tracking-wider text-(--stitch-on-surface-subtle)">
                          <div className="flex items-center justify-center">
                            <XCircle className="mr-1 h-4 w-4 text-(--status-failure)" />
                            Failed
                          </div>
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-(--stitch-on-surface-subtle)">
                          Last Updated
                        </th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-(--stitch-outline) bg-(--stitch-surface-card)">
                      {stats.recentRuns.map((run) => (
                        <tr
                          key={run.runId}
                          className="transition-colors hover:bg-(--stitch-surface-low)"
                        >
                          <td className="px-6 py-4 whitespace-nowrap">
                            <Link
                              to={`/runs/${run.runId}`}
                              className="font-medium text-(--stitch-primary) hover:opacity-90"
                            >
                              {run.runId}
                            </Link>
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-center">
                            <span className="font-semibold text-(--stitch-on-surface)">
                              {run.total}
                            </span>
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-center">
                            <span className="font-semibold text-(--status-success)">
                              {run.passed}
                            </span>
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-center">
                            <span className="font-semibold text-(--status-failure)">
                              {run.failed}
                            </span>
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-sm text-(--stitch-on-surface-muted)">
                            {run.lastUpdated ? (
                              <div className="flex flex-col">
                                <span>
                                  {new Date(
                                    run.lastUpdated,
                                  ).toLocaleDateString()}
                                </span>
                                <span className="text-xs text-(--stitch-on-surface-subtle)">
                                  {new Date(
                                    run.lastUpdated,
                                  ).toLocaleTimeString()}
                                </span>
                              </div>
                            ) : (
                              <span className="text-(--stitch-on-surface-subtle)">
                                N/A
                              </span>
                            )}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </CardContent>
          </Card>

          {/* Quick Actions */}
          <Card>
            <CardHeader>
              <CardTitle>Quick Actions</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <Link
                  to="/runs"
                  className="flex items-center justify-center rounded-lg border px-4 py-3 transition-all"
                  style={{
                    borderColor: "var(--stitch-outline)",
                  }}
                >
                  <Activity className="mr-2 h-5 w-5 text-(--stitch-primary)" />
                  <span className="font-medium text-(--stitch-on-surface)">
                    View All Runs
                  </span>
                </Link>
                <button
                  onClick={fetchDashboardStats}
                  className="flex items-center justify-center rounded-lg border px-4 py-3 transition-all"
                  style={{
                    borderColor: "var(--stitch-outline)",
                  }}
                >
                  <Clock className="mr-2 h-5 w-5 text-(--stitch-tertiary)" />
                  <span className="font-medium text-(--stitch-on-surface)">
                    Refresh Data
                  </span>
                </button>
                <a
                  href="https://github.com/stanterprise/observer"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex items-center justify-center rounded-lg border px-4 py-3 transition-all"
                  style={{
                    borderColor: "var(--stitch-outline)",
                  }}
                >
                  <span className="font-medium text-(--stitch-on-surface)">
                    Documentation
                  </span>
                </a>
              </div>
            </CardContent>
          </Card>
        </>
      )}
    </div>
  );
}
