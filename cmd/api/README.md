# API Service

The API service provides HTTP endpoints for querying test execution data. It offers both GraphQL and REST interfaces for flexible data access.

## Architecture

The API service provides:

1. ✅ **GraphQL API** - Flexible query interface with playground
2. ✅ **REST API** - Simple HTTP endpoints for common operations
3. ⏳ WebSocket connections for real-time updates (future)
4. ⏳ Static file serving for the web UI (future)
5. ⏳ Authentication middleware (future)

## Current State

The API service is **production-ready** with:
- ✅ Complete GraphQL API with schema and resolvers
- ✅ REST endpoints for test and run queries
- ✅ Health check endpoint
- ✅ Database connection (read-only mode)
- ✅ Pagination and filtering support
- ✅ Comprehensive test coverage (11 tests)

## Running

### Without database

```bash
./bin/api
# or
make build-api && ./bin/api
```

### With database (recommended)

```bash
DATABASE_URL='postgres://postgres:postgres@localhost:5432/observer?sslmode=disable' ./bin/api
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
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP listening port |
| `DATABASE_URL` | - | PostgreSQL/SQLite connection string (optional but recommended) |

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

Test the API service manually:

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

## Future Enhancements

- [ ] WebSocket support for real-time updates
- [ ] Web UI static file serving
- [ ] Authentication middleware (dev token, OIDC)
- [ ] Rate limiting
- [ ] CORS configuration
- [ ] Metrics endpoint (Prometheus format)
- [ ] OpenTelemetry tracing
