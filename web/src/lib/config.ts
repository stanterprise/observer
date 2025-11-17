// Environment configuration for API and WebSocket endpoints
// These can be overridden via environment variables at build time or runtime

export const config = {
  // API endpoint - defaults to /api for same-origin requests (proxied by Nginx)
  // In distributed mode, Nginx will proxy /api/* to the API service
  // In AIO mode, the API service runs on port 8080 within the same container
  apiUrl: import.meta.env.VITE_API_URL || '/api',

  // WebSocket endpoint - defaults to /ws for same-origin requests (proxied by Nginx)
  // In distributed mode, Nginx will proxy /ws to the API service WebSocket endpoint
  // In AIO mode, the API service WebSocket runs on port 8080 within the same container
  wsUrl: import.meta.env.VITE_WS_URL || (
    window.location.protocol === 'https:' 
      ? `wss://${window.location.host}/ws`
      : `ws://${window.location.host}/ws`
  ),
}

// Helper to construct API URLs
export function apiUrl(path: string): string {
  const base = config.apiUrl.replace(/\/$/, '')
  const p = path.startsWith('/') ? path : `/${path}`
  return `${base}${p}`
}

// Helper to get WebSocket URL
export function wsUrl(): string {
  return config.wsUrl
}
