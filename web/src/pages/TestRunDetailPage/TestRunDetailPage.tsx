import { useEffect, useState, useCallback, useMemo } from "react";
import { useParams, Link } from "react-router-dom";
import { apiUrl, config } from "@/lib/config";
import { Card, CardContent } from "@/components/Card";
import { useRefresh } from "@/lib/refresh";
import {
  ArrowLeft,
  Play,
  Eye,
  EyeOff,
  Search,
  X,
  Filter,
  Map as MapIcon,
  FileText,
} from "lucide-react";
import { cn } from "@/lib/utils";

import { SuiteTitleCard } from "./SuiteTitleCard";
import type { TestStatus } from "@/types/common";
import { assembleSuiteHierarchy } from "../TestSuiteRunsPage/utils";
import { getRunStatus } from "../TestSuiteRunsPage/utils";
import type { Test } from "@/types/testCase";
import type { TestSuite } from "@/types/testSuite";
import TestSuiteRecord from "./TestSuiteRecord";
import type { TestRun } from "@/types/testRun";

function isFlakyTest(test: Test): boolean {
  return test.status === "PASSED" && (test.attempts?.length ?? 0) > 1;
}

function flattenSuiteTests(suites: TestSuite[]): Test[] {
  const flattened: Test[] = [];

  const visit = (suite: TestSuite) => {
    flattened.push(...(suite.tests ?? []));
    suite.suites?.forEach(visit);
  };

  suites.forEach(visit);
  return flattened;
}

function flattenSuites(suites: TestSuite[]): TestSuite[] {
  const flattened: TestSuite[] = [];

  const visit = (suite: TestSuite) => {
    flattened.push({
      ...suite,
      tests: [],
      suites: [],
    });
    suite.suites?.forEach(visit);
  };

  suites.forEach(visit);
  return flattened;
}

function dedupeTests(tests: Test[]): Test[] {
  const byKey = new Map<string, Test>();

  for (const test of tests) {
    byKey.set(`${test.id}:${test.suiteId ?? ""}`, test);
  }

  return Array.from(byKey.values());
}

function computeStatistics(tests: Test[]) {
  return {
    total: tests.length,
    passed: tests.filter((test) => test.status === "PASSED").length,
    failed: tests.filter((test) => test.status === "FAILED").length,
    skipped: tests.filter((test) => test.status === "SKIPPED").length,
    running: tests.filter((test) => test.status === "RUNNING").length,
    pending: tests.filter((test) => test.status === "PENDING").length,
    notRun: tests.filter((test) => test.status === "NOT_RUN").length,
    broken: tests.filter((test) => test.status === "BROKEN").length,
    timedout: tests.filter((test) => test.status === "TIMEDOUT").length,
    interrupted: tests.filter((test) => test.status === "INTERRUPTED").length,
    unknown: tests.filter((test) => test.status === "UNKNOWN").length,
    expected: tests.filter(
      (test) => test.status === "PASSED" && (test.attempts?.length ?? 0) === 1,
    ).length,
    flaky: tests.filter(
      (test) => test.status === "PASSED" && (test.attempts?.length ?? 0) > 1,
    ).length,
  };
}

function normalizeRunDetail(data: TestRun): TestRun {
  const nestedSuites = data.suites ?? [];
  const suites = flattenSuites(nestedSuites);
  const tests = dedupeTests([
    ...(data.tests ?? []),
    ...flattenSuiteTests(nestedSuites),
  ]);

  return {
    ...data,
    suites,
    tests,
    statistics: computeStatistics(tests),
  };
}

export function TestRunDetailPage() {
  const pollIntervalMs = config.pollingIntervalMs;
  const { autoRefreshEnabled } = useRefresh();
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
        const data = normalizeRunDetail(await response.json());
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
    if (!runId || !autoRefreshEnabled) return;
    const intervalId = window.setInterval(() => {
      fetchRunDetail(runId, { silent: true });
    }, pollIntervalMs);

    return () => {
      window.clearInterval(intervalId);
    };
  }, [runId, autoRefreshEnabled, fetchRunDetail, pollIntervalMs]);

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

    if (!runDetail.suites || runDetail.suites.length === 0) {
      return {
        id: runDetail.id,
        name: runDetail.name || runDetail.id,
        type: "ROOT",
        runId: runDetail.id,
        tests: runDetail.tests || [],
        suites: [],
      } as TestSuite;
    }

    return assembleSuiteHierarchy(runDetail.suites, runDetail.tests || []);
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
        const isFlaky = isFlakyTest(test);
        const matchesPassed =
          selectedStatuses.has("PASSED") && test.status === "PASSED";
        const matchesFlaky = selectedStatuses.has("FLAKY") && isFlaky;
        const matchesStandardStatus =
          test.status !== "PASSED" && selectedStatuses.has(test.status);

        if (!matchesPassed && !matchesFlaky && !matchesStandardStatus) {
          return false;
        }
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
            <div className="h-10 w-10 bg-(--stitch-surface-highest) rounded-lg animate-pulse" />
            <div className="h-8 w-48 bg-(--stitch-surface-highest) rounded animate-pulse" />
          </div>
        </div>
        <div className="bg-(--stitch-surface-card) rounded-lg shadow-md border border-(--stitch-outline) overflow-hidden">
          <div className="h-8 bg-(--stitch-surface-highest) animate-pulse" />
          <div className="p-6 space-y-4">
            <div className="h-6 bg-(--stitch-surface-highest) rounded w-3/4 animate-pulse" />
            <div className="grid grid-cols-2 md:grid-cols-4 gap-6">
              {[1, 2, 3, 4].map((i) => (
                <div
                  key={i}
                  className="h-32 bg-(--stitch-surface-low) rounded-lg animate-pulse"
                />
              ))}
            </div>
          </div>
        </div>
        <div className="space-y-3">
          {[1, 2, 3].map((i) => (
            <div
              key={i}
              className="h-24 bg-(--stitch-surface-low) rounded-lg animate-pulse"
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
          className="group inline-flex items-center gap-2 rounded-md px-2 py-1 text-(--stitch-primary) transition-colors hover:bg-(--stitch-primary-soft) focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-(--stitch-primary) focus-visible:ring-offset-2 focus-visible:ring-offset-(--stitch-background)"
        >
          <ArrowLeft className="h-5 w-5 group-hover:-translate-x-1 transition-transform" />
          <span className="font-medium">Back to Test Runs</span>
        </Link>
        <Card
          className="border-(--status-failure-border)"
          style={{ backgroundColor: "var(--status-failure-soft)" }}
        >
          <CardContent className="py-12">
            <div className="text-center max-w-md mx-auto">
              <div
                className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full"
                style={{ backgroundColor: "var(--status-failure-soft)" }}
              >
                <svg
                  className="h-8 w-8 text-(--status-failure)"
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
              <h3 className="text-lg font-semibold text-(--stitch-on-surface) mb-2">
                {error ? "Failed to Load Test Run" : "Test Run Not Found"}
              </h3>
              <p className="text-sm text-(--stitch-on-surface-muted) mb-6">
                {error ||
                  "The test run you're looking for doesn't exist or has been deleted."}
              </p>
              <Link
                to="/suite_runs"
                className="inline-flex items-center rounded-lg px-4 py-2 text-white transition-opacity hover:opacity-90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-(--stitch-primary) focus-visible:ring-offset-2 focus-visible:ring-offset-(--stitch-background)"
                style={{
                  background:
                    "linear-gradient(135deg, var(--stitch-primary), var(--stitch-primary-end))",
                }}
              >
                View All Test Runs
              </Link>
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  const overallStatus: TestStatus = getRunStatus(runDetail);
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
            className="group inline-flex h-10 w-10 items-center justify-center rounded-lg border border-(--stitch-outline) bg-(--stitch-surface-card) text-(--stitch-on-surface-muted) transition-colors hover:bg-(--stitch-surface-low) focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-(--stitch-primary) focus-visible:ring-offset-2 focus-visible:ring-offset-(--stitch-background)"
            aria-label="Back to test runs"
          >
            <ArrowLeft className="h-5 w-5 group-hover:-translate-x-0.5 transition-transform" />
          </Link>
          <div>
            <h1 className="text-2xl md:text-3xl font-bold text-(--stitch-on-surface) tracking-tight">
              Test Suite Run
            </h1>
            <p className="text-sm text-(--stitch-on-surface-subtle) mt-1">
              {runDetail.name || runDetail.id}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Link
            to={`/suite_runs/${runId}/raw-messages`}
            className="inline-flex items-center gap-2 rounded-lg border border-(--stitch-outline) bg-(--stitch-surface-card) px-4 py-2 text-(--stitch-on-surface-muted) transition-colors hover:bg-(--stitch-surface-low) focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-(--stitch-primary) focus-visible:ring-offset-2 focus-visible:ring-offset-(--stitch-background)"
          >
            <FileText className="h-5 w-5" />
            <span className="font-medium">Raw Messages</span>
          </Link>
          <Link
            to={`/suite_runs/${runId}/map`}
            className="inline-flex items-center gap-2 rounded-lg px-4 py-2 text-white transition-opacity hover:opacity-90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-(--stitch-primary) focus-visible:ring-offset-2 focus-visible:ring-offset-(--stitch-background)"
            style={{
              background:
                "linear-gradient(135deg, var(--stitch-primary), var(--stitch-primary-end))",
            }}
          >
            <MapIcon className="h-5 w-5" />
            <span className="font-medium">View Test Map</span>
          </Link>
        </div>
      </div>

      {/* Run Summary Card with improved spacing */}
      <div className="transition-all duration-300">
        <SuiteTitleCard runDetail={runDetail} overallStatus={overallStatus} />
      </div>

      {/* Test Cases List with enhanced design */}
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-xl font-semibold text-(--stitch-on-surface)">
            Test Cases
            {hasActiveFilters && (
              <span className="ml-2 text-sm font-normal text-(--stitch-on-surface-subtle)">
                ({filteredTestCount} of {runDetail.statistics?.total || 0})
              </span>
            )}
          </h2>
          {availableSuiteTypes.length > 0 && (
            <div className="flex items-center gap-2">
              <span className="text-sm text-(--stitch-on-surface-muted) font-medium">
                Suites:
              </span>
              <div className="flex gap-2">
                {availableSuiteTypes.map((type) => {
                  const isHidden = hiddenSuiteTypes.has(type);
                  return (
                    <button
                      key={type}
                      onClick={() => toggleSuiteType(type)}
                      className={cn(
                        "inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm font-medium transition-all border focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-(--stitch-primary)",
                        isHidden
                          ? "bg-(--stitch-surface-low) text-(--stitch-on-surface-subtle) border-(--stitch-outline) hover:bg-(--stitch-surface-highest)"
                          : "bg-(--stitch-primary-soft) text-(--stitch-primary) border-(--status-running-border) hover:bg-(--stitch-primary-soft)",
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
        <Card className="border-(--stitch-outline) bg-(--stitch-surface-low)/50">
          <CardContent className="py-4">
            <div className="space-y-4">
              {/* Search Bar */}
              <div className="relative">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-5 w-5 text-(--stitch-on-surface-subtle)" />
                <input
                  type="text"
                  placeholder="Search test cases by name or ID..."
                  value={searchText}
                  onChange={(e) => setSearchText(e.target.value)}
                  className="w-full rounded-lg border border-(--stitch-outline) bg-(--stitch-surface-card) py-2.5 pl-10 pr-10 text-(--stitch-on-surface) placeholder:text-(--stitch-on-surface-subtle) transition-colors focus:outline-none focus:ring-2 focus:ring-(--stitch-primary) focus:ring-offset-2 focus:ring-offset-(--stitch-background)"
                />
                {searchText && (
                  <button
                    onClick={() => setSearchText("")}
                    className="absolute right-3 top-1/2 -translate-y-1/2 text-(--stitch-on-surface-subtle) hover:text-(--stitch-on-surface-muted) transition-colors"
                    aria-label="Clear search"
                  >
                    <X className="h-5 w-5" />
                  </button>
                )}
              </div>

              {/* Status Filters */}
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <label className="text-sm font-medium text-(--stitch-on-surface-muted) flex items-center gap-2">
                    <Filter className="h-4 w-4" />
                    Filter by Status
                  </label>
                  {hasActiveFilters && (
                    <button
                      onClick={clearFilters}
                      className="rounded-md px-2 py-1 text-sm font-medium text-(--stitch-primary) transition-colors hover:bg-(--stitch-primary-soft) focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-(--stitch-primary)"
                    >
                      Clear All Filters
                    </button>
                  )}
                </div>
                <div className="flex flex-wrap gap-2">
                  {(
                    [
                      "PASSED",
                      "FLAKY",
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
                        ? "bg-(--status-success-soft) border-(--status-success-border) text-(--status-success)"
                        : "bg-(--stitch-surface-card) border-(--status-success-border) text-(--status-success) hover:bg-(--status-success-soft)",
                      FLAKY: isSelected
                        ? "bg-(--status-warning-soft) border-(--status-warning-border) text-(--status-warning)"
                        : "bg-(--stitch-surface-card) border-(--status-warning-border) text-(--status-warning) hover:bg-(--status-warning-soft)",
                      FAILED: isSelected
                        ? "bg-(--status-failure-soft) border-(--status-failure-border) text-(--status-failure)"
                        : "bg-(--stitch-surface-card) border-(--status-failure-border) text-(--status-failure) hover:bg-(--status-failure-soft)",
                      RUNNING: isSelected
                        ? "bg-(--status-running-soft) border-(--status-running-border) text-(--status-running)"
                        : "bg-(--stitch-surface-card) border-(--status-running-border) text-(--status-running) hover:bg-(--status-running-soft)",
                      SKIPPED: isSelected
                        ? "bg-(--status-neutral-soft) border-(--status-neutral-border) text-(--status-neutral)"
                        : "bg-(--stitch-surface-card) border-(--stitch-outline) text-(--stitch-on-surface-muted) hover:bg-(--stitch-surface-low)",
                      BROKEN: isSelected
                        ? "bg-(--status-broken-soft) border-(--status-broken-border) text-(--status-broken)"
                        : "bg-(--stitch-surface-card) border-(--status-broken-border) text-(--status-broken) hover:bg-(--status-broken-soft)",
                      TIMEDOUT: isSelected
                        ? "bg-(--status-timedout-soft) border-(--status-timedout-border) text-(--status-timedout)"
                        : "bg-(--stitch-surface-card) border-(--status-timedout-border) text-(--status-timedout) hover:bg-(--status-timedout-soft)",
                      INTERRUPTED: isSelected
                        ? "bg-(--status-interrupted-soft) border-(--status-interrupted-border) text-(--status-interrupted)"
                        : "bg-(--stitch-surface-card) border-(--status-interrupted-border) text-(--status-interrupted) hover:bg-(--status-interrupted-soft)",
                      PENDING: isSelected
                        ? "bg-(--status-warning-soft) border-(--status-warning-border) text-(--status-warning)"
                        : "bg-(--stitch-surface-card) border-(--status-warning-border) text-(--status-warning) hover:bg-(--status-warning-soft)",
                      UNKNOWN: isSelected
                        ? "bg-(--status-neutral-soft) border-(--status-neutral-border) text-(--status-neutral)"
                        : "bg-(--stitch-surface-card) border-(--status-neutral-border) text-(--status-neutral) hover:bg-(--status-neutral-soft)",
                      NOT_RUN: isSelected
                        ? "bg-(--status-neutral-soft) border-(--status-neutral-border) text-(--status-neutral)"
                        : "bg-(--stitch-surface-card) border-(--status-neutral-border) text-(--status-neutral) hover:bg-(--status-neutral-soft)",
                    };
                    return (
                      <button
                        key={status}
                        onClick={() => toggleStatus(status)}
                        className={cn(
                          "rounded-md border-2 px-3 py-1.5 text-sm font-medium transition-all focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-(--stitch-primary)",
                          statusColors[status] ||
                            "bg-(--stitch-surface-card) border-(--stitch-outline) text-(--stitch-on-surface-muted)",
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
                  <label className="text-sm font-medium text-(--stitch-on-surface-muted)">
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
                            "rounded-md border px-3 py-1.5 text-sm font-medium transition-all focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-(--stitch-primary)",
                            isSelected
                              ? "bg-(--stitch-primary-soft) border-(--status-running-border) text-(--stitch-primary)"
                              : "bg-(--stitch-surface-card) border-(--stitch-outline) text-(--stitch-on-surface-muted) hover:bg-(--stitch-surface-low)",
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
                <div className="mx-auto h-16 w-16 rounded-full bg-(--stitch-surface-low) flex items-center justify-center mb-4">
                  <Play className="h-8 w-8 text-(--stitch-on-surface-subtle)" />
                </div>
                <h3 className="text-base font-semibold text-(--stitch-on-surface) mb-2">
                  No Test Cases Yet
                </h3>
                <p className="text-sm text-(--stitch-on-surface-subtle)">
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
                <div className="mx-auto h-16 w-16 rounded-full bg-(--stitch-surface-low) flex items-center justify-center mb-4">
                  <Filter className="h-8 w-8 text-(--stitch-on-surface-subtle)" />
                </div>
                <h3 className="text-base font-semibold text-(--stitch-on-surface) mb-2">
                  No Matching Tests
                </h3>
                <p className="text-sm text-(--stitch-on-surface-subtle) mb-4">
                  No test cases match your current filters. Try adjusting your
                  search criteria.
                </p>
                <button
                  onClick={clearFilters}
                  className="inline-flex items-center rounded-lg px-4 py-2 text-white transition-opacity hover:opacity-90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-(--stitch-primary) focus-visible:ring-offset-2 focus-visible:ring-offset-(--stitch-background)"
                  style={{
                    background:
                      "linear-gradient(135deg, var(--stitch-primary), var(--stitch-primary-end))",
                  }}
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
