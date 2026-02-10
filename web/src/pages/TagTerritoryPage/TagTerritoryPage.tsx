import { useEffect, useState, useCallback } from "react";
import { useParams, Link } from "react-router-dom";
import { apiUrl } from "@/lib/config";
import { ArrowLeft, Filter } from "lucide-react";
import { TagTerritoryMap } from "@/components/TagTerritoryMap";
import type { TagTerritoryTest } from "@/types/tagTerritory";
import type { TestRun } from "@/types/testRun";

export function TagTerritoryPage() {
  const { runId } = useParams<{ runId: string }>();
  const [runDetail, setRunDetail] = useState<TestRun | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [maxVisibleTags, setMaxVisibleTags] = useState(30);
  const [renderRegions, setRenderRegions] = useState(true);

  const fetchRunDetail = useCallback(async (id: string) => {
    try {
      setLoading(true);
      const response = await fetch(apiUrl(`/runs/${id}`));
      if (!response.ok) {
        throw new Error(`Failed to fetch run details: ${response.statusText}`);
      }
      const data = await response.json();
      setRunDetail(data);
      setError(null);
    } catch (err) {
      console.error("Error fetching run details:", err);
      setError(
        err instanceof Error ? err.message : "Failed to fetch run details",
      );
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (runId) {
      fetchRunDetail(runId);
    }
  }, [runId, fetchRunDetail]);

  // Convert tests to TagTerritoryTest format
  const territoryTests: TagTerritoryTest[] = (runDetail?.tests || []).map(
    (test) => ({
      id: test.id,
      name: test.title,
      tags: test.tags || [],
      status: test.status,
      durationMs: test.duration ? test.duration / 1_000_000 : 0, // Convert nanoseconds to milliseconds
      retries: test.retryCount || 0,
    }),
  );

  const handleTestHover = useCallback((test: TagTerritoryTest | null) => {
    // Could update a tooltip or side panel here
    console.log("Hovered test:", test?.name);
  }, []);

  const handleTagClick = useCallback((tag: string) => {
    console.log("Clicked tag:", tag);
  }, []);

  if (loading) {
    return (
      <div className="container mx-auto px-4 py-8">
        <div className="animate-pulse">
          <div className="h-8 bg-gray-200 rounded w-1/3 mb-4"></div>
          <div className="h-96 bg-gray-200 rounded"></div>
        </div>
      </div>
    );
  }

  if (error || !runDetail) {
    return (
      <div className="container mx-auto px-4 py-8">
        <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded">
          {error || "Failed to load test run"}
        </div>
      </div>
    );
  }

  if (territoryTests.length === 0) {
    return (
      <div className="container mx-auto px-4 py-8">
        <Link
          to={`/suite_runs/${runId}`}
          className="inline-flex items-center text-blue-600 hover:text-blue-700 mb-4"
        >
          <ArrowLeft className="mr-2 h-4 w-4" />
          Back to Run Details
        </Link>
        <div className="bg-yellow-50 border border-yellow-200 text-yellow-700 px-4 py-3 rounded">
          No tests found in this run
        </div>
      </div>
    );
  }

  return (
    <div className="container mx-auto px-4 py-8">
      {/* Header */}
      <div className="mb-6">
        <Link
          to={`/suite_runs/${runId}`}
          className="inline-flex items-center text-blue-600 hover:text-blue-700 mb-4"
        >
          <ArrowLeft className="mr-2 h-4 w-4" />
          Back to Run Details
        </Link>

        <div className="flex items-start justify-between">
          <div>
            <h1 className="text-3xl font-bold text-gray-900 mb-2">
              Tag Territory Map
            </h1>
            <p className="text-gray-600">
              {runDetail.name || `Run ${runId}`}
            </p>
            <p className="text-sm text-gray-500 mt-1">
              Visualizing {territoryTests.length} tests across tag territories
            </p>
          </div>

          {/* Controls */}
          <div className="bg-white border border-gray-200 rounded-lg p-4 space-y-4">
            <div>
              <label className="flex items-center gap-2 text-sm font-medium text-gray-700 mb-2">
                <Filter className="h-4 w-4" />
                Max Visible Tags
              </label>
              <select
                value={maxVisibleTags}
                onChange={(e) => setMaxVisibleTags(Number(e.target.value))}
                className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
              >
                <option value={10}>10 tags</option>
                <option value={30}>30 tags</option>
                <option value={60}>60 tags</option>
                <option value={100}>100 tags</option>
              </select>
            </div>

            <div>
              <label className="flex items-center gap-2 text-sm">
                <input
                  type="checkbox"
                  checked={renderRegions}
                  onChange={(e) => setRenderRegions(e.target.checked)}
                  className="rounded border-gray-300"
                />
                <span className="text-gray-700">Render tag regions</span>
              </label>
            </div>
          </div>
        </div>
      </div>

      {/* Info Panel */}
      <div className="bg-blue-50 border border-blue-200 rounded-lg p-4 mb-6">
        <h3 className="font-semibold text-blue-900 mb-2">How to use:</h3>
        <ul className="text-sm text-blue-800 space-y-1">
          <li>
            • <strong>Hover</strong> over test dots to see details
          </li>
          <li>
            • <strong>Click</strong> tag labels to focus on specific tags
          </li>
          <li>
            • <strong>Drag</strong> to pan, <strong>scroll</strong> to zoom
          </li>
          <li>
            • Test colors represent status (green=passed, red=failed, etc.)
          </li>
          <li>
            • Larger dots indicate tests with more tags or longer duration
          </li>
        </ul>
      </div>

      {/* Map */}
      <div className="flex justify-center">
        <TagTerritoryMap
          tests={territoryTests}
          maxVisibleTags={maxVisibleTags}
          renderRegions={renderRegions}
          onTestHover={handleTestHover}
          onTagClick={handleTagClick}
        />
      </div>
    </div>
  );
}
