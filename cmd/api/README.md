# API Service

The API service provides HTTP endpoints for the web UI and external integrations. It serves as the query interface for test data and provides **real-time event streaming via WebSocket**.

## Architecture

The API service provides:

1. **WebSocket endpoint for real-time test events** ✅
2. GraphQL API for flexible querying (future)
3. RESTful endpoints for simple operations (future)
4. Static file serving for the web UI (future)
5. Authentication middleware (OIDC in distributed mode) (future)

## Current State

The API service includes:
- **WebSocket endpoint (`/ws`) for real-time event streaming** ✅
- **NATS JetStream consumer for event relay** ✅
- Health check endpoint (`/health`)
- Basic information endpoint (`/`)
- Database connection (read-only mode, optional)

## Running

### Without database or NATS (minimal mode)

```bash
./bin/api
# or
make build-api && ./bin/api
```

### With database (read-only)

```bash
DATABASE_URL='postgres://postgres:postgres@localhost:5432/observer?sslmode=disable' ./bin/api
```

### With NATS for WebSocket events

```bash
NATS_URL='nats://localhost:4222' ./bin/api
```

### Full configuration (database + WebSocket)

```bash
DATABASE_URL='postgres://postgres:postgres@localhost:5432/observer?sslmode=disable' \
NATS_URL='nats://localhost:4222' \
./bin/api
```

Default port: `8080`

### Custom port

```bash
PORT=3000 ./bin/api
```

## Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/` | GET | Service information |
| `/health` | GET | Health check |
| `/ws` | WebSocket | Real-time event stream ✅ |
| `/api/graphql` | POST | GraphQL endpoint (future) |
| `/metrics` | GET | Prometheus metrics (future) |

## WebSocket Real-Time Events

The `/ws` endpoint provides real-time streaming of test execution events.

### Connection

```javascript
const ws = new WebSocket('ws://localhost:8080/ws');

ws.onopen = () => console.log('Connected');
ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Event:', data.type, data);
};
```

### Event Format

All events follow this structure:

```json
{
  "type": "test.begin|test.end|step.begin|step.end",
  "timestamp": "2025-11-14T05:00:00Z",
  "data": { /* event-specific protobuf data */ }
}
```

### Event Types

- `test.begin` - Test case execution started
- `test.end` - Test case execution completed
- `step.begin` - Test step started
- `step.end` - Test step completed

### Test Client

A simple HTML test client is available at [`../../docs/websocket-test-client.html`](../../docs/websocket-test-client.html). Open it in a browser and connect to `ws://localhost:8080/ws` to view real-time events.

**Requirements**: WebSocket functionality requires `NATS_URL` to be configured. Without NATS, the endpoint will accept connections but won't relay events.

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP listening port |
| `DATABASE_URL` | - | PostgreSQL connection string (optional) |
| `NATS_URL` | - | NATS server URL for WebSocket (optional) |
| `NATS_STREAM` | `tests_events` | JetStream stream name |
| `NATS_WS_CONSUMER` | `websocket` | Consumer name for WebSocket |
| `AUTH_MODE` | `dev` | Authentication mode: `dev` or `oidc` (future) |
| `OIDC_ISSUER` | - | OIDC issuer URL (future) |

## Testing

### Basic HTTP endpoints

```bash
# Health check
curl http://localhost:8080/health

# Service info
curl http://localhost:8080/
```

### WebSocket connection

1. Start NATS:
   ```bash
   make nats-up
   ```

2. Start API service with NATS:
   ```bash
   NATS_URL='nats://localhost:4222' ./bin/api
   ```

3. Open the test client:
   ```bash
   open docs/websocket-test-client.html
   ```

4. In another terminal, send test events via ingestion service:
   ```bash
   # Start ingestion
   NATS_URL='nats://localhost:4222' ./bin/ingestion
   
   # Send test events (use your test reporter or manual gRPC calls)
   ```

5. Watch events appear in real-time in the test client

## Future Enhancements

- [ ] GraphQL API implementation (using gqlgen)
- [x] ~~WebSocket support for real-time updates~~ ✅ Completed
- [ ] Web UI static file serving
- [ ] Authentication middleware (dev token, OIDC)
- [ ] Rate limiting
- [ ] CORS configuration
- [ ] Metrics endpoint
- [ ] OpenTelemetry tracing
