import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { apiUrl } from "@/lib/config";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/Card";
import { Tag, AlertCircle, TrendingUp, ArrowRight } from "lucide-react";

interface MarkerInfo {
  marker: string;
  count: number;
}

interface MarkersResponse {
  markers: MarkerInfo[];
  count: number;
}

export function MarkerBrowsePage() {
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
      const data: MarkersResponse = await response.json();
      setMarkers(data.markers || []);
      setError(null);
    } catch (err) {
      console.error("Error fetching markers:", err);
      setError(
        err instanceof Error ? err.message : "Failed to fetch markers"
      );
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-gray-600">Loading markers...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="space-y-4">
        <Card>
          <CardContent className="py-12">
            <div className="flex flex-col items-center justify-center space-y-4">
              <AlertCircle className="h-16 w-16 text-red-500" />
              <div className="text-red-600 text-center">
                <p className="font-semibold">Error: {error}</p>
                <p className="text-sm mt-1">
                  Unable to fetch markers. Please try again later.
                </p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-2">
          <Tag className="h-6 w-6 text-blue-600" />
          <h1 className="text-3xl font-bold text-gray-900">Test Markers</h1>
        </div>
      </div>

      <p className="text-gray-600">
        Browse test runs organized by marker values. Markers are used to group
        and track test runs across different environments, releases, or builds.
      </p>

      {/* Summary Card */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center space-x-2">
            <TrendingUp className="h-5 w-5 text-blue-600" />
            <span>Available Markers</span>
          </CardTitle>
        </CardHeader>
        <CardContent>
          {markers.length === 0 ? (
            <div className="text-center py-8 text-gray-500">
              <p className="mb-2">No markers found.</p>
              <p className="text-sm">
                Markers are created when test runs include a MARKER field in
                their metadata.
              </p>
            </div>
          ) : (
            <div className="grid grid-cols-1 gap-4">
              {markers.map((markerInfo) => (
                <Link
                  key={markerInfo.marker}
                  to={`/marker/${encodeURIComponent(markerInfo.marker)}/stats`}
                  className="group"
                >
                  <Card className="hover:shadow-lg transition-shadow duration-200 border-2 border-transparent group-hover:border-blue-500">
                    <CardContent className="py-4">
                      <div className="flex items-center justify-between">
                        <div className="flex-1">
                          <div className="flex items-center space-x-3">
                            <Tag className="h-5 w-5 text-blue-600 group-hover:text-blue-700" />
                            <div>
                              <h3 className="font-mono text-lg font-semibold text-gray-900 group-hover:text-blue-600 break-all">
                                {markerInfo.marker}
                              </h3>
                              <p className="text-sm text-gray-600 mt-1">
                                {markerInfo.count} test run
                                {markerInfo.count !== 1 ? "s" : ""}
                              </p>
                            </div>
                          </div>
                        </div>
                        <div className="flex items-center space-x-2">
                          <span className="text-sm text-blue-600 group-hover:text-blue-700 font-medium">
                            View Statistics
                          </span>
                          <ArrowRight className="h-5 w-5 text-blue-600 group-hover:text-blue-700 group-hover:translate-x-1 transition-transform" />
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                </Link>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
