import { useEffect, useRef, useState, useCallback, useMemo } from "react";
import type {
  TagTerritoryMapProps,
  SimulationState,
  SimulationConfig,
  TestNode,
  TagNode,
  TagInfo,
} from "@/types/tagTerritory";
import { selectTopTags, getStatusColor } from "@/utils/tagSelection";
import { formatDuration } from "@/utils/timeUtils";
import { cn } from "@/lib/utils";

const DEFAULT_MAX_VISIBLE_TAGS = 30;
const DEFAULT_WIDTH = 1200;
const DEFAULT_HEIGHT = 800;

export function TagTerritoryMap({
  tests,
  maxVisibleTags = DEFAULT_MAX_VISIBLE_TAGS,
  width = DEFAULT_WIDTH,
  height = DEFAULT_HEIGHT,
  renderRegions = true,
  onTestHover,
  onTagClick,
}: TagTerritoryMapProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const workerRef = useRef<Worker | null>(null);

  const [state, setState] = useState<SimulationState | null>(null);
  const [isSimulating, setIsSimulating] = useState(false);
  const [selectedTags, setSelectedTags] = useState<TagInfo[]>([]);
  const [hoveredTest, setHoveredTest] = useState<TestNode | null>(null);
  const [hoveredTag, setHoveredTag] = useState<TagNode | null>(null);
  const [focusedTag, setFocusedTag] = useState<string | null>(null);

  // Pan and zoom state
  const [transform, setTransform] = useState({
    x: 0,
    y: 0,
    scale: 1,
  });
  const [isPanning, setIsPanning] = useState(false);
  const [panStart, setPanStart] = useState({ x: 0, y: 0 });

  // Helper function to check if a point is within a tag label
  const isPointInTagLabel = useCallback(
    (tag: TagNode, x: number, y: number, ctx: CanvasRenderingContext2D): boolean => {
      const isFocused = focusedTag === tag.name;
      ctx.font = isFocused ? "bold 14px sans-serif" : "12px sans-serif";
      const metrics = ctx.measureText(tag.name);
      const padding = 6;
      const labelWidth = metrics.width + padding * 2;
      const labelHeight = 20;

      const dx = Math.abs(tag.x - x);
      const dy = Math.abs(tag.y - y);

      return dx < labelWidth / 2 && dy < labelHeight / 2;
    },
    [focusedTag],
  );

  // Initialize worker
  useEffect(() => {
    const worker = new Worker(
      new URL("../workers/tagTerritoryWorker.ts", import.meta.url),
      { type: "module" },
    );

    worker.onmessage = (e) => {
      const { type, state: newState, error } = e.data;

      if (type === "error") {
        console.error("Worker error:", error);
        setIsSimulating(false);
        return;
      }

      if (newState) {
        setState(newState);
      }

      if (type === "complete") {
        setIsSimulating(false);
      }
    };

    workerRef.current = worker;

    return () => {
      worker.postMessage({ type: "stop" });
      worker.terminate();
    };
  }, []);

  // Select top tags based on tests and max visible tags
  const selectedTagsData = useMemo(() => {
    if (tests.length === 0) return [];
    return selectTopTags(tests, maxVisibleTags);
  }, [tests, maxVisibleTags]);

  // Update selectedTags when selectedTagsData changes
  useEffect(() => {
    setSelectedTags(selectedTagsData);
  }, [selectedTagsData]);

  // Start simulation when tests change
  useEffect(() => {
    if (!workerRef.current || tests.length === 0 || selectedTagsData.length === 0) return;

    // Filter tests to only include those with selected tags
    const tagNames = new Set(selectedTagsData.map((t) => t.name));
    const filteredTests = tests.filter((test) =>
      test.tags.some((tag) => tagNames.has(tag)),
    );

    if (filteredTests.length === 0) {
      console.warn("No tests match selected tags");
      return;
    }

    // Start simulation
    const config: SimulationConfig = {
      width,
      height,
      seed: 42, // Deterministic seed
      maxIterations: 500,
      stabilityThreshold: 0.1,
      forces: {
        testTagAttraction: 0.5,
        tagTagRepulsion: 5000,
        centering: 0.001,
        separation: 0.5,
      },
    };

    setIsSimulating(true);
    workerRef.current.postMessage({
      type: "init",
      config,
      tests: filteredTests,
      tags: selectedTagsData,
    });

    workerRef.current.postMessage({ type: "step" });
  }, [tests, maxVisibleTags, width, height, selectedTagsData]);

  // Render canvas
  useEffect(() => {
    if (!canvasRef.current || !state) return;

    const canvas = canvasRef.current;
    const ctx = canvas.getContext("2d");
    if (!ctx) return;

    // Set canvas size
    canvas.width = width;
    canvas.height = height;

    // Clear canvas
    ctx.fillStyle = "#f9fafb"; // gray-50
    ctx.fillRect(0, 0, width, height);

    // Apply transform
    ctx.save();
    ctx.translate(transform.x, transform.y);
    ctx.scale(transform.scale, transform.scale);

    // Render tag halos (soft gradient circles)
    if (renderRegions) {
      state.tags.forEach((tag) => {
        const gradient = ctx.createRadialGradient(
          tag.x,
          tag.y,
          0,
          tag.x,
          tag.y,
          tag.radius * 2,
        );

        // Parse color and create gradient
        const color = tag.info.color;
        gradient.addColorStop(0, color.replace("hsl", "hsla").replace(")", ", 0.3)"));
        gradient.addColorStop(0.5, color.replace("hsl", "hsla").replace(")", ", 0.15)"));
        gradient.addColorStop(1, color.replace("hsl", "hsla").replace(")", ", 0)"));

        ctx.fillStyle = gradient;
        ctx.fillRect(
          tag.x - tag.radius * 2,
          tag.y - tag.radius * 2,
          tag.radius * 4,
          tag.radius * 4,
        );
      });
    }

    // Render test nodes
    state.tests.forEach((test) => {
      const isHovered = hoveredTest?.id === test.id;
      const isRelatedToHoveredTag =
        hoveredTag && hoveredTag.testIds.has(test.id);
      const isRelatedToFocusedTag =
        focusedTag && test.test.tags.includes(focusedTag);
      const isDimmed =
        (hoveredTag && !isRelatedToHoveredTag) ||
        (focusedTag && !isRelatedToFocusedTag);

      ctx.globalAlpha = isDimmed ? 0.2 : 1.0;

      // Draw test node
      ctx.fillStyle = getStatusColor(test.test.status);
      ctx.beginPath();
      ctx.arc(test.x, test.y, test.radius, 0, Math.PI * 2);
      ctx.fill();

      // Highlight if hovered
      if (isHovered) {
        ctx.strokeStyle = "#000";
        ctx.lineWidth = 2;
        ctx.beginPath();
        ctx.arc(test.x, test.y, test.radius + 2, 0, Math.PI * 2);
        ctx.stroke();
      }

      ctx.globalAlpha = 1.0;
    });

    // Render tag labels
    state.tags.forEach((tag) => {
      const isHovered = hoveredTag?.name === tag.name;
      const isFocused = focusedTag === tag.name;

      ctx.globalAlpha = hoveredTag && !isHovered ? 0.4 : 1.0;

      // Draw tag label background
      ctx.font = isFocused ? "bold 14px sans-serif" : "12px sans-serif";
      const metrics = ctx.measureText(tag.name);
      const padding = 6;
      const bgWidth = metrics.width + padding * 2;
      const bgHeight = 20;

      ctx.fillStyle = isHovered || isFocused ? tag.info.color : "#ffffff";
      ctx.strokeStyle = tag.info.color;
      ctx.lineWidth = 2;
      ctx.beginPath();
      ctx.roundRect(
        tag.x - bgWidth / 2,
        tag.y - bgHeight / 2,
        bgWidth,
        bgHeight,
        4,
      );
      ctx.fill();
      ctx.stroke();

      // Draw tag label text
      ctx.fillStyle = isHovered || isFocused ? "#ffffff" : "#1f2937";
      ctx.textAlign = "center";
      ctx.textBaseline = "middle";
      ctx.fillText(tag.name, tag.x, tag.y);

      ctx.globalAlpha = 1.0;
    });

    ctx.restore();
  }, [
    state,
    width,
    height,
    renderRegions,
    hoveredTest,
    hoveredTag,
    focusedTag,
    transform,
  ]);

  // Mouse event handlers
  const getMousePosition = useCallback(
    (e: React.MouseEvent<HTMLCanvasElement>) => {
      const canvas = canvasRef.current;
      if (!canvas) return { x: 0, y: 0 };

      const rect = canvas.getBoundingClientRect();
      const x = (e.clientX - rect.left - transform.x) / transform.scale;
      const y = (e.clientY - rect.top - transform.y) / transform.scale;
      return { x, y };
    },
    [transform],
  );

  const handleMouseMove = useCallback(
    (e: React.MouseEvent<HTMLCanvasElement>) => {
      if (isPanning) {
        const dx = e.clientX - panStart.x;
        const dy = e.clientY - panStart.y;
        setTransform((t) => ({
          ...t,
          x: t.x + dx,
          y: t.y + dy,
        }));
        setPanStart({ x: e.clientX, y: e.clientY });
        return;
      }

      if (!state) return;

      const { x, y } = getMousePosition(e);

      // Check for hovered test
      let foundTest: TestNode | null = null;
      for (const test of state.tests) {
        const dx = test.x - x;
        const dy = test.y - y;
        const distance = Math.sqrt(dx * dx + dy * dy);
        if (distance < test.radius) {
          foundTest = test;
          break;
        }
      }

      if (foundTest !== hoveredTest) {
        setHoveredTest(foundTest);
        onTestHover?.(foundTest?.test || null);
      }

      // Check for hovered tag
      const canvas = canvasRef.current;
      if (!canvas) return;
      const ctx = canvas.getContext("2d");
      if (!ctx) return;

      let foundTag: TagNode | null = null;
      for (const tag of state.tags) {
        if (isPointInTagLabel(tag, x, y, ctx)) {
          foundTag = tag;
          break;
        }
      }

      setHoveredTag(foundTag);
    },
    [state, hoveredTest, isPanning, panStart, getMousePosition, onTestHover, isPointInTagLabel],
  );

  const handleMouseDown = useCallback(
    (e: React.MouseEvent<HTMLCanvasElement>) => {
      if (e.button === 0) {
        // Left click
        if (!state) return;

        const { x, y } = getMousePosition(e);
        const canvas = canvasRef.current;
        if (!canvas) return;
        const ctx = canvas.getContext("2d");
        if (!ctx) return;

        // Check for tag click
        for (const tag of state.tags) {
          if (isPointInTagLabel(tag, x, y, ctx)) {
            // Toggle focused tag
            if (focusedTag === tag.name) {
              setFocusedTag(null);
            } else {
              setFocusedTag(tag.name);
              onTagClick?.(tag.name);
            }
            return;
          }
        }

        // Start panning
        setIsPanning(true);
        setPanStart({ x: e.clientX, y: e.clientY });
      }
    },
    [state, focusedTag, getMousePosition, onTagClick, isPointInTagLabel],
  );

  const handleMouseUp = useCallback(() => {
    setIsPanning(false);
  }, []);

  const handleWheel = useCallback((e: React.WheelEvent<HTMLCanvasElement>) => {
    e.preventDefault();

    const delta = e.deltaY > 0 ? 0.9 : 1.1;
    setTransform((t) => ({
      ...t,
      scale: Math.max(0.1, Math.min(5, t.scale * delta)),
    }));
  }, []);

  const handleResetView = useCallback(() => {
    setTransform({ x: 0, y: 0, scale: 1 });
    setFocusedTag(null);
  }, []);

  return (
    <div ref={containerRef} className="relative">
      {/* Canvas */}
      <canvas
        ref={canvasRef}
        width={width}
        height={height}
        className={cn(
          "border border-gray-300 rounded-lg bg-gray-50",
          isPanning ? "cursor-grabbing" : "cursor-grab",
        )}
        onMouseMove={handleMouseMove}
        onMouseDown={handleMouseDown}
        onMouseUp={handleMouseUp}
        onMouseLeave={() => {
          setHoveredTest(null);
          setHoveredTag(null);
          setIsPanning(false);
        }}
        onWheel={handleWheel}
      />

      {/* Status overlay */}
      <div className="absolute top-4 left-4 bg-white/90 backdrop-blur-sm px-4 py-2 rounded-lg shadow-sm border border-gray-200">
        <div className="text-sm text-gray-600">
          {isSimulating ? (
            <span className="flex items-center gap-2">
              <span className="inline-block w-2 h-2 bg-blue-500 rounded-full animate-pulse" />
              Simulating... (iteration {state?.iteration || 0})
            </span>
          ) : (
            <span className="flex items-center gap-2">
              <span className="inline-block w-2 h-2 bg-green-500 rounded-full" />
              Stable ({state?.iteration || 0} iterations)
            </span>
          )}
        </div>
        <div className="text-xs text-gray-500 mt-1">
          {tests.length} tests, {selectedTags.length} tags
        </div>
      </div>

      {/* Controls */}
      <div className="absolute top-4 right-4 flex flex-col gap-2">
        <button
          onClick={handleResetView}
          className="bg-white/90 backdrop-blur-sm px-3 py-2 rounded-lg shadow-sm border border-gray-200 text-sm hover:bg-white transition-colors"
        >
          Reset View
        </button>
      </div>

      {/* Legend */}
      <div className="absolute bottom-4 left-4 bg-white/90 backdrop-blur-sm px-4 py-3 rounded-lg shadow-sm border border-gray-200 max-w-md max-h-64 overflow-y-auto">
        <h3 className="text-sm font-semibold text-gray-900 mb-2">
          Top Tags ({selectedTags.length})
        </h3>
        <div className="space-y-1">
          {selectedTags.slice(0, 10).map((tag) => (
            <div
              key={tag.name}
              className="flex items-center gap-2 text-xs cursor-pointer hover:bg-gray-100 px-2 py-1 rounded"
              onClick={() => {
                if (focusedTag === tag.name) {
                  setFocusedTag(null);
                } else {
                  setFocusedTag(tag.name);
                  onTagClick?.(tag.name);
                }
              }}
            >
              <div
                className="w-3 h-3 rounded-full flex-shrink-0"
                style={{ backgroundColor: tag.color }}
              />
              <span
                className={cn(
                  "flex-1 truncate",
                  focusedTag === tag.name && "font-semibold",
                )}
              >
                {tag.name}
              </span>
              <span className="text-gray-500">{tag.count}</span>
            </div>
          ))}
          {selectedTags.length > 10 && (
            <div className="text-xs text-gray-500 text-center pt-1">
              +{selectedTags.length - 10} more tags
            </div>
          )}
        </div>
      </div>

      {/* Hover tooltip */}
      {hoveredTest && (
        <div className="absolute top-1/2 left-1/2 transform -translate-x-1/2 -translate-y-1/2 bg-white px-4 py-3 rounded-lg shadow-lg border border-gray-200 max-w-md pointer-events-none">
          <h4 className="font-semibold text-sm mb-1 truncate">
            {hoveredTest.test.name}
          </h4>
          <div className="text-xs text-gray-600 space-y-1">
            <div>Status: {hoveredTest.test.status}</div>
            <div>Duration: {formatDuration(hoveredTest.test.durationMs)}</div>
            {hoveredTest.test.retries > 0 && (
              <div>Retries: {hoveredTest.test.retries}</div>
            )}
            <div className="flex flex-wrap gap-1 mt-2">
              {hoveredTest.test.tags.slice(0, 5).map((tag) => (
                <span
                  key={tag}
                  className="inline-block px-2 py-0.5 rounded-full text-xs bg-blue-100 text-blue-800"
                >
                  {tag}
                </span>
              ))}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
