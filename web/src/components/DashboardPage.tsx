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
        <div className="text-gray-600">Loading dashboard...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center h-64 space-y-4">
        <AlertTriangle className="h-12 w-12 text-red-500" />
        <div className="text-red-600">Error: {error}</div>
        <button
          onClick={fetchDashboardStats}
          className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors"
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
          <h1 className="text-3xl font-bold text-gray-900">Dashboard</h1>
          <p className="text-gray-600 mt-1">
            Overview of your test execution health and recent activity
          </p>
        </div>
        <button
          onClick={fetchDashboardStats}
          className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors"
        >
          Refresh
        </button>
      </div>

      {!hasData ? (
        /* Empty State */
        <Card>
          <CardContent className="py-12">
            <div className="text-center">
              <Activity className="mx-auto h-16 w-16 text-gray-400" />
              <h3 className="mt-4 text-lg font-medium text-gray-900">
                Welcome to Observer
              </h3>
              <p className="mt-2 text-gray-500 max-w-md mx-auto">
                No test runs detected yet. Start running your tests to see
                real-time observability data and analytics here.
              </p>
              <div className="mt-6">
                <Link
                  to="/suite_runs"
                  className="inline-flex items-center px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors"
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
                    <p className="text-sm font-medium text-gray-600">
                      Total Runs
                    </p>
                    <p className="mt-2 text-3xl font-bold text-gray-900">
                      {stats.totalRuns}
                    </p>
                  </div>
                  <Activity className="h-12 w-12 text-blue-500 opacity-75" />
                </div>
              </CardContent>
            </Card>

            {/* Total Tests */}
            <Card>
              <CardContent className="pt-6">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium text-gray-600">
                      Total Tests
                    </p>
                    <p className="mt-2 text-3xl font-bold text-gray-900">
                      {stats.totalTests}
                    </p>
                  </div>
                  <Clock className="h-12 w-12 text-purple-500 opacity-75" />
                </div>
              </CardContent>
            </Card>

            {/* Pass Rate */}
            <Card>
              <CardContent className="pt-6">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium text-gray-600">
                      Pass Rate
                    </p>
                    <p className="mt-2 text-3xl font-bold text-gray-900">
                      {stats.passRate}%
                    </p>
                  </div>
                  <TrendingUp
                    className={`h-12 w-12 opacity-75 ${
                      stats.passRate >= 90
                        ? "text-green-500"
                        : stats.passRate >= 70
                        ? "text-yellow-500"
                        : "text-red-500"
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
                    <p className="text-sm font-medium text-gray-600">
                      Test Health
                    </p>
                    <p className="mt-2 text-xl font-semibold">
                      {stats.passRate >= 95 ? (
                        <span className="text-green-600">Excellent</span>
                      ) : stats.passRate >= 80 ? (
                        <span className="text-blue-600">Good</span>
                      ) : stats.passRate >= 60 ? (
                        <span className="text-yellow-600">Fair</span>
                      ) : (
                        <span className="text-red-600">Needs Attention</span>
                      )}
                    </p>
                  </div>
                  {stats.passRate >= 80 ? (
                    <CheckCircle className="h-12 w-12 text-green-500 opacity-75" />
                  ) : (
                    <XCircle className="h-12 w-12 text-red-500 opacity-75" />
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
                  to="/suite_runs"
                  className="text-sm text-blue-600 hover:text-blue-700 font-medium"
                >
                  View All →
                </Link>
              </div>
            </CardHeader>
            <CardContent className="p-0">
              {stats.recentRuns.length === 0 ? (
                <div className="text-center py-8 text-gray-500">
                  No recent runs available
                </div>
              ) : (
                <div className="overflow-x-auto">
                  <table className="min-w-full divide-y divide-gray-200">
                    <thead className="bg-gray-50">
                      <tr>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                          Run ID
                        </th>
                        <th className="px-6 py-3 text-center text-xs font-medium text-gray-500 uppercase tracking-wider">
                          Total
                        </th>
                        <th className="px-6 py-3 text-center text-xs font-medium text-gray-500 uppercase tracking-wider">
                          <div className="flex items-center justify-center">
                            <CheckCircle className="h-4 w-4 mr-1 text-green-600" />
                            Passed
                          </div>
                        </th>
                        <th className="px-6 py-3 text-center text-xs font-medium text-gray-500 uppercase tracking-wider">
                          <div className="flex items-center justify-center">
                            <XCircle className="h-4 w-4 mr-1 text-red-600" />
                            Failed
                          </div>
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                          Last Updated
                        </th>
                      </tr>
                    </thead>
                    <tbody className="bg-white divide-y divide-gray-200">
                      {stats.recentRuns.map((run) => (
                        <tr
                          key={run.runId}
                          className="hover:bg-gray-50 transition-colors"
                        >
                          <td className="px-6 py-4 whitespace-nowrap">
                            <Link
                              to={`/suite_runs/${run.runId}`}
                              className="text-blue-600 hover:text-blue-800 font-medium"
                            >
                              {run.runId}
                            </Link>
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-center">
                            <span className="text-gray-900 font-semibold">
                              {run.total}
                            </span>
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-center">
                            <span className="text-green-600 font-semibold">
                              {run.passed}
                            </span>
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-center">
                            <span className="text-red-600 font-semibold">
                              {run.failed}
                            </span>
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                            {run.lastUpdated ? (
                              <div className="flex flex-col">
                                <span>
                                  {new Date(run.lastUpdated).toLocaleDateString()}
                                </span>
                                <span className="text-xs text-gray-400">
                                  {new Date(run.lastUpdated).toLocaleTimeString()}
                                </span>
                              </div>
                            ) : (
                              <span className="text-gray-400">N/A</span>
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
                  to="/suite_runs"
                  className="flex items-center justify-center px-4 py-3 border border-gray-300 rounded-lg hover:border-blue-500 hover:shadow-md transition-all"
                >
                  <Activity className="h-5 w-5 mr-2 text-blue-600" />
                  <span className="font-medium text-gray-900">
                    View All Runs
                  </span>
                </Link>
                <button
                  onClick={fetchDashboardStats}
                  className="flex items-center justify-center px-4 py-3 border border-gray-300 rounded-lg hover:border-blue-500 hover:shadow-md transition-all"
                >
                  <Clock className="h-5 w-5 mr-2 text-purple-600" />
                  <span className="font-medium text-gray-900">
                    Refresh Data
                  </span>
                </button>
                <a
                  href="https://github.com/stanterprise/observer"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex items-center justify-center px-4 py-3 border border-gray-300 rounded-lg hover:border-blue-500 hover:shadow-md transition-all"
                >
                  <span className="font-medium text-gray-900">
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
