import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { apiUrl } from "@/lib/config";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/Card";
import { Tag, AlertTriangle, Loader } from "lucide-react";

interface MarkerInfo {
  marker: string;
  count: number;
}

export default function MarkersPage() {
  const [markers, setMarkers] = useState<MarkerInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetchMarkers();
  }, []);

  const fetchMarkers = async () => {
    try {
      setLoading(true);
      const response = await fetch(apiUrl("/markers"));
      if (!response.ok) {
        throw new Error(`Failed to fetch markers: ${response.statusText}`);
      }
      const data = await response.json();
      const markersList = (data.markers || []) as MarkerInfo[];
      setMarkers(markersList);
      setError(null);
    } catch (err) {
      console.error("Error fetching markers:", err);
      setError(err instanceof Error ? err.message : "Failed to fetch markers");
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Loader className="h-8 w-8 animate-spin text-blue-600" />
        <span className="ml-3 text-gray-600">Loading markers...</span>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center h-64 space-y-4">
        <AlertTriangle className="h-12 w-12 text-red-500" />
        <div className="text-red-600">Error: {error}</div>
        <button
          onClick={fetchMarkers}
          className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors"
        >
          Retry
        </button>
      </div>
    );
  }

  const hasMarkers = markers.length > 0;

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-gray-900">Test Markers</h1>
          <p className="text-gray-600 mt-1">
            Browse all available test markers and view their run statistics
          </p>
        </div>
        <button
          onClick={fetchMarkers}
          className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors"
        >
          Refresh
        </button>
      </div>

      {!hasMarkers ? (
        /* Empty State */
        <Card>
          <CardContent className="py-12">
            <div className="text-center">
              <Tag className="mx-auto h-16 w-16 text-gray-400" />
              <h3 className="mt-4 text-lg font-medium text-gray-900">
                No Markers Found
              </h3>
              <p className="mt-2 text-gray-500 max-w-md mx-auto">
                No test runs with MARKER metadata have been recorded yet.
                Markers are used to tag and categorize test runs for easier
                filtering and analysis.
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
        <Card>
          <CardHeader>
            <CardTitle>Available Markers ({markers.length})</CardTitle>
          </CardHeader>
          <CardContent className="p-0">
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-gray-200">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Marker
                    </th>
                    <th className="px-6 py-3 text-center text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Test Runs
                    </th>
                    <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Actions
                    </th>
                  </tr>
                </thead>
                <tbody className="bg-white divide-y divide-gray-200">
                  {markers.map((markerInfo) => (
                    <tr
                      key={markerInfo.marker}
                      className="hover:bg-gray-50 transition-colors"
                    >
                      <td className="px-6 py-4 whitespace-nowrap">
                        <div className="flex items-center">
                          <Tag className="h-5 w-5 text-blue-600 mr-2" />
                          <span className="text-gray-900 font-medium">
                            {markerInfo.marker}
                          </span>
                        </div>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-center">
                        <span className="inline-flex items-center px-3 py-1 rounded-full text-sm font-semibold bg-blue-100 text-blue-800">
                          {markerInfo.count} {markerInfo.count === 1 ? "run" : "runs"}
                        </span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-right text-sm">
                        <Link
                          to={`/suite_runs?marker=${encodeURIComponent(markerInfo.marker)}`}
                          className="inline-flex items-center px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors"
                        >
                          View Stats
                        </Link>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
