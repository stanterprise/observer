import { useEffect, useState, useCallback, useMemo } from "react";
import { useParams, Link } from "react-router-dom";
import { apiUrl, config } from "@/lib/config";
import { Card, CardContent } from "@/components/Card";
import { ArrowLeft, Play, Eye, EyeOff, Search, X, Filter } from "lucide-react";
import { cn } from "@/lib/utils";

import { SuiteTitleCard } from "./SuiteTitleCard";
import type { TestStatus } from "@/types/common";
import { assembleSuiteHierarchy } from "../TestSuiteRunsPage/utils";
import type { TestSuite } from "@/types/testSuite";
import TestSuiteRecord from "./TestSuiteRecord";
import type { TestRun } from "@/types/testRun";

export function TestRunDetailPage() {
  const pollIntervalMs = config.pollingIntervalMs;
  const { runId } = useParams<{ runId: string }>();
  const [runDetail, setRunDetail] = useState<TestRun | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [hiddenSuiteTypes, setHiddenSuiteTypes] = useState<Set<string>>(
    new Set(["ROOT", "PROJECT", "FILE"]),
  );

  // Filter state
  const [searchText, setSearchText] = useState("");
  const [selectedStatuses, setSelectedStatuses] = useState<Set<TestStatus>>(
    new Set(),
  );
  const [selectedTags, setSelectedTags] = useState<Set<string>>(new Set());

  const fetchRunDetail = useCallback(
    async (id: string, options?: { silent?: boolean }) => {
      const silent = options?.silent ?? false;
      try {
        if (!silent) {
          setLoading(true);
        }
        const response = await fetch(apiUrl(`/runs/${id}`));
        if (!response.ok) {
          throw new Error(
            `Failed to fetch run details: ${response.statusText}`,
          );
        }
        const data = await response.json();

        data.statistics = {
          total: data.tests.length,
          passed: data.tests.filter((t: any) => t.status === "PASSED").length,
          failed: data.tests.filter((t: any) => t.status === "FAILED").length,
          skipped: data.tests.filter((t: any) => t.status === "SKIPPED").length,
          running: data.tests.filter((t: any) => t.status === "RUNNING").length,
          broken: data.tests.filter((t: any) => t.status === "BROKEN").length,
          timedout: data.tests.filter((t: any) => t.status === "TIMEDOUT")
            .length,
          interrupted: data.tests.filter((t: any) => t.status === "INTERRUPTED")
            .length,
          unknown: data.tests.filter((t: any) => t.status === "UNKNOWN").length,
          expected: data.tests.filter(
            (t: any) => t.status === "PASSED" && t.attempts.length === 1,
          ).length,
          flaky: data.tests.filter(
            (t: any) => t.status === "PASSED" && t.attempts.length > 1,
          ).length,
        };
        console.log("Fetched run details:", data);

        setRunDetail(data);
        setError(null);
      } catch (err) {
        console.error("Error fetching run details:", err);
        setError(
          err instanceof Error ? err.message : "Failed to fetch run details",
        );
      } finally {
        if (!silent) {
          setLoading(false);
        }
      }
    },
    [],
  );

  const countTests = useCallback((suites: TestSuite[]): number => {
    let total = 0;
    for (const suite of suites) {
      total += suite.tests?.length || 0; // API returns 'tests' (lowercase)
      total += countTests(suite.suites ?? []);
    }

    return total;
  }, []);

  useEffect(() => {
    if (runId) {
      fetchRunDetail(runId);
    }
  }, [runId, fetchRunDetail]);

  useEffect(() => {
    if (!runId) return;
    const intervalId = window.setInterval(() => {
      fetchRunDetail(runId, { silent: true });
    }, pollIntervalMs);

    return () => {
      window.clearInterval(intervalId);
    };
  }, [runId, fetchRunDetail, pollIntervalMs]);

  // Compute root suite hierarchy (must be computed before early returns)
  const rootSuite = useMemo(() => {
    if (!runDetail) {
      return {
        id: "",
        name: "",
        type: "",
        runId: runId || "",
        tests: [],
        suites: [],
      } as TestSuite;
    }
    return assembleSuiteHierarchy(
      runDetail.suites || [],
      runDetail.tests || [],
    );
  }, [runDetail, runId]);

  // Get unique suite types from the hierarchy
  const availableSuiteTypes = useMemo(() => {
    const getSuiteTypes = (suite: TestSuite): Set<string> => {
      const types = new Set<string>();
      if (suite.type) types.add(suite.type.toUpperCase());
      suite.suites?.forEach((s) => {
        getSuiteTypes(s).forEach((t) => types.add(t));
      });
      return types;
    };
    return Array.from(getSuiteTypes(rootSuite)).sort();
  }, [rootSuite]);

  // Extract all unique tags from tests
  const allTags = useMemo(() => {
    if (!runDetail?.tests) return [];
    const tagSet = new Set<string>();
    runDetail.tests.forEach((test) => {
      test.tags?.forEach((tag) => tagSet.add(tag));
    });
    return Array.from(tagSet).sort();
  }, [runDetail?.tests]);

  // Check if any filters are currently active
  const hasActiveFilters = useMemo(() => {
    return (
      searchText.trim() !== "" ||
      selectedStatuses.size > 0 ||
      selectedTags.size > 0
    );
  }, [searchText, selectedStatuses, selectedTags]);

  // Filter tests based on current filters
  // Note: This filtering modifies the suite hierarchy structure for display purposes only.
  // Tests are still updated in runDetail.tests (the source data), but won't appear in
  // filteredSuite until filters are changed to include them.
  const filteredSuite = useMemo(() => {
    if (!hasActiveFilters) return rootSuite;

    // Helper to check if a test matches all filters
    type SuiteTest = NonNullable<TestSuite["tests"]>[number];
    const testMatchesFilters = (test: SuiteTest): boolean => {
      // Search text filter
      if (searchText.trim() !== "") {
        const search = searchText.toLowerCase();
        const matchesTitle = test.title?.toLowerCase().includes(search);
        const matchesId = test.id?.toLowerCase().includes(search);
        if (!matchesTitle && !matchesId) return false;
      }

      // Status filter
      if (selectedStatuses.size > 0) {
        if (!selectedStatuses.has(test.status as TestStatus)) return false;
      }

      // Tag filter (test must have ALL selected tags)
      if (selectedTags.size > 0) {
        if (!test.tags || test.tags.length === 0) return false;
        const testTags = new Set(test.tags);
        for (const tag of selectedTags) {
          if (!testTags.has(tag)) return false;
        }
      }

      return true;
    };

    // Recursively filter suite hierarchy
    const filterSuite = (suite: TestSuite): TestSuite => {
      const filteredTests = suite.tests?.filter(testMatchesFilters) || [];
      const filteredSubsuites =
        suite.suites
          ?.map(filterSuite)
          .filter(
            (s) => (s.tests?.length || 0) > 0 || (s.suites?.length || 0) > 0,
          ) || [];

      return {
        ...suite,
        tests: filteredTests,
        suites: filteredSubsuites,
      };
    };

    return filterSuite(rootSuite);
  }, [rootSuite, hasActiveFilters, searchText, selectedStatuses, selectedTags]);

  // Count filtered tests
  const filteredTestCount = useMemo(() => {
    return countTests([filteredSuite]);
  }, [filteredSuite, countTests]);

  if (loading) {
    return (
      <div className="space-y-6 animate-in fade-in duration-300">
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-4">
            <div className="h-10 w-10 bg-gray-200 rounded-lg animate-pulse" />
            <div className="h-8 w-48 bg-gray-200 rounded animate-pulse" />
          </div>
        </div>
        <div className="bg-white rounded-lg shadow-md border border-gray-200 overflow-hidden">
          <div className="h-8 bg-gray-200 animate-pulse" />
          <div className="p-6 space-y-4">
            <div className="h-6 bg-gray-200 rounded w-3/4 animate-pulse" />
            <div className="grid grid-cols-2 md:grid-cols-4 gap-6">
              {[1, 2, 3, 4].map((i) => (
                <div
                  key={i}
                  className="h-32 bg-gray-100 rounded-lg animate-pulse"
                />
              ))}
            </div>
          </div>
        </div>
        <div className="space-y-3">
          {[1, 2, 3].map((i) => (
            <div
              key={i}
              className="h-24 bg-gray-100 rounded-lg animate-pulse"
            />
          ))}
        </div>
      </div>
    );
  }

  if (error || !runDetail) {
    return (
      <div className="space-y-6 animate-in fade-in duration-300">
        <Link
          to="/suite_runs"
          className="inline-flex items-center gap-2 text-blue-600 hover:text-blue-700 transition-colors group"
        >
          <ArrowLeft className="h-5 w-5 group-hover:-translate-x-1 transition-transform" />
          <span className="font-medium">Back to Test Runs</span>
        </Link>
        <Card className="border-red-200 bg-red-50/50">
          <CardContent className="py-12">
            <div className="text-center max-w-md mx-auto">
              <div className="mx-auto h-16 w-16 rounded-full bg-red-100 flex items-center justify-center mb-4">
                <svg
                  className="h-8 w-8 text-red-600"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
                  />
                </svg>
              </div>
              <h3 className="text-lg font-semibold text-gray-900 mb-2">
                {error ? "Failed to Load Test Run" : "Test Run Not Found"}
              </h3>
              <p className="text-sm text-gray-600 mb-6">
                {error ||
                  "The test run you're looking for doesn't exist or has been deleted."}
              </p>
              <Link
                to="/suite_runs"
                className="inline-flex items-center px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
              >
                View All Test Runs
              </Link>
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  const overallStatus: TestStatus =
    runDetail.statistics!.running && runDetail.statistics!.running > 0
      ? "RUNNING"
      : runDetail.statistics!.failed > 0
        ? "FAILED"
        : runDetail.statistics!.passed === runDetail.statistics!.total &&
            runDetail.statistics!.total > 0
          ? "PASSED"
          : runDetail.statistics!.skipped === runDetail.statistics!.total &&
              runDetail.statistics!.total > 0
            ? "SKIPPED"
            : "NOT_RUN";
  console.log(
    "Rendering run detail:",
    runDetail,
    "Overall status:",
    overallStatus,
  );

  const toggleSuiteType = (type: string) => {
    setHiddenSuiteTypes((prev) => {
      const newSet = new Set(prev);
      if (newSet.has(type)) {
        newSet.delete(type);
      } else {
        newSet.add(type);
      }
      return newSet;
    });
  };

  // Filter helper functions
  const toggleStatus = (status: TestStatus) => {
    setSelectedStatuses((prev) => {
      const newSet = new Set(prev);
      if (newSet.has(status)) {
        newSet.delete(status);
      } else {
        newSet.add(status);
      }
      return newSet;
    });
  };

  const toggleTag = (tag: string) => {
    setSelectedTags((prev) => {
      const newSet = new Set(prev);
      if (newSet.has(tag)) {
        newSet.delete(tag);
      } else {
        newSet.add(tag);
      }
      return newSet;
    });
  };

  const clearFilters = () => {
    setSearchText("");
    setSelectedStatuses(new Set());
    setSelectedTags(new Set());
  };

  return (
    <div className="space-y-6 pb-8 animate-in fade-in duration-300">
      {/* Header with improved visual design */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Link
            to="/suite_runs"
            className="inline-flex items-center justify-center h-10 w-10 rounded-lg bg-white border border-gray-200 text-gray-700 hover:bg-gray-50 hover:border-gray-300 transition-all shadow-sm hover:shadow group"
            aria-label="Back to test runs"
          >
            <ArrowLeft className="h-5 w-5 group-hover:-translate-x-0.5 transition-transform" />
          </Link>
          <div>
            <h1 className="text-2xl md:text-3xl font-bold text-gray-900 tracking-tight">
              Test Suite Run
            </h1>
            <p className="text-sm text-gray-500 mt-1">
              {runDetail.name || runDetail.id}
            </p>
          </div>
        </div>
      </div>

      {/* Run Summary Card with improved spacing */}
      <div className="transition-all duration-300">
        <SuiteTitleCard runDetail={runDetail} overallStatus={overallStatus} />
      </div>

      {/* Test Cases List with enhanced design */}
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-xl font-semibold text-gray-900">
            Test Cases
            {hasActiveFilters && (
              <span className="ml-2 text-sm font-normal text-gray-500">
                ({filteredTestCount} of {runDetail.statistics?.total || 0})
              </span>
            )}
          </h2>
          {availableSuiteTypes.length > 0 && (
            <div className="flex items-center gap-2">
              <span className="text-sm text-gray-600 font-medium">Suites:</span>
              <div className="flex gap-2">
                {availableSuiteTypes.map((type) => {
                  const isHidden = hiddenSuiteTypes.has(type);
                  return (
                    <button
                      key={type}
                      onClick={() => toggleSuiteType(type)}
                      className={cn(
                        "inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm font-medium transition-all border",
                        isHidden
                          ? "bg-gray-100 text-gray-500 border-gray-200 hover:bg-gray-200"
                          : "bg-blue-50 text-blue-700 border-blue-200 hover:bg-blue-100",
                      )}
                      aria-label={`${
                        isHidden ? "Show" : "Hide"
                      } ${type} suites`}
                    >
                      {isHidden ? (
                        <EyeOff className="h-4 w-4" />
                      ) : (
                        <Eye className="h-4 w-4" />
                      )}
                      {type}
                    </button>
                  );
                })}
              </div>
            </div>
          )}
        </div>

        {/* Filter Panel */}
        <Card className="border-gray-200 bg-gray-50/50">
          <CardContent className="py-4">
            <div className="space-y-4">
              {/* Search Bar */}
              <div className="relative">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-5 w-5 text-gray-400" />
                <input
                  type="text"
                  placeholder="Search test cases by name or ID..."
                  value={searchText}
                  onChange={(e) => setSearchText(e.target.value)}
                  className="w-full pl-10 pr-10 py-2.5 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 transition-all"
                />
                {searchText && (
                  <button
                    onClick={() => setSearchText("")}
                    className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 transition-colors"
                    aria-label="Clear search"
                  >
                    <X className="h-5 w-5" />
                  </button>
                )}
              </div>

              {/* Status Filters */}
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <label className="text-sm font-medium text-gray-700 flex items-center gap-2">
                    <Filter className="h-4 w-4" />
                    Filter by Status
                  </label>
                  {hasActiveFilters && (
                    <button
                      onClick={clearFilters}
                      className="text-sm text-blue-600 hover:text-blue-700 font-medium transition-colors"
                    >
                      Clear All Filters
                    </button>
                  )}
                </div>
                <div className="flex flex-wrap gap-2">
                  {(
                    [
                      "PASSED",
                      "FAILED",
                      "RUNNING",
                      "SKIPPED",
                      "BROKEN",
                      "TIMEDOUT",
                      "INTERRUPTED",
                      "PENDING",
                      "UNKNOWN",
                      "NOT_RUN",
                    ] as TestStatus[]
                  ).map((status) => {
                    const isSelected = selectedStatuses.has(status);
                    const statusColors: Record<string, string> = {
                      PASSED: isSelected
                        ? "bg-green-100 border-green-300 text-green-800"
                        : "bg-white border-green-200 text-green-600 hover:bg-green-50",
                      FAILED: isSelected
                        ? "bg-red-100 border-red-300 text-red-800"
                        : "bg-white border-red-200 text-red-600 hover:bg-red-50",
                      RUNNING: isSelected
                        ? "bg-blue-100 border-blue-300 text-blue-800"
                        : "bg-white border-blue-200 text-blue-600 hover:bg-blue-50",
                      SKIPPED: isSelected
                        ? "bg-gray-100 border-gray-300 text-gray-800"
                        : "bg-white border-gray-200 text-gray-600 hover:bg-gray-50",
                      BROKEN: isSelected
                        ? "bg-orange-100 border-orange-300 text-orange-800"
                        : "bg-white border-orange-200 text-orange-600 hover:bg-orange-50",
                      TIMEDOUT: isSelected
                        ? "bg-purple-100 border-purple-300 text-purple-800"
                        : "bg-white border-purple-200 text-purple-600 hover:bg-purple-50",
                      INTERRUPTED: isSelected
                        ? "bg-yellow-100 border-yellow-300 text-yellow-800"
                        : "bg-white border-yellow-200 text-yellow-600 hover:bg-yellow-50",
                      PENDING: isSelected
                        ? "bg-amber-100 border-amber-300 text-amber-800"
                        : "bg-white border-amber-200 text-amber-600 hover:bg-amber-50",
                      UNKNOWN: isSelected
                        ? "bg-slate-100 border-slate-300 text-slate-800"
                        : "bg-white border-slate-200 text-slate-600 hover:bg-slate-50",
                      NOT_RUN: isSelected
                        ? "bg-zinc-100 border-zinc-300 text-zinc-800"
                        : "bg-white border-zinc-200 text-zinc-600 hover:bg-zinc-50",
                    };
                    return (
                      <button
                        key={status}
                        onClick={() => toggleStatus(status)}
                        className={cn(
                          "px-3 py-1.5 rounded-md text-sm font-medium transition-all border-2",
                          statusColors[status] ||
                            "bg-white border-gray-200 text-gray-600",
                        )}
                      >
                        {status}
                      </button>
                    );
                  })}
                </div>
              </div>

              {/* Tag Filters */}
              {allTags.length > 0 && (
                <div className="space-y-2">
                  <label className="text-sm font-medium text-gray-700">
                    Filter by Tags
                  </label>
                  <div className="flex flex-wrap gap-2">
                    {allTags.map((tag) => {
                      const isSelected = selectedTags.has(tag);
                      return (
                        <button
                          key={tag}
                          onClick={() => toggleTag(tag)}
                          className={cn(
                            "px-3 py-1.5 rounded-md text-sm font-medium transition-all border",
                            isSelected
                              ? "bg-indigo-100 border-indigo-300 text-indigo-800"
                              : "bg-white border-gray-200 text-gray-600 hover:bg-gray-50",
                          )}
                        >
                          {tag}
                        </button>
                      );
                    })}
                  </div>
                </div>
              )}
            </div>
          </CardContent>
        </Card>

        {!runDetail.tests || runDetail.tests.length === 0 ? (
          <Card className="border-dashed">
            <CardContent className="py-16">
              <div className="text-center max-w-sm mx-auto">
                <div className="mx-auto h-16 w-16 rounded-full bg-gray-100 flex items-center justify-center mb-4">
                  <Play className="h-8 w-8 text-gray-400" />
                </div>
                <h3 className="text-base font-semibold text-gray-900 mb-2">
                  No Test Cases Yet
                </h3>
                <p className="text-sm text-gray-500">
                  This test run doesn't have any test cases yet. They will
                  appear here as tests are executed.
                </p>
              </div>
            </CardContent>
          </Card>
        ) : filteredTestCount === 0 && hasActiveFilters ? (
          <Card className="border-dashed">
            <CardContent className="py-16">
              <div className="text-center max-w-sm mx-auto">
                <div className="mx-auto h-16 w-16 rounded-full bg-gray-100 flex items-center justify-center mb-4">
                  <Filter className="h-8 w-8 text-gray-400" />
                </div>
                <h3 className="text-base font-semibold text-gray-900 mb-2">
                  No Matching Tests
                </h3>
                <p className="text-sm text-gray-500 mb-4">
                  No test cases match your current filters. Try adjusting your
                  search criteria.
                </p>
                <button
                  onClick={clearFilters}
                  className="inline-flex items-center px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
                >
                  Clear All Filters
                </button>
              </div>
            </CardContent>
          </Card>
        ) : (
          <div className="transition-all duration-300">
            <TestSuiteRecord
              suite={filteredSuite}
              hiddenSuiteTypes={hiddenSuiteTypes}
            />
          </div>
        )}
      </div>
    </div>
  );
}
