import { useState, useCallback } from "react";
import { Link } from "react-router-dom";
import { ArrowLeft, Filter, RefreshCw } from "lucide-react";
import { TagTerritoryMap } from "@/components/TagTerritoryMap";
import type { TagTerritoryTest } from "@/types/tagTerritory";
import { generateMockTests, generateRealisticTestRun } from "@/utils/mockData";

export function TagTerritoryDemoPage() {
  const [tests, setTests] = useState<TagTerritoryTest[]>(() =>
    generateRealisticTestRun(),
  );
  const [maxVisibleTags, setMaxVisibleTags] = useState(30);
  const [renderRegions, setRenderRegions] = useState(true);

  const handleRegenerate = useCallback((count: number) => {
    setTests(generateMockTests(count));
  }, []);

  const handleGenerateRealistic = useCallback(() => {
    setTests(generateRealisticTestRun());
  }, []);

  const handleTestHover = useCallback((test: TagTerritoryTest | null) => {
    console.log("Hovered test:", test?.name);
  }, []);

  const handleTagClick = useCallback((tag: string) => {
    console.log("Clicked tag:", tag);
  }, []);

  return (
    <div className="container mx-auto px-4 py-8">
      {/* Header */}
      <div className="mb-6">
        <Link
          to="/suite_runs"
          className="inline-flex items-center text-blue-600 hover:text-blue-700 mb-4"
        >
          <ArrowLeft className="mr-2 h-4 w-4" />
          Back to Test Runs
        </Link>

        <div className="flex items-start justify-between">
          <div>
            <h1 className="text-3xl font-bold text-gray-900 mb-2">
              Tag Territory Map - Demo
            </h1>
            <p className="text-gray-600">
              Interactive visualization demo with mock data
            </p>
            <p className="text-sm text-gray-500 mt-1">
              Currently showing {tests.length} tests
            </p>
          </div>

          {/* Controls */}
          <div className="bg-white border border-gray-200 rounded-lg p-4 space-y-4 min-w-[250px]">
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

            <div className="border-t border-gray-200 pt-4">
              <label className="text-sm font-medium text-gray-700 mb-2 block">
                Generate Mock Data
              </label>
              <div className="space-y-2">
                <button
                  onClick={handleGenerateRealistic}
                  className="w-full flex items-center justify-center gap-2 px-3 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors text-sm"
                >
                  <RefreshCw className="h-4 w-4" />
                  Realistic Run (~130 tests)
                </button>
                <button
                  onClick={() => handleRegenerate(50)}
                  className="w-full flex items-center justify-center gap-2 px-3 py-2 bg-gray-600 text-white rounded-md hover:bg-gray-700 transition-colors text-sm"
                >
                  <RefreshCw className="h-4 w-4" />
                  Random (50 tests)
                </button>
                <button
                  onClick={() => handleRegenerate(200)}
                  className="w-full flex items-center justify-center gap-2 px-3 py-2 bg-gray-600 text-white rounded-md hover:bg-gray-700 transition-colors text-sm"
                >
                  <RefreshCw className="h-4 w-4" />
                  Random (200 tests)
                </button>
                <button
                  onClick={() => handleRegenerate(500)}
                  className="w-full flex items-center justify-center gap-2 px-3 py-2 bg-gray-600 text-white rounded-md hover:bg-gray-700 transition-colors text-sm"
                >
                  <RefreshCw className="h-4 w-4" />
                  Random (500 tests)
                </button>
              </div>
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
          <li>
            • Tag halos show the "territory" of each tag (overlaps indicate
            co-occurrence)
          </li>
        </ul>
      </div>

      {/* Map */}
      <div className="flex justify-center">
        <TagTerritoryMap
          tests={tests}
          maxVisibleTags={maxVisibleTags}
          renderRegions={renderRegions}
          onTestHover={handleTestHover}
          onTagClick={handleTagClick}
        />
      </div>

      {/* Technical Details */}
      <div className="mt-8 bg-gray-50 border border-gray-200 rounded-lg p-6">
        <h3 className="text-lg font-semibold text-gray-900 mb-3">
          Technical Details
        </h3>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm text-gray-700">
          <div>
            <h4 className="font-medium mb-2">Physics Simulation</h4>
            <ul className="space-y-1 text-gray-600">
              <li>• Runs in Web Worker (non-blocking)</li>
              <li>• Force-directed layout algorithm</li>
              <li>• Test-tag attraction forces</li>
              <li>• Tag-tag repulsion (based on similarity)</li>
              <li>• Collision detection and separation</li>
            </ul>
          </div>
          <div>
            <h4 className="font-medium mb-2">Performance</h4>
            <ul className="space-y-1 text-gray-600">
              <li>• Designed for up to 2,000 tests</li>
              <li>• Up to 150 tags (top N by impact)</li>
              <li>• Canvas rendering for efficiency</li>
              <li>• Simulation typically stabilizes in &lt;1s</li>
              <li>• Deterministic layout (seeded RNG)</li>
            </ul>
          </div>
        </div>
      </div>
    </div>
  );
}
