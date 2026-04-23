import {
  useEffect,
  useState,
  useCallback,
  useMemo,
  useRef,
  useLayoutEffect,
  startTransition,
} from "react";
import { useParams, Link, useNavigate } from "react-router-dom";
import { apiUrl, config } from "@/lib/config";
import { Card, CardContent } from "@/components/Card";
import { useRefresh } from "@/lib/refresh";
import { ArrowLeft, Map, Info } from "lucide-react";
import { cn } from "@/lib/utils";
import type { TestRun } from "@/types/testRun";
import type { TestSuite } from "@/types/testSuite";
import type { Test } from "@/types/testCase";
import TestBox, { type TestMapDensity } from "./TestBox";

function flattenTests(run: TestRun): Test[] {
  const flat: Test[] = [...(run.tests ?? [])];
  const visitSuite = (suite: TestSuite) => {
    flat.push(...(suite.tests ?? []));
    suite.suites?.forEach(visitSuite);
  };
  run.suites?.forEach(visitSuite);
  return flat;
}

function getRunRefreshFingerprint(run: TestRun): string {
  return [
    run.updatedAt,
    run.statistics?.lastUpdated ?? "",
    run.status ?? "",
    run.totalTests ?? "",
    run.tests?.length ?? 0,
    run.suites?.length ?? 0,
  ].join(":");
}

function calculateTileLayout(
  totalTests: number,
  width: number,
  height: number,
): {
  density: TestMapDensity;
  tileWidth: number;
  tileHeight: number;
  columns: number;
  rows: number;
  gap: number;
} {
  if (totalTests === 0 || width <= 0 || height <= 0) {
    return {
      density: "comfortable",
      tileWidth: 24,
      tileHeight: 24,
      columns: 1,
      rows: 1,
      gap: 6,
    };
  }

  const density: TestMapDensity =
    totalTests <= 16
      ? "comfortable"
      : totalTests <= 120
        ? "compact"
        : totalTests <= 1000
          ? "dense"
          : "ultra";

  const gap =
    totalTests > 1800 ? 1 : totalTests > 800 ? 2 : totalTests > 250 ? 3 : 4;
  const minTileSize =
    totalTests > 1800 ? 3 : totalTests > 800 ? 4 : totalTests > 250 ? 6 : 10;

  let best: {
    tileWidth: number;
    tileHeight: number;
    columns: number;
    rows: number;
    score: number;
  } | null = null;

  for (let columns = 1; columns <= totalTests; columns += 1) {
    const rows = Math.ceil(totalTests / columns);
    const tileWidth = Math.floor((width - gap * (columns - 1)) / columns);
    const tileHeight = Math.floor((height - gap * (rows - 1)) / rows);

    if (tileWidth < minTileSize || tileHeight < minTileSize) {
      continue;
    }

    const score =
      Math.min(tileWidth, tileHeight) * 1000 - Math.abs(tileWidth - tileHeight);
    if (!best || score > best.score) {
      best = { tileWidth, tileHeight, columns, rows, score };
    }
  }

  if (!best) {
    const columns = Math.max(
      1,
      Math.ceil(Math.sqrt((totalTests * width) / Math.max(height, 1))),
    );
    const rows = Math.max(1, Math.ceil(totalTests / columns));
    const tileWidth = Math.max(
      2,
      Math.floor((width - gap * (columns - 1)) / columns),
    );
    const tileHeight = Math.max(
      2,
      Math.floor((height - gap * (rows - 1)) / rows),
    );

    return { density, tileWidth, tileHeight, columns, rows, gap };
  }

  return {
    density,
    tileWidth: best.tileWidth,
    tileHeight: best.tileHeight,
    columns: best.columns,
    rows: best.rows,
    gap,
  };
}

const MAP_PROGRESSIVE_RENDER_THRESHOLD = 600;

function getInitialBatchSize(totalTests: number): number {
  if (totalTests > 1800) {
    return 140;
  }
  if (totalTests > 1200) {
    return 180;
  }
  if (totalTests > 800) {
    return 220;
  }
  return 320;
}

function getBatchSize(totalTests: number): number {
  if (totalTests > 1800) {
    return 220;
  }
  if (totalTests > 1200) {
    return 260;
  }
  if (totalTests > 800) {
    return 320;
  }
  return 420;
}

function getMapSkeletonBackgroundStyle(
  tileWidth: number,
  tileHeight: number,
  gap: number,
) {
  const stepX = Math.max(tileWidth + gap, 6);
  const stepY = Math.max(tileHeight + gap, 6);

  return {
    backgroundColor: "var(--stitch-surface-low)",
    backgroundImage: [
      "linear-gradient(to right, color-mix(in srgb, var(--stitch-surface-highest) 85%, transparent) 1px, transparent 1px)",
      "linear-gradient(to bottom, color-mix(in srgb, var(--stitch-surface-highest) 85%, transparent) 1px, transparent 1px)",
    ].join(","),
    backgroundSize: `${stepX}px ${stepY}px`,
    backgroundPosition: "0 0",
  } as const;
}

export function TestMapPage() {
  const pollIntervalMs = config.pollingIntervalMs;
  const { autoRefreshEnabled } = useRefresh();
  const { runId } = useParams<{ runId: string }>();
  const navigate = useNavigate();
  const [runDetail, setRunDetail] = useState<TestRun | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedTags, setSelectedTags] = useState<Set<string>>(new Set());
  const mapViewportRef = useRef<HTMLDivElement>(null);
  const [mapViewport, setMapViewport] = useState({ width: 0, height: 0 });
  const [renderedTileCount, setRenderedTileCount] = useState(0);
  const [isConstructingMap, setIsConstructingMap] = useState(false);

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
        setRunDetail((previous) => {
          if (
            previous &&
            silent &&
            getRunRefreshFingerprint(previous) ===
              getRunRefreshFingerprint(data)
          ) {
            return previous;
          }

          return data;
        });
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

  // Flatten all tests from run and nested suites
  const allTests = useMemo<Test[]>(
    () => (runDetail ? flattenTests(runDetail) : []),
    [runDetail],
  );

  const mapLayout = useMemo<{
    density: TestMapDensity;
    tileWidth: number;
    tileHeight: number;
    columns: number;
    rows: number;
    gap: number;
  }>(() => {
    return calculateTileLayout(
      allTests.length,
      mapViewport.width,
      mapViewport.height,
    );
  }, [allTests.length, mapViewport.height, mapViewport.width]);

  const renderedTests = useMemo(
    () => allTests.slice(0, renderedTileCount),
    [allTests, renderedTileCount],
  );
  const constructionSignature = useMemo(
    () =>
      [
        runId ?? "unknown-run",
        allTests.length,
        mapLayout.columns,
        mapLayout.rows,
        mapLayout.tileWidth,
        mapLayout.tileHeight,
        mapLayout.gap,
      ].join(":"),
    [
      allTests.length,
      mapLayout.columns,
      mapLayout.gap,
      mapLayout.rows,
      mapLayout.tileHeight,
      mapLayout.tileWidth,
      runId,
    ],
  );

  const isProgressiveRender =
    allTests.length >= MAP_PROGRESSIVE_RENDER_THRESHOLD;
  const mapConstructionProgress =
    allTests.length > 0
      ? Math.round((renderedTileCount / allTests.length) * 100)
      : 0;
  const showConstructionOverlay =
    loading || isConstructingMap || renderedTileCount < allTests.length;
  const mapSkeletonStyle = useMemo(
    () =>
      getMapSkeletonBackgroundStyle(
        mapLayout.tileWidth,
        mapLayout.tileHeight,
        mapLayout.gap,
      ),
    [mapLayout.gap, mapLayout.tileHeight, mapLayout.tileWidth],
  );

  useLayoutEffect(() => {
    const updateViewport = () => {
      if (!mapViewportRef.current) {
        return;
      }

      const rect = mapViewportRef.current.getBoundingClientRect();
      const footer = document.querySelector<HTMLElement>(
        '[role="contentinfo"], footer',
      );
      const footerHeight = footer
        ? Math.ceil(footer.getBoundingClientRect().height)
        : 0;
      const width = Math.max(
        0,
        Math.floor(rect.width || mapViewportRef.current.offsetWidth),
      );
      const height = Math.max(
        180,
        Math.floor(window.innerHeight - rect.top - footerHeight - 48),
      );

      if (width === 0) {
        return;
      }

      setMapViewport((previous) => {
        if (previous.width === width && previous.height === height) {
          return previous;
        }

        return {
          width,
          height,
        };
      });
    };

    let animationFrame = 0;
    let secondAnimationFrame = 0;

    const scheduleUpdate = () => {
      cancelAnimationFrame(animationFrame);
      cancelAnimationFrame(secondAnimationFrame);
      animationFrame = window.requestAnimationFrame(() => {
        updateViewport();
        secondAnimationFrame = window.requestAnimationFrame(updateViewport);
      });
    };

    scheduleUpdate();

    const observer = new ResizeObserver(() => {
      scheduleUpdate();
    });

    if (mapViewportRef.current) {
      observer.observe(mapViewportRef.current);
      if (mapViewportRef.current.parentElement) {
        observer.observe(mapViewportRef.current.parentElement);
      }
    }

    window.addEventListener("resize", scheduleUpdate);
    return () => {
      cancelAnimationFrame(animationFrame);
      cancelAnimationFrame(secondAnimationFrame);
      observer.disconnect();
      window.removeEventListener("resize", scheduleUpdate);
    };
  }, [allTests.length, loading]);

  useEffect(() => {
    const totalTests = allTests.length;

    if (loading || mapViewport.width === 0 || mapViewport.height === 0) {
      startTransition(() => {
        setRenderedTileCount(0);
        setIsConstructingMap(false);
      });
      return;
    }

    if (totalTests === 0) {
      startTransition(() => {
        setRenderedTileCount(0);
        setIsConstructingMap(false);
      });
      return;
    }

    if (!isProgressiveRender) {
      startTransition(() => {
        setRenderedTileCount(totalTests);
        setIsConstructingMap(false);
      });
      return;
    }

    let cancelled = false;
    let animationFrame = 0;
    let rendered = 0;
    const initialBatchSize = getInitialBatchSize(totalTests);
    const batchSize = getBatchSize(totalTests);

    startTransition(() => {
      setRenderedTileCount(0);
      setIsConstructingMap(true);
    });

    const renderNextBatch = () => {
      if (cancelled) {
        return;
      }

      rendered = Math.min(
        totalTests,
        rendered === 0 ? initialBatchSize : rendered + batchSize,
      );

      startTransition(() => {
        setRenderedTileCount(rendered);
        if (rendered >= totalTests) {
          setIsConstructingMap(false);
        }
      });

      if (rendered < totalTests) {
        animationFrame = window.requestAnimationFrame(renderNextBatch);
      }
    };

    animationFrame = window.requestAnimationFrame(renderNextBatch);

    return () => {
      cancelled = true;
      window.cancelAnimationFrame(animationFrame);
    };
  }, [constructionSignature, isProgressiveRender, loading]);

  // Extract all tags with occurrence counts
  const tagOccurrences = useMemo((): Record<string, number> => {
    const counts: Record<string, number> = {};
    allTests.forEach((test) => {
      test.tags?.forEach((tag) => {
        counts[tag] = (counts[tag] || 0) + 1;
      });
    });

    return counts;
  }, [allTests]);

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
    allTests.forEach((test) => {
      if (test.tags && test.tags.some((tag) => selectedTags.has(tag))) {
        ids.add(test.id);
      }
    });

    return ids;
  }, [selectedTags, allTests]);

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

  const handleTestClick = (testId: string) => {
    navigate(`/suite_runs/${runId}/tests/${testId}`);
  };

  if (loading) {
    return (
      <div className="space-y-6 animate-in fade-in duration-300">
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-4">
            <div className="h-10 w-10 bg-[var(--stitch-surface-low)] rounded-lg animate-pulse" />
            <div className="h-8 w-48 bg-[var(--stitch-surface-low)] rounded animate-pulse" />
          </div>
        </div>
        <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
          <div className="lg:col-span-3">
            <div className="relative h-96 overflow-hidden rounded-lg border border-(--stitch-outline) bg-(--stitch-surface-low)">
              <div
                className="absolute inset-0"
                style={{
                  backgroundImage:
                    "linear-gradient(to right, color-mix(in srgb, var(--stitch-surface-highest) 75%, transparent) 1px, transparent 1px), linear-gradient(to bottom, color-mix(in srgb, var(--stitch-surface-highest) 75%, transparent) 1px, transparent 1px)",
                  backgroundSize: "28px 28px",
                }}
              />
              <div className="absolute inset-0 bg-linear-to-r from-transparent via-white/35 to-transparent animate-shimmer" />
            </div>
          </div>
          <div className="lg:col-span-1">
            <div className="h-96 bg-[var(--stitch-surface-low)] rounded-lg animate-pulse" />
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
          className="inline-flex items-center gap-2 text-[var(--stitch-primary)] hover:text-[var(--stitch-primary)] transition-colors group"
        >
          <ArrowLeft className="h-5 w-5 group-hover:-translate-x-1 transition-transform" />
          <span className="font-medium">Back to Test Run</span>
        </Link>
        <Card className="border-[var(--status-failure-border)] bg-[var(--status-failure-soft)]/50">
          <CardContent className="py-12">
            <div className="text-center max-w-md mx-auto">
              <div className="mx-auto h-16 w-16 rounded-full bg-[var(--status-failure-soft)] flex items-center justify-center mb-4">
                <Info className="h-8 w-8 text-[var(--status-failure)]" />
              </div>
              <h3 className="text-lg font-semibold text-[var(--stitch-on-surface)] mb-2">
                Failed to Load Test Map
              </h3>
              <p className="text-sm text-[var(--stitch-on-surface-muted)] mb-6">
                {error || "Unknown error"}
              </p>
              <Link
                to={`/suite_runs/${runId}`}
                className="inline-flex items-center px-4 py-2 bg-[var(--stitch-primary-soft)] text-[var(--stitch-on-surface)] rounded-lg hover:bg-[var(--stitch-primary-soft)] transition-colors"
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
    <div className="space-y-6 animate-in fade-in duration-300">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Link
            to={`/suite_runs/${runId}`}
            className="inline-flex items-center justify-center h-10 w-10 rounded-lg bg-[var(--stitch-surface-card)] border border-[var(--stitch-outline)] text-[var(--stitch-on-surface)] hover:bg-[var(--stitch-surface-card)] hover:border-[var(--stitch-outline)] transition-all shadow-sm hover:shadow group"
            aria-label="Back to test run"
          >
            <ArrowLeft className="h-5 w-5 group-hover:-translate-x-0.5 transition-transform" />
          </Link>
          <div>
            <h1 className="text-2xl md:text-3xl font-bold text-[var(--stitch-on-surface)] tracking-tight flex items-center gap-2">
              <Map className="h-7 w-7 text-[var(--stitch-primary)]" />
              Test Map
            </h1>
            <p className="text-sm text-[var(--stitch-on-surface-muted)] mt-1">
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
                <div>
                  <h2 className="text-lg font-semibold text-[var(--stitch-on-surface)]">
                    Test Map
                  </h2>
                  <p className="mt-1 text-xs text-[var(--stitch-on-surface-muted)]">
                    {allTests.length} test{allTests.length === 1 ? "" : "s"} •{" "}
                    {mapLayout.columns} columns • {mapLayout.rows} rows •{" "}
                    {mapLayout.tileWidth}x{mapLayout.tileHeight}px tiles
                  </p>
                  <div className="mt-1 flex min-h-5 items-center">
                    {showConstructionOverlay && !loading ? (
                      <p className="text-xs font-medium text-[var(--stitch-primary)]">
                        {isConstructingMap
                          ? `Constructing map ${renderedTileCount}/${allTests.length} (${mapConstructionProgress}%)`
                          : "Finalizing map"}
                      </p>
                    ) : (
                      <span
                        className="invisible text-xs font-medium"
                        aria-hidden="true"
                      >
                        Map ready
                      </span>
                    )}
                  </div>
                </div>
                <div className="flex items-center gap-4 text-xs">
                  <div className="flex items-center gap-2">
                    <div className="h-3 w-3 rounded-full bg-[var(--status-success)]" />
                    <span>Passed</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <div className="h-3 w-3 rounded-full bg-[var(--stitch-tertiary)]" />
                    <span>Flaky</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <div className="h-3 w-3 rounded-full bg-[var(--status-failure)]" />
                    <span>Failed</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <div className="h-3 w-3 rounded-full bg-[var(--status-neutral)]" />
                    <span>Other</span>
                  </div>
                </div>
              </div>

              {allTests.length === 0 ? (
                <div className="text-center py-16 text-[var(--stitch-on-surface-muted)]">
                  <Map className="h-16 w-16 mx-auto mb-4 text-[var(--stitch-on-surface-subtle)]" />
                  <p>No tests available</p>
                </div>
              ) : (
                <div
                  ref={mapViewportRef}
                  className="relative w-full overflow-hidden rounded-lg"
                  style={{
                    minHeight: mapViewport.height
                      ? `${mapViewport.height}px`
                      : "360px",
                  }}
                >
                  {showConstructionOverlay && (
                    <div
                      className="pointer-events-none absolute inset-0 rounded-lg"
                      style={mapSkeletonStyle}
                    >
                      <div className="absolute inset-0 bg-linear-to-r from-transparent via-white/25 to-transparent animate-shimmer" />
                    </div>
                  )}
                  {mapViewport.width > 0 ? (
                    <div
                      className="relative z-10 grid w-full content-start overflow-hidden"
                      style={{
                        height: `${mapViewport.height}px`,
                        gridTemplateColumns: `repeat(${mapLayout.columns}, minmax(0, ${mapLayout.tileWidth}px))`,
                        gridAutoRows: `${mapLayout.tileHeight}px`,
                        gap: `${mapLayout.gap}px`,
                      }}
                    >
                      {renderedTests.map((test) => {
                        const isHighlighted = highlightedTestIds.has(test.id);
                        const isFaded = selectedTags.size > 0 && !isHighlighted;

                        return (
                          <TestBox
                            key={test.id}
                            test={test}
                            density={mapLayout.density}
                            width={mapLayout.tileWidth}
                            height={mapLayout.tileHeight}
                            isHighlighted={isHighlighted}
                            isFaded={isFaded}
                            onClick={() => handleTestClick(test.id)}
                          />
                        );
                      })}
                    </div>
                  ) : (
                    <div className="h-[360px] w-full rounded-lg bg-[var(--stitch-surface-low)]" />
                  )}
                </div>
              )}
            </CardContent>
          </Card>
        </div>

        {/* Tag Filter Sidebar */}
        <div className="lg:col-span-1">
          <Card className="sticky top-6">
            <CardContent className="p-6">
              <h2 className="text-lg font-semibold text-[var(--stitch-on-surface)] mb-4">
                Filter by Tags
              </h2>

              {sortedTags.length === 0 ? (
                <p className="text-sm text-[var(--stitch-on-surface-muted)] text-center py-8">
                  No tags available
                </p>
              ) : (
                <>
                  <div className="mb-4 text-xs text-[var(--stitch-on-surface-muted)]">
                    {selectedTags.size > 0
                      ? `${highlightedTestIds.size} test${highlightedTestIds.size !== 1 ? "s" : ""} highlighted`
                      : "Select tags to highlight tests"}
                  </div>
                  <div className="space-y-2 max-h-[500px] overflow-y-auto">
                    {sortedTags.map((tag) => {
                      const count = tagOccurrences[tag] || 0;
                      const isSelected = selectedTags.has(tag);

                      return (
                        <button
                          key={tag}
                          onClick={() => toggleTag(tag)}
                          className={cn(
                            "w-full text-left px-3 py-2 rounded-lg border transition-all text-sm",
                            isSelected
                              ? "bg-[var(--stitch-primary-soft)] border-[var(--status-running-border)] text-[var(--stitch-primary)] font-medium"
                              : "bg-[var(--stitch-surface-card)] border-[var(--stitch-outline)] text-[var(--stitch-on-surface)] hover:bg-[var(--stitch-surface-card)] hover:border-[var(--stitch-outline)]",
                          )}
                        >
                          <div className="flex items-center justify-between">
                            <span className="truncate flex-1">{tag}</span>
                            <span
                              className={cn(
                                "ml-2 px-2 py-0.5 rounded text-xs font-medium",
                                isSelected
                                  ? "bg-[var(--stitch-primary-soft)] text-[var(--stitch-primary)]"
                                  : "bg-[var(--stitch-surface-low)] text-[var(--stitch-on-surface-muted)]",
                              )}
                            >
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
                      className="w-full mt-4 px-3 py-2 text-sm text-[var(--stitch-primary)] hover:text-[var(--stitch-primary)] font-medium transition-colors"
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
