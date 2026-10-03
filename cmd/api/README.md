# API Service

The API service provides PostgreSQL-backed REST endpoints for the web UI and external integrations, plus an optional NATS-backed WebSocket relay for live events.

## Architecture

The API service currently provides:

1. REST endpoints for PostgreSQL-backed test and run queries.
2. A `/ws` endpoint for live event streaming when NATS is configured.
3. `/health` and `/` service endpoints.

## Current State

REST reads require PostgreSQL. WebSocket connections can be accepted without NATS, but events are relayed only when `NATS_URL` is configured.

## Running

### Without database or NATS (minimal mode)

```bash
./bin/api
# or
make build-api && ./bin/api
```

### With PostgreSQL (recommended)

```bash
DATABASE_URL='postgres://observer:observer@localhost:5432/observer?sslmode=disable' ./bin/api
```

### With NATS for WebSocket events

```bash
NATS_URL='nats://localhost:4222' ./bin/api
```

### Full configuration (PostgreSQL + WebSocket)

```bash
DATABASE_URL='postgres://observer:observer@localhost:5432/observer?sslmode=disable' \
NATS_URL='nats://localhost:4222' \
./bin/api
```

Default port: `8080`

### Custom port

```bash
PORT=3000 ./bin/api
```

## Endpoints

### General

| Endpoint  | Method    | Description               |
| --------- | --------- | ------------------------- |
| `/`       | GET       | Service information       |
| `/health` | GET       | Health check              |
| `/ws`     | WebSocket | Real-time event stream ✅ |

GraphQL (`/api/graphql`, `/api/playground`), Prometheus metrics (`/metrics`), and authentication/OIDC are not currently registered or implemented by this service. Do not use them as available interfaces.

### REST API

| Endpoint                           | Method | Description                              |
| ---------------------------------- | ------ | ---------------------------------------- |
| `/api/tests`                       | GET    | List tests with filtering and pagination |
| `/api/tests/{testId}/trends`       | GET    | Test outcome and duration history        |
| `/api/runs`                        | GET    | List runs                                |
| `/api/runs/stats`                  | GET    | List run statistics                      |
| `/api/runs/{runId}`                | GET    | Get run statistics and tests             |
| `/api/runs/{runId}/tests/{testId}` | GET    | Get a test within a run                  |

**Query Parameters for `/api/tests`:**

- `runId` - Filter by run ID
- `status` - Filter by status (PASSED, FAILED, SKIPPED)
- `project` - Filter by project name
- `limit` - Number of results (default: 20)
- `offset` - Pagination offset (default: 0)

**Example REST Queries:**

```bash
# List all tests
curl http://localhost:8080/api/tests

# Filter by status
curl http://localhost:8080/api/tests?status=PASSED

# Filter by run
curl http://localhost:8080/api/tests?runId=run-1

# Get specific test with steps
# Get a test within a run
curl http://localhost:8080/api/runs/run-1/tests/test-123

# Get run statistics
curl http://localhost:8080/api/runs/run-1
```

## WebSocket Real-Time Events

The `/ws` endpoint provides real-time streaming of test execution events.

### Connection

```javascript
const ws = new WebSocket("ws://localhost:8080/ws");

ws.onopen = () => console.log("Connected");
ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log("Event:", data.type, data);
};
```

### Event Format

All events follow this structure:

```json
{
  "type": "test.begin|test.end|step.begin|step.end",
  "timestamp": "2025-11-14T05:00:00Z",
  "data": {
    /* event-specific protobuf data */
  }
}
```

### Event Types

- `test.begin` - Test case execution started
- `test.end` - Test case execution completed
- `step.begin` - Test step started
- `step.end` - Test step completed

The WebSocket implementation is maintained alongside the service code in `pkg/api/websocket/`. Connect a WebSocket client to `ws://localhost:8080/ws`; live events require `NATS_URL` to be configured.

## Environment Variables

| Variable           | Default        | Description                                          |
| ------------------ | -------------- | ---------------------------------------------------- |
| `PORT`             | `8080`         | HTTP listening port                                  |
| `DATABASE_URL`     | -              | PostgreSQL connection string (required for REST API) |
| `POSTGRES_DSN`     | -              | Alternate PostgreSQL DSN env var                     |
| `NATS_URL`         | -              | NATS server URL for WebSocket (optional)             |
| `NATS_STREAM`      | `tests_events` | JetStream stream name                                |
| `NATS_WS_CONSUMER` | `websocket`    | Consumer name for WebSocket                          |

## Testing

Run the API tests:

```bash
go test ./cmd/api/... -v
```

```bash
# Health check
curl http://localhost:8080/health

# Service info
curl http://localhost:8080/

# REST API
curl http://localhost:8080/api/tests
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

3. In another terminal, send test events via ingestion service:

   ```bash
   # Start ingestion
   NATS_URL='nats://localhost:4222' ./bin/ingestion

   # Send test events (use your test reporter or manual gRPC calls)
   ```

4. Observe events from your WebSocket client.

## Future Enhancements

- GraphQL API
- Authentication and OIDC
- [ ] Rate limiting
- Prometheus metrics and OpenTelemetry tracing

For public product status, see the [Observer roadmap](https://observer.stanterprise.com/docs/roadmap/).
