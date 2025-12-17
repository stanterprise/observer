# Web UI Testing Guide

## Prerequisites

Before testing the Web UI, ensure the following services are running:

1. **MongoDB Database**: `make mongo-up`
2. **NATS Server**: `make nats-up`
3. **Ingestion Service** (creates NATS stream): `NATS_URL='nats://localhost:4222' ./bin/ingestion`
4. **Processor Service** (processes events):
   ```bash
   MONGODB_URI='mongodb://root:password@localhost:27017/observer?authSource=admin' \
   NATS_URL='nats://localhost:4222' \
   ./bin/processor
   ```
5. **API Service** (provides REST API and WebSocket):
   ```bash
   MONGODB_URI='mongodb://root:password@localhost:27017/observer?authSource=admin' \
   NATS_URL='nats://localhost:4222' \
   ./bin/api
   ```

## Development Mode Testing

### Start Web UI Development Server

```bash
cd web
npm install  # First time only
npm run dev
```

The development server will start at `http://localhost:3000` with:

- API proxy: `/api/*` → `http://localhost:8080/api/*`
- WebSocket proxy: `/ws` → `ws://localhost:8080/ws`

### Verify Components

1. **Navigation Bar**: Should show "Observer" logo and connection status
2. **Connection Indicator**: Should show green "Connected" when WebSocket is active
3. **Test Runs Page**: Should display a message "No tests found" initially

## Docker Testing

### Test Distributed Mode

```bash
# Build all images
make docker-build-all

# Start distributed profile
docker compose --profile dist up -d

# Access Web UI
open http://localhost:3000
```

Services in distributed mode:

- **Web UI**: `http://localhost:3000` (Nginx serving React app)
- **API**: Internal (proxied through Web UI)
- **Ingestion**: `localhost:50051` (gRPC)
- **NATS**: `localhost:4222`

### Test AIO Mode

```bash
# Build AIO image
make docker-build-aio

# Start AIO profile
docker compose --profile aio up -d

# Access Web UI
open http://localhost:3000
```

In AIO mode, all services run in a single container with Nginx serving the Web UI.

## Manual UI Testing Checklist

### Navigation

- [ ] Logo displays correctly
- [ ] Connection status shows (green = connected, red = disconnected)
- [ ] "Test Runs" link works

### Test Runs Page

- [ ] Empty state shows when no tests exist
- [ ] Refresh button works
- [ ] Real-time updates work when WebSocket receives events

### API Integration

- [ ] GET `/api/tests` endpoint returns test data
- [ ] Test cards display with correct information
- [ ] Status badges show correct colors (green=passed, red=failed, etc.)
- [ ] Timestamps format correctly

### WebSocket Integration

- [ ] WebSocket connects automatically on page load
- [ ] Connection status updates in header
- [ ] Events received from WebSocket trigger UI refresh
- [ ] Auto-reconnection works after disconnection

## Troubleshooting

### Web UI won't load

- Check that API service is running: `curl http://localhost:8080/health`
- Check browser console for errors
- Verify proxy configuration in `vite.config.ts`

### WebSocket not connecting

- Verify NATS is running: `docker ps | grep nats`
- Check API logs for WebSocket initialization errors
- Ensure NATS stream exists (created by ingestion service)

### API requests failing

- Verify database connection
- Check processor service logs
- Ensure MongoDB is running and reachable via `MONGODB_URI`

## Environment Variables

For production deployment, configure:

| Variable           | Description                  | Default             |
| ------------------ | ---------------------------- | ------------------- |
| `VITE_API_URL`     | Base URL for API requests    | `/api`              |
| `VITE_WS_URL`      | WebSocket endpoint URL       | Auto-detected       |
| `API_BACKEND_HOST` | Nginx upstream host (Docker) | `localhost` / `api` |
| `API_BACKEND_PORT` | Nginx upstream port (Docker) | `8080`              |

## Screenshots

When testing, take screenshots of:

1. Empty state (no tests)
2. Test list with sample data
3. Connection status indicators
4. WebSocket events in browser DevTools Network tab
