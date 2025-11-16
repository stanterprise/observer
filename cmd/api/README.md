# API Service

The API service provides HTTP endpoints for the web UI and external integrations. It offers both GraphQL and REST interfaces for flexible data access and provides **real-time event streaming via WebSocket**.

## Architecture

The API service provides:

1. ✅ **GraphQL API** - Flexible query interface with playground
2. ✅ **REST API** - Simple HTTP endpoints for common operations
3. ✅ **WebSocket endpoint for real-time test events**
4. Static file serving for the web UI (future)
5. Authentication middleware (OIDC in distributed mode) (future)

## Current State

The API service is **production-ready** with:
- ✅ Complete GraphQL API with schema and resolvers
- ✅ REST endpoints for test and run queries
- ✅ Health check endpoint
- ✅ Database connection (read-only mode)
- ✅ Pagination and filtering support
- ✅ Comprehensive test coverage (11 tests)

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

### With database (recommended)

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

### General

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/` | GET | Service information |
| `/health` | GET | Health check |
| `/ws` | WebSocket | Real-time event stream ✅ |
| `/api/graphql` | POST | GraphQL endpoint (future) |
| `/metrics` | GET | Prometheus metrics (future) |

### GraphQL API

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/graphql` | POST | GraphQL API endpoint |
| `/api/playground` | GET | GraphQL Playground (interactive UI) |

**Example GraphQL Query:**

```graphql
query {
  testCases(filter: { status: "PASSED" }, limit: 10) {
    nodes {
      id
      title
      status
      steps {
        id
        status
      }
    }
    pageInfo {
      totalCount
      hasNextPage
    }
  }
}
```

### REST API

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/tests` | GET | List test cases (supports filtering/pagination) |
| `/api/tests/{id}` | GET | Get specific test case with steps |
| `/api/runs` | GET | List all test run IDs |
| `/api/runs/{runId}` | GET | Get run statistics and tests |

**Query Parameters for `/api/tests`:**
- `runId` - Filter by run ID
- `status` - Filter by status (PASSED, FAILED, SKIPPED)
- `search` - Search in test titles (case-insensitive)
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
curl http://localhost:8080/api/tests/test-123

# Get run statistics
curl http://localhost:8080/api/runs/run-1
`>>>> master

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

## GraphQL Schema

### Types

- **TestCaseRun** - Represents a test execution
  - `id`, `runId`, `title`, `status`, `metadata`, `createdAt`, `updatedAt`
  - `steps` - Associated step runs

- **StepRun** - Represents a test step execution
  - `id`, `runId`, `testCaseRunId`, `status`, `createdAt`, `updatedAt`
  - `testCase` - Parent test case

- **RunStats** - Test run statistics
  - `totalTests`, `passedTests`, `failedTests`, `skippedTests`, `totalSteps`

### Queries

- `testCase(id: ID!)` - Get single test case
- `testCases(filter, limit, offset)` - List test cases with filtering
- `step(id: ID!)` - Get single step
- `testRuns(limit, offset)` - List run IDs
- `runStats(runId: String!)` - Get run statistics

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

# GraphQL query
curl -X POST http://localhost:8080/api/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{ testCases(limit: 5) { nodes { id title } } }"}'

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

- [ ] Web UI static file serving
- [ ] Authentication middleware (dev token, OIDC)
- [ ] Rate limiting
- [ ] CORS configuration
- [ ] Metrics endpoint (Prometheus format)
- [ ] OpenTelemetry tracing
