# Local Web Development Guide

This guide explains how to run the web UI locally for development while running the backend services in Docker Compose.

## Quick Start

1. **Start backend services** (database, NATS, ingestion, processor, API):

   ```bash
   docker compose --profile web-dev up -d
   ```

2. **Install web dependencies** (first time only):

   ```bash
   cd web
   npm install
   ```

3. **Run web dev server**:

   ```bash
   npm run dev
   ```

4. **Access the app** at http://localhost:3000

## Architecture

When running in local dev mode:

```
Browser → Vite Dev Server (:3000) ─┬→ /api/* → Proxy → Docker API Service (:8080)
                                   └→ /ws → Proxy → Docker API WebSocket (:8080/ws)
```

- **Vite dev server** runs on port 3000 (hot reload, fast refresh)
- **API service** runs in Docker on port 8080 (exposed to host)
- **Vite proxy** forwards `/api` and `/ws` requests to Docker API
- **CORS** is enabled on API service for cross-origin requests

## Configuration

### Vite Proxy (vite.config.ts)

The Vite dev server is configured to proxy API and WebSocket requests:

```typescript
server: {
  port: 3000,
  proxy: {
    '/api': {
      target: 'http://localhost:8080',  // Docker API service
      changeOrigin: true,
    },
    '/ws': {
      target: 'ws://localhost:8080',    // WebSocket endpoint
      ws: true,
    },
  },
}
```

### Docker Compose (web-dev profile)

The `web-dev` profile starts backend services with:

- **API service** exposed on port 8080
- **CORS enabled** (`CORS_ALLOWED_ORIGINS=*`)
- All other services (ingestion, processor, db, NATS) running

### Environment Variables

Create `.env.development.local` if you need to override defaults:

```bash
cp .env.development.local.example .env.development.local
```

Available variables:

- `VITE_API_URL` - API endpoint (default: `/api`)
- `VITE_WS_URL` - WebSocket endpoint (default: auto-detected)

## Troubleshooting

### CORS Errors

If you see CORS errors in the browser console:

1. Check that API service is running: `docker compose ps api`
2. Verify CORS is enabled: `docker compose logs api | grep CORS`
3. Ensure API service has `CORS_ALLOWED_ORIGINS=*` in docker-compose.yml

### WebSocket Connection Failed

If WebSocket fails to connect:

1. Check API service logs: `docker compose logs -f api`
2. Verify NATS is running: `docker compose ps nats`
3. Check WebSocket URL in browser console (should be `ws://localhost:3000/ws`)
4. Ensure Vite proxy is configured correctly in `vite.config.ts`

### API Requests Return 404

If API requests fail:

1. Check Vite proxy logs in terminal where you ran `npm run dev`
2. Verify API service is healthy: `curl http://localhost:8080/health`
3. Check API service logs: `docker compose logs -f api`

### Port Already in Use

If port 3000 or 8080 is already in use:

1. Stop conflicting services
2. Or change ports in `vite.config.ts` (web) or docker-compose.yml (API)

## Useful Commands

```bash
# View API service logs
docker compose logs -f api

# View all backend services
docker compose --profile web-dev ps

# Restart API service
docker compose restart api

# Stop all backend services
docker compose --profile web-dev down

# View NATS monitoring
open http://localhost:8222

# View database
docker compose exec db psql -U postgres -d observer
```

## Production Build

To test production build locally:

```bash
# Build web UI
npm run build

# Run full distributed stack (includes Nginx serving the built UI)
docker compose --profile dist up -d

# Access at http://localhost:3000
```
