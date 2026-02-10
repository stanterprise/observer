import { useEffect, useState, useCallback, useMemo, useRef } from "react";
import { useParams, Link, useNavigate } from "react-router-dom";
import { apiUrl, config } from "@/lib/config";
import { Card, CardContent } from "@/components/Card";
import { ArrowLeft, Map, Info } from "lucide-react";
import { cn } from "@/lib/utils";
import type { TestRun } from "@/types/testRun";
import type { Test } from "@/types/testCase";
import type { TestStatus } from "@/types/common";

// Helper to determine if a test is flaky (passed with retries)
const isFlaky = (test: Test): boolean => {
  return test.status === "PASSED" && (test.attempts?.length ?? 0) > 1;
};

// Helper to get color for test status including flaky
const getTestStatusColor = (test: Test): string => {
  if (isFlaky(test)) {
    return "bg-amber-400 border-amber-500"; // Flaky: amber/yellow
  }

  const statusColors: Record<TestStatus, string> = {
    PASSED: "bg-green-500 border-green-600",
    FAILED: "bg-red-500 border-red-600",
    SKIPPED: "bg-gray-400 border-gray-500",
    RUNNING: "bg-blue-500 border-blue-600",
    PENDING: "bg-yellow-400 border-yellow-500",
    BROKEN: "bg-orange-500 border-orange-600",
    TIMEDOUT: "bg-purple-500 border-purple-600",
    INTERRUPTED: "bg-pink-500 border-pink-600",
    NOT_RUN: "bg-gray-300 border-gray-400",
    UNKNOWN: "bg-gray-400 border-gray-500",
  };

  return statusColors[test.status] || statusColors.UNKNOWN;
};

// Get human-readable status label
const getStatusLabel = (test: Test): string => {
  if (isFlaky(test)) {
    return "Flaky";
  }
  return test.status.charAt(0) + test.status.slice(1).toLowerCase().replace("_", " ");
};

interface TestBoxProps {
  test: Test;
  isHighlighted: boolean;
  isFaded: boolean; // New prop for fading non-highlighted tests
  onClick: () => void;
  size: number; // Dynamic size in pixels
}

function TestBox({ test, isHighlighted, isFaded, onClick, size }: TestBoxProps) {
  const [showTooltip, setShowTooltip] = useState(false);
  const colorClass = getTestStatusColor(test);
  
  // Scale border width based on size (1px for small, 2px for large)
  const borderWidth = size < 12 ? 1 : 2;

  return (
    <div className="relative">
      <div
        className={cn(
          "rounded cursor-pointer transition-all duration-200",
          colorClass,
          isHighlighted && "ring-2 ring-blue-300 scale-110",
          !isHighlighted && "hover:scale-105 hover:shadow-md"
        )}
        style={{ 
          width: `${size}px`, 
          height: `${size}px`,
          borderWidth: `${borderWidth}px`,
          opacity: isFaded ? 0.25 : 1, // Apply fading when isFaded is true
        }}
        onClick={onClick}
        onMouseEnter={() => setShowTooltip(true)}
        onMouseLeave={() => setShowTooltip(false)}
        role="button"
        tabIndex={0}
        aria-label={`Test: ${test.title}`}
      />
      {showTooltip && (
        <div className="absolute z-50 bottom-full left-1/2 -translate-x-1/2 mb-2 w-64 p-3 bg-gray-900 text-white text-xs rounded-lg shadow-xl pointer-events-none">
          <div className="font-semibold mb-1 truncate">{test.title}</div>
          <div className="space-y-1 text-gray-300">
            <div>Status: <span className="font-medium text-white">{getStatusLabel(test)}</span></div>
            {test.duration && (
              <div>Duration: {(test.duration / 1_000_000).toFixed(2)}ms</div>
            )}
            {test.tags && test.tags.length > 0 && (
              <div>Tags: {test.tags.join(", ")}</div>
            )}
            {isFlaky(test) && (
              <div className="text-amber-300">
                ⚠️ Passed with {test.attempts!.length} attempts
              </div>
            )}
          </div>
          <div className="absolute top-full left-1/2 -translate-x-1/2 -mt-1">
            <div className="border-4 border-transparent border-t-gray-900" />
          </div>
        </div>
      )}
    </div>
  );
}

export function TestMapPage() {
  const pollIntervalMs = config.pollingIntervalMs;
  const { runId } = useParams<{ runId: string }>();
  const navigate = useNavigate();
  const [runDetail, setRunDetail] = useState<TestRun | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedTags, setSelectedTags] = useState<Set<string>>(new Set());
  const containerRef = useRef<HTMLDivElement>(null);
  const [containerDimensions, setContainerDimensions] = useState({ width: 0, height: 0 });

  const fetchRunDetail = useCallback(
    async (id: string, options?: { silent?: boolean }) => {
      const silent = options?.silent ?? false;
      try {
        if (!silent) {
          setLoading(true);
        }
        const response = await fetch(apiUrl(`/runs/${id}`));
        if (!response.ok) {
          throw new Error(`Failed to fetch run details: ${response.statusText}`);
        }
        const data = await response.json();
        setRunDetail(data);
        setError(null);
      } catch (err) {
        console.error("Error fetching run details:", err);
        setError(err instanceof Error ? err.message : "Failed to fetch run details");
      } finally {
        if (!silent) {
          setLoading(false);
        }
      }
    },
    []
  );

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

  // Measure container dimensions
  useEffect(() => {
    const updateDimensions = () => {
      if (containerRef.current) {
        const rect = containerRef.current.getBoundingClientRect();
        setContainerDimensions({ width: rect.width, height: rect.height });
      }
    };

    updateDimensions();
    window.addEventListener('resize', updateDimensions);
    return () => window.removeEventListener('resize', updateDimensions);
  }, [runDetail]);

  // Calculate optimal test box size based on total test count and available space
  const testBoxSize = useMemo(() => {
    if (!runDetail?.tests || runDetail.tests.length === 0) return 32;
    
    const totalTests = runDetail.tests.length;
    const { width, height } = containerDimensions;
    
    // Reserve space for padding and gaps
    const availableWidth = width - 48; // Account for card padding
    const availableHeight = height - 120; // Account for header and padding
    
    if (availableWidth <= 0 || availableHeight <= 0) return 32;
    
    // Enforce 8:6 aspect ratio (1.333...)
    const TARGET_ASPECT_RATIO = 8 / 6;
    const cols = Math.ceil(Math.sqrt(totalTests * TARGET_ASPECT_RATIO));
    const rows = Math.ceil(totalTests / cols);
    
    // Calculate size that fits all tests
    const gap = 2; // Gap between boxes in pixels
    const sizeByWidth = (availableWidth - (cols - 1) * gap) / cols;
    const sizeByHeight = (availableHeight - (rows - 1) * gap) / rows;
    
    const calculatedSize = Math.floor(Math.min(sizeByWidth, sizeByHeight));
    
    // Clamp between min and max sizes
    const MIN_SIZE = 4;
    const MAX_SIZE = 32;
    return Math.max(MIN_SIZE, Math.min(MAX_SIZE, calculatedSize));
  }, [runDetail?.tests, containerDimensions]);

  // Extract all tags with occurrence counts
  const tagOccurrences = useMemo((): Record<string, number> => {
    if (!runDetail?.tests) return {};
    
    const counts: Record<string, number> = {};
    runDetail.tests.forEach(test => {
      test.tags?.forEach(tag => {
        counts[tag] = (counts[tag] || 0) + 1;
      });
    });
    
    return counts;
  }, [runDetail?.tests]);

  // Sort tags by occurrence (descending)
  const sortedTags = useMemo((): string[] => {
    return Object.entries(tagOccurrences)
      .sort((a: [string, number], b: [string, number]) => b[1] - a[1])
      .map(([tag]: [string, number]) => tag);
  }, [tagOccurrences]);

  // Get highlighted test IDs based on selected tags
  const highlightedTestIds = useMemo(() => {
    if (selectedTags.size === 0) return new Set<string>();
    
    const ids = new Set<string>();
    runDetail?.tests?.forEach(test => {
      if (test.tags && test.tags.some(tag => selectedTags.has(tag))) {
        ids.add(test.id);
      }
    });
    
    return ids;
  }, [selectedTags, runDetail?.tests]);

  const toggleTag = (tag: string) => {
    setSelectedTags(prev => {
      const newSet = new Set(prev);
      if (newSet.has(tag)) {
        newSet.delete(tag);
      } else {
        newSet.add(tag);
      }
      return newSet;
    });
  };

  const handleTestClick = (testId: string) => {
    navigate(`/suite_runs/${runId}/tests/${testId}`);
  };

  if (loading) {
    return (
      <div className="space-y-6 animate-in fade-in duration-300">
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-4">
            <div className="h-10 w-10 bg-gray-200 rounded-lg animate-pulse" />
            <div className="h-8 w-48 bg-gray-200 rounded animate-pulse" />
          </div>
        </div>
        <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
          <div className="lg:col-span-3">
            <div className="h-96 bg-gray-200 rounded-lg animate-pulse" />
          </div>
          <div className="lg:col-span-1">
            <div className="h-96 bg-gray-200 rounded-lg animate-pulse" />
          </div>
        </div>
      </div>
    );
  }

  if (error || !runDetail) {
    return (
      <div className="space-y-6 animate-in fade-in duration-300">
        <Link
          to={`/suite_runs/${runId}`}
          className="inline-flex items-center gap-2 text-blue-600 hover:text-blue-700 transition-colors group"
        >
          <ArrowLeft className="h-5 w-5 group-hover:-translate-x-1 transition-transform" />
          <span className="font-medium">Back to Test Run</span>
        </Link>
        <Card className="border-red-200 bg-red-50/50">
          <CardContent className="py-12">
            <div className="text-center max-w-md mx-auto">
              <div className="mx-auto h-16 w-16 rounded-full bg-red-100 flex items-center justify-center mb-4">
                <Info className="h-8 w-8 text-red-600" />
              </div>
              <h3 className="text-lg font-semibold text-gray-900 mb-2">
                Failed to Load Test Map
              </h3>
              <p className="text-sm text-gray-600 mb-6">{error || "Unknown error"}</p>
              <Link
                to={`/suite_runs/${runId}`}
                className="inline-flex items-center px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
              >
                Back to Test Run
              </Link>
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-6 pb-8 animate-in fade-in duration-300">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Link
            to={`/suite_runs/${runId}`}
            className="inline-flex items-center justify-center h-10 w-10 rounded-lg bg-white border border-gray-200 text-gray-700 hover:bg-gray-50 hover:border-gray-300 transition-all shadow-sm hover:shadow group"
            aria-label="Back to test run"
          >
            <ArrowLeft className="h-5 w-5 group-hover:-translate-x-0.5 transition-transform" />
          </Link>
          <div>
            <h1 className="text-2xl md:text-3xl font-bold text-gray-900 tracking-tight flex items-center gap-2">
              <Map className="h-7 w-7 text-blue-600" />
              Test Map
            </h1>
            <p className="text-sm text-gray-500 mt-1">
              {runDetail.name || runDetail.id}
            </p>
          </div>
        </div>
      </div>

      {/* Main Layout */}
      <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
        {/* Test Map Canvas */}
        <div className="lg:col-span-3">
          <Card>
            <CardContent className="p-6">
              <div className="mb-4 flex items-center justify-between">
                <h2 className="text-lg font-semibold text-gray-900">
                  Test Map
                </h2>
                <div className="flex items-center gap-4 text-xs">
                  <div className="flex items-center gap-2">
                    <div className="w-3 h-3 rounded bg-green-500 border border-green-600" />
                    <span>Passed</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <div className="w-3 h-3 rounded bg-amber-400 border border-amber-500" />
                    <span>Flaky</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <div className="w-3 h-3 rounded bg-red-500 border border-red-600" />
                    <span>Failed</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <div className="w-3 h-3 rounded bg-gray-400 border border-gray-500" />
                    <span>Other</span>
                  </div>
                </div>
              </div>
              
              {!runDetail.tests || runDetail.tests.length === 0 ? (
                <div className="text-center py-16 text-gray-500">
                  <Map className="h-16 w-16 mx-auto mb-4 text-gray-300" />
                  <p>No tests available</p>
                </div>
              ) : (
                <div 
                  ref={containerRef}
                  className="min-h-[500px]"
                  style={{ height: 'calc(100vh - 320px)' }}
                >
                  <div 
                    className="flex flex-wrap content-start"
                    style={{ gap: '2px' }}
                  >
                    {runDetail.tests.map(test => {
                      const isHighlighted = highlightedTestIds.has(test.id);
                      const isFaded = selectedTags.size > 0 && !isHighlighted;
                      
                      return (
                        <TestBox
                          key={test.id}
                          test={test}
                          size={testBoxSize}
                          isHighlighted={isHighlighted}
                          isFaded={isFaded}
                          onClick={() => handleTestClick(test.id)}
                        />
                      );
                    })}
                  </div>
                </div>
              )}
            </CardContent>
          </Card>
        </div>

        {/* Tag Filter Sidebar */}
        <div className="lg:col-span-1">
          <Card className="sticky top-6">
            <CardContent className="p-6">
              <h2 className="text-lg font-semibold text-gray-900 mb-4">
                Filter by Tags
              </h2>
              
              {sortedTags.length === 0 ? (
                <p className="text-sm text-gray-500 text-center py-8">
                  No tags available
                </p>
              ) : (
                <>
                  <div className="mb-4 text-xs text-gray-500">
                    {selectedTags.size > 0 
                      ? `${highlightedTestIds.size} test${highlightedTestIds.size !== 1 ? 's' : ''} highlighted`
                      : "Select tags to highlight tests"}
                  </div>
                  <div className="space-y-2 max-h-[500px] overflow-y-auto">
                    {sortedTags.map(tag => {
                      const count = tagOccurrences[tag] || 0;
                      const isSelected = selectedTags.has(tag);
                      
                      return (
                        <button
                          key={tag}
                          onClick={() => toggleTag(tag)}
                          className={cn(
                            "w-full text-left px-3 py-2 rounded-lg border transition-all text-sm",
                            isSelected
                              ? "bg-blue-100 border-blue-300 text-blue-900 font-medium"
                              : "bg-white border-gray-200 text-gray-700 hover:bg-gray-50 hover:border-gray-300"
                          )}
                        >
                          <div className="flex items-center justify-between">
                            <span className="truncate flex-1">{tag}</span>
                            <span className={cn(
                              "ml-2 px-2 py-0.5 rounded text-xs font-medium",
                              isSelected 
                                ? "bg-blue-200 text-blue-800" 
                                : "bg-gray-100 text-gray-600"
                            )}>
                              {count}
                            </span>
                          </div>
                        </button>
                      );
                    })}
                  </div>
                  
                  {selectedTags.size > 0 && (
                    <button
                      onClick={() => setSelectedTags(new Set())}
                      className="w-full mt-4 px-3 py-2 text-sm text-blue-600 hover:text-blue-700 font-medium transition-colors"
                    >
                      Clear Selection
                    </button>
                  )}
                </>
              )}
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
