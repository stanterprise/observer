// Environment configuration for API and WebSocket endpoints
// These can be overridden via environment variables at build time or runtime

export const config = {
  // API endpoint - defaults to /api for same-origin requests (proxied by Nginx)
  // In distributed mode, Nginx will proxy /api/* to the API service
  // In AIO mode, the API service runs on port 8080 within the same container
  apiUrl: import.meta.env.VITE_API_URL || "/api",

  // WebSocket endpoint - defaults to /ws for same-origin requests (proxied by Nginx)
  // In distributed mode, Nginx will proxy /ws to the API service WebSocket endpoint
  // In AIO mode, the API service WebSocket runs on port 8080 within the same container
  wsUrl:
    import.meta.env.VITE_WS_URL ||
    (window.location.protocol === "https:"
      ? `wss://${window.location.host}/ws`
      : `ws://${window.location.host}/ws`),
};

// Helper to construct API URLs
export function apiUrl(path: string): string {
  const base = config.apiUrl.replace(/\/$/, "");
  const p = path.startsWith("/") ? path : `/${path}`;
  return `${base}${p}`;
}

// WebSocket filter options
export interface WebSocketFilters {
  runId?: string;
  testId?: string;
  suiteId?: string;
  eventTypes?: string[]; // e.g., ['test.begin', 'test.end', 'run.start']
}

// Helper to get WebSocket URL with optional filters
export function wsUrl(filters?: WebSocketFilters): string {
  if (!filters) {
    return config.wsUrl;
  }

  const params = new URLSearchParams();

  if (filters.runId) {
    params.append("runId", filters.runId);
  }
  if (filters.testId) {
    params.append("testId", filters.testId);
  }
  if (filters.suiteId) {
    params.append("suiteId", filters.suiteId);
  }
  if (filters.eventTypes && filters.eventTypes.length > 0) {
    params.append("eventTypes", filters.eventTypes.join(","));
  }

  const queryString = params.toString();
  return queryString ? `${config.wsUrl}?${queryString}` : config.wsUrl;
}
