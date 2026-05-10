# Observer Service - AI Agent Instructions

## Architecture Overview

This is a **test observability system** that ingests test execution events via gRPC. The system operates in two deployment modes:

- **All-in-One (AIO)**: Single container with embedded services (MongoDB, embedded NATS via s6-overlay) for dev/local use
- **Distributed**: Multi-container deployment (MongoDB, separate NATS, independent services) for production/CI

**Current implementation status**: **Phase 3+ In Progress** - System has completed distributed architecture with WebSocket real-time streaming and Web UI implementation.

### Service Architecture

```
Test Reporter → Ingestion (gRPC) → NATS JetStream ──┬→ Processor (Consumer) → Database
                                                     │
                                                     └→ API Consumer (WebSocket) → Web UI (React)

                                          Database ← API Service (REST + WebSocket; GraphQL planned) → Web UI
```

**Components:**

1. **Ingestion Service** (`cmd/ingestion/`) - Stateless gRPC endpoint that validates events and publishes to NATS (no DB writes)
2. **Processor Service** (`cmd/processor/`) - NATS JetStream consumer that persists events to database (requires DB and NATS)
3. **API Service** (`cmd/api/`) - HTTP REST server + WebSocket endpoint for real-time events (GraphQL planned)
4. **Web UI** (`web/`) - React + TypeScript + Tailwind CSS dashboard with real-time updates (✅ Implemented)
5. **Legacy Server** (`server/main.go`) - Monolithic backward-compatible server (direct DB writes, optional NATS publish)

**NATS Consumer Pattern:**

- **Processor Consumer** (✅ Implemented): Subscribes to all events, writes to database for persistence
- **WebSocket Consumer** (✅ Implemented): Subscribes to events, relays real-time updates to Web UI clients via WebSocket
- Multiple independent consumers can subscribe to the same NATS JetStream stream

**Key directories:**

- `cmd/{ingestion,processor,api}/` - Independent service entrypoints with signal handling
- `pkg/publisher/` - NATS JetStream publisher (wraps event serialization + stream management)
- `pkg/consumer/` - NATS consumer with event routing and DB persistence logic
- `pkg/server/` - gRPC service implementation with panic recovery + logging interceptors
- `pkg/websocket/` - WebSocket hub for real-time event streaming to web clients
- `pkg/api/` - REST API handlers (GraphQL planned/stubbed)
- `web/` - React + TypeScript + Tailwind CSS web interface
- `internal/database/` - MongoDB connection helper (`ConnectMongoDBFromEnv`)
- `internal/models/` - MongoDB document models (`TestRunDocument`, `SuiteDocument`, `TestDocument`, `StepDocument`)
- `internal/repository/` - MongoDB repository (document upserts, nested updates)
- `tests/` - In-process `bufconn` tests + NATS integration tests

## Critical Dependencies

- **Protobuf schema**: `github.com/stanterprise/proto-go/testsystem/v1` (external, versioned at v0.0.14)
- **NATS JetStream**: Event bus for service decoupling (nats.go v1.47.0)
- **MongoDB**: Document database via official Go driver (`go.mongodb.org/mongo-driver`)
- **slog**: Go 1.21+ structured logging (always use `*slog.Logger`, never `log.Printf`)
- **Playwright Reporter**: `github.com/stanterprise/stanterprise-playwright-reporter` - Custom reporter for integration testing

## Test Reporter Integration

The **Playwright custom reporter** is maintained in a separate repository: `github.com/stanterprise/stanterprise-playwright-reporter`

This reporter serves as:

- **Reference implementation** for the gRPC protocol
- **Integration test client** for Observer development
- **Production reporter** for Playwright test suites

The reporter can be installed directly from GitHub via npm:

```bash
npm install github:stanterprise/stanterprise-playwright-reporter
```

When developing or testing Observer:

1. Use the Playwright reporter codebase to generate realistic test events
2. Validate protobuf schema compatibility between reporter and Observer
3. Test end-to-end flows: Playwright → Reporter → Observer gRPC → NATS → DB
4. Verify metadata handling and event sequencing

**Development workflow**:

```bash
# Clone reporter repository
git clone https://github.com/stanterprise/stanterprise-playwright-reporter

# Start Observer locally
make mongo-up nats-up
./bin/ingestion &
./bin/processor &

# Run Playwright tests with reporter pointing to local Observer
cd stanterprise-playwright-reporter
npm test  # Configure reporter to use localhost:50051
```

## Database Integration Patterns

### Connection Strategy

Services connect to MongoDB via `internal/database/ConnectMongoDBFromEnv()`.

- **Ingestion** is stateless and does not require MongoDB.
- **Processor** requires MongoDB (it persists events).
- **API** requires MongoDB (it serves REST queries).

Connection pattern:

```go
mongoDB, err := database.ConnectMongoDBFromEnv(logger)
if err != nil {
    logger.Error("mongodb connect failed", "error", err)
    os.Exit(1)
}
if mongoDB == nil {
    logger.Error("MONGODB_URI or MONGO_URI not set")
    os.Exit(1)
}
defer mongoDB.Close(context.Background())

repo := repository.NewMongoRepository(mongoDB.TestRunsCollection(), logger)
```

**MongoDB URI formats supported**:

1. **Single URI env var**: `MONGODB_URI` (preferred) or `MONGO_URI`
   - Example: `MONGODB_URI=mongodb://user:pass@host:27017/observer?authSource=admin`
   - Atlas example: `mongodb+srv://user:pass@cluster/observer`
2. **Split vars** (assembled automatically):
   - `MONGO_HOST`, `MONGO_PORT` (default `27017`)
   - `MONGO_USER`, `MONGO_PASSWORD`
   - `MONGO_DATABASE` (default `observer`)
   - `MONGO_AUTH_SOURCE` (default `admin`)

MongoDB connections use conservative pooling defaults (see `internal/database/mongodb.go`).

### Schema Evolution (MongoDB)

MongoDB is schema-flexible. Prefer **additive** changes:

- Add new fields with sensible defaults (treat missing fields as zero values).
- Avoid renaming fields; if needed, read both old and new fields during a transition.
- Keep document growth bounded (steps can be large; be mindful of the 16MB document limit).

### Idempotent Upsert Pattern

All persistence handlers use **idempotent upserts** so JetStream redelivery and event replay are safe.

For root suite begin, the repository uses an `UpdateOne` with `upsert=true` + `$setOnInsert`:

```go
opts := options.Update().SetUpsert(true)
filter := bson.M{"_id": suite.ID}
update := bson.M{
    "$setOnInsert": bson.M{
        "_id":        suite.ID,
        "created_at": now,
        "tests":      []*m.TestDocument{},
    },
    "$set": bson.M{
        "name":       suite.Name,
        "metadata":   suite.Metadata,
        "updated_at": now,
    },
}
_, err := r.collection.UpdateOne(ctx, filter, update, opts)
```

For nested suite/test/step updates, the repository uses **single-document atomic updates** with `$set`/`$push` and `arrayFilters`.

### Transaction Safety

MongoDB updates are designed to be safe without explicit locks by using atomic updates.

Critical safety rule used throughout the repository: **always include the root `_id` in filters** when updating nested arrays to avoid cross-document mutations.

Example (note the `_id` guard):

```go
filter := bson.M{
    "_id":             rootDocID, // CRITICAL: Prevent cross-document mutation
    "suites.id":       suiteID,
    "suites.tests.id": test.ID,
}
```

## NATS Integration Patterns

### Publisher (Ingestion Service)

Located in `pkg/publisher/nats.go`. Wraps events in envelope with timestamp + type:

```go
type Event struct {
    Type      EventType       `json:"type"`  // test.begin, test.end, step.begin, step.end
    Timestamp time.Time       `json:"timestamp"`
    Data      json.RawMessage `json:"data"`  // Protobuf request marshaled to JSON
}
```

**Stream configuration** (auto-created on startup):

- Name: `tests_events` (configurable via `NATS_STREAM`)
- Subjects: `tests.events.v1.>` (prefix configurable via `NATS_SUBJECT_PREFIX`)
- Retention: Limits policy, 24h max age (supports multiple independent consumers)
- Storage: File-based (survives restarts)

**Usage pattern** (see `cmd/ingestion/main.go`):

```go
pub, err := publisher.NewNATSPublisher(publisher.NATSConfig{
    URL: natsURL,
    StreamName: envOr("NATS_STREAM", publisher.DefaultStreamName),
}, logger)
defer pub.Close()

// In gRPC handler:
pub.Publish(ctx, publisher.EventTypeTestBegin, protoRequest)
```

### Consumer (Processor Service) ✅ **Phase 2 Complete**

Located in `pkg/consumer/nats_mongodb.go`. Pull-based JetStream consumer with batch fetching, persisting to MongoDB via `internal/repository`:

```go
mongoDB, err := database.ConnectMongoDBFromEnv(logger)
if err != nil || mongoDB == nil {
    // processor requires MongoDB
}
repo := repository.NewMongoRepository(mongoDB.TestRunsCollection(), logger)

natsConsumer, err := consumer.NewMongoNATSConsumer(consumer.MongoNATSConsumerConfig{
    URL:          natsURL,
    StreamName:   "tests_events",
    ConsumerName: "processor", // Durable consumer name (enables horizontal scaling)
    BatchSize:    10,
    MaxWait:      5 * time.Second,
}, logger, repo)

natsConsumer.Start(ctx, cfg) // Blocks until context cancelled
```

**Consumer configuration:**

- Durable name enables resumption after restart
- Explicit ack policy (must call `msg.Ack()` or `msg.Nak()`)
- MaxDeliver: 5 (retry up to 5 times before DLQ)
- AckWait: 30s (message redelivered if not acked)

**Event routing** (in `processMessage()`):

- Unmarshals event envelope → routes by `event.Type`
- Calls dedicated handlers: `handleTestBegin()`, `handleTestEnd()`, etc.
- Handlers delegate persistence to MongoDB repository upsert methods
- Unknown event types are acked to prevent redelivery loop

**Processor Service Architecture:**

- No longer runs gRPC server (transformed in commit 87b0209)
- Pure NATS consumer with database persistence
- Supports horizontal scaling via durable consumer groups
- Graceful shutdown with context cancellation

## gRPC Service Conventions

### Interceptor Chain

All servers use this chain via `pkg/server/NewGRPCServer()` (order matters):

1. `recoveryInterceptor` - Catches panics, logs stack trace, returns `Internal` status
2. `loggingInterceptor` - Logs RPC method, duration, peer address, status code

### Validation Pattern

Validate required fields early and return `InvalidArgument`:

```go
if in == nil || in.TestCase == nil {
    return nil, status.Error(codes.InvalidArgument, "test_case is required")
}
if err := validateTestID(in.TestCase.Id); err != nil {
    return nil, status.Error(codes.InvalidArgument, err.Error())
}
```

### Error Handling

- Client errors (bad input): `codes.InvalidArgument`
- MongoDB failures: `codes.Internal` (don't leak internal details)
- NATS publish failures: return `codes.Internal` (ingestion is NATS-first)
- Always log errors with context: `logger.Error("persist failed", "id", id, "error", err)`

### Persistence Boundary

The ingestion service is stateless: it validates gRPC requests and publishes to JetStream.

- Persistence happens in the processor (JetStream consumer) via MongoDB.
- The API reads from MongoDB and can optionally relay events over WebSocket by consuming from JetStream.

## Testing Strategy

### Unit Tests with bufconn

All gRPC tests use **in-process `bufconn.Listener`** - no TCP ports, no race conditions. See `tests/helper_test.go`:

```go
testBufListener = bufconn.Listen(bufSize)
testGRPCServer.Serve(testBufListener)

// Client connects via custom dialer:
grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
    return testBufListener.Dial()
}), grpc.WithInsecure())
```

**TestMain pattern**: Setup server once in `TestMain()`, ensure graceful shutdown on test exit.

### NATS Integration Tests

Located in `tests/nats_integration_test.go`. Requires external NATS server:

```bash
make nats-up  # Starts NATS via docker-compose
make test-nats-integration  # Sets NATS_TEST_URL=nats://localhost:4222
```

Tests validate:

- Event publishing → consumption → DB persistence round-trip
- Consumer acknowledgment and redelivery behavior
- Stream/consumer auto-creation

## Build & Run Workflows

### Build All Components

```bash
make build-all  # Builds: observer, ingestion, processor, api → bin/
```

Individual targets: `make build-ingestion`, `make build-processor`, `make build-api`

### Local Development Modes

**Option 1: Full Docker (Recommended)**

```bash
# Start all backend services (DB, NATS, ingestion, processor, API)
docker compose --profile web-dev up -d

# Run web UI locally with hot reload
cd web && npm install && npm run dev  # Opens http://localhost:3000
```

**Option 2: Hybrid (Backend in Docker, Services locally)**

```bash
# Start infrastructure only
make mongo-up nats-up

# Run services individually (separate terminals)
NATS_URL=nats://localhost:4222 ./bin/ingestion
MONGODB_URI='mongodb://root:password@localhost:27017/observer?authSource=admin' \
    NATS_URL=nats://localhost:4222 ./bin/processor
MONGODB_URI='mongodb://root:password@localhost:27017/observer?authSource=admin' \
    NATS_URL=nats://localhost:4222 ./bin/api

# Run web UI
cd web && npm run dev
```

**Option 3: Legacy Monolithic Mode**

```bash
make run-dev  # Single process with MONGODB_URI (see Makefile defaults)
```

### Docker Compose Profiles

**web-dev profile** (backend services for local web development):

```bash
docker compose --profile web-dev up -d
# Starts: mongodb, nats, ingestion, processor, api
# Run web UI locally: cd web && npm run dev
```

**dist profile** (full distributed deployment):

```bash
docker compose --profile dist up -d
# Starts: mongodb, nats, ingestion, processor, api, web (Nginx)
# Access web UI at http://localhost:3000
```

**aio profile** (single container with s6-overlay):

```bash
docker compose --profile aio up -d
# Single container with embedded NATS, MongoDB, all services
# Exposes: :3000 (Web), :50051 (gRPC), :8080 (API), :4222 (NATS)
```

### MongoDB Reset / Clear

```bash
make mongo-reset
make mongo-clear
```

### Environment Variables

**Ingestion Service:**

- `PORT` - gRPC listen port (default: 50051)
- `NATS_URL` - NATS server URL (optional, e.g., `nats://localhost:4222`)
- `NATS_STREAM` - JetStream stream name (default: `tests_events`)
- `NATS_SUBJECT_PREFIX` - Subject prefix (default: `tests.events.v1`)

**Processor Service:**

- `NATS_URL` - NATS server URL (required)
- `NATS_STREAM`, `NATS_CONSUMER` - Stream and consumer names
- `MONGODB_URI` or `MONGO_URI` - MongoDB connection string (required)
- `MONGO_HOST`, `MONGO_PORT`, `MONGO_USER`, `MONGO_PASSWORD`, `MONGO_DATABASE`, `MONGO_AUTH_SOURCE` - Split MongoDB vars (alternative)

**API Service:**

- `PORT` - HTTP listen port (default: 8080)
- `MONGODB_URI` or `MONGO_URI` - MongoDB connection string (required)
- `NATS_URL` - NATS server URL (optional, for WebSocket relay)
- `NATS_STREAM` - JetStream stream name (default: `tests_events`)
- `NATS_WS_CONSUMER` - Consumer name for WebSocket (default: `websocket`)
- `CORS_ALLOWED_ORIGINS` - CORS origins (default: `*` for development)

**Web UI** (environment variables injected via Nginx template):

- `API_BACKEND_HOST` - API service hostname (default: `api` in Docker, `localhost` for dev)
- `API_BACKEND_PORT` - API service port (default: `8080`)

**Development**: Web UI uses Vite proxy in dev mode (`vite.config.ts`):

- `/api` → `http://localhost:8080`
- `/ws` → `ws://localhost:8080`

**Production**: Nginx reverse proxy configuration (`docker/nginx/nginx.conf.template`):

- `/api` → `http://${API_BACKEND_HOST}:${API_BACKEND_PORT}`
- `/ws` → `ws://${API_BACKEND_HOST}:${API_BACKEND_PORT}`

## Code Style & Patterns

## Frontend Implementation Guardrails (Stitch)

When implementing or editing UI under `web/src/**`, treat the Stitch system as mandatory, not advisory.

### Source of Truth

- Design tokens and usage guidance: `web/src/lib/styleGuide.ts`
- Human-readable frontend policy: `web/STYLE_GUIDE.md`
- Font loading and font variables: `web/src/index.css`

### Required Visual Rules

- Use tonal layering for separation. Do not rely on visible borders for card structure.
- Card depth should come from background tier contrast (`surface` tokens), not drop shadows.
- Use ghost borders only when explicitly required for accessibility or when demonstrating ghost-border behavior.
- Light mode typography: Space Grotesk for display/headlines, Inter for body/labels.
- Dark mode typography: Space Grotesk for display/headlines/body/labels unless a page explicitly overrides by spec.
- Primary actions must use the Stitch gradient direction and token pairing for the selected variant.
- Status chips and component examples must use Stitch token mappings (success/error/flaky) for the active variant.

### Required Validation for Frontend Changes

For any UI-affecting change in `web/src/**`, run:

```bash
cd web && npm run build
```

and verify:

1. No TypeScript errors.
2. No unintended fallback fonts.
3. Visible tonal contrast between page background, section cards, and nested cards in both light and dark variants when applicable.

### Implementation Priority

When there is a conflict between legacy component defaults and Stitch guidance, prefer Stitch guidance for new or modified frontend work.

### Import Aliases

Consistent alias for document models: `import m "github.com/stanterprise/observer/internal/models"`

### Logger Nil-Safety

All functions accepting `*slog.Logger` must handle nil (critical for library code):

```go
if logger == nil {
    logger = slog.New(slog.NewTextHandler(&noopWriter{}, nil))
}
```

Pattern used in: `pkg/server/New()`, `pkg/publisher/NewNATSPublisher()`, `pkg/consumer/NewNATSConsumer()`

### Metadata Conversion

Protobuf `map[string]string` → MongoDB document metadata (`map[string]interface{}`):

```go
md := make(map[string]interface{})
for k, v := range protoMetadata {
    md[k] = v
}
doc.Metadata = md
```

### Graceful Shutdown Pattern

All service entrypoints follow this pattern (see `cmd/ingestion/main.go`, `cmd/processor/main.go`):

```go
sigCh := make(chan os.Signal, 1)
signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

select {
case sig := <-sigCh:
    logger.Info("shutdown signal received", "signal", sig)
case err := <-errChan:
    if err != nil {
        logger.Error("server serve error", "error", err)
    }
}

// Graceful stop with timeout
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
// ... shutdown logic
```

## Common Tasks

### Add New gRPC Method

1. Update protobuf in `stanterprise/proto-go` repository
2. Bump version in `go.mod`: `go get github.com/stanterprise/proto-go@v0.0.X`
3. Implement handler in `pkg/server/server.go` following validation → log → publish pattern
4. Add corresponding consumer handler in `pkg/consumer/nats.go`
5. Add bufconn test in `tests/main_test.go`

### Add DB Migration

MongoDB does not require schema migrations in the same way as SQL.

1. Update document structs in `internal/models/` (additive fields preferred)
2. Update repository write paths in `internal/repository/` (keep `_id` guards and arrayFilters)
3. Update API read/response shapes in `pkg/api/rest_mongodb.go` (if needed)
4. Add/adjust tests (unit + integration) to cover new fields

### Debug Database Connection

```bash
make mongo-env-print  # Shows resolved Mongo env values
make mongo-shell      # Opens mongosh in MongoDB container
```

For connectivity issues:

- Verify container health: `docker compose ps mongodb`
- Ping from shell: `mongosh --eval "db.adminCommand('ping')"`

### Debug NATS Connection

```bash
make nats-logs  # Tail NATS server logs
curl http://localhost:8222/streaming/channelsz  # JetStream stats
```

Check consumer lag: Fetch consumer info via NATS CLI or monitoring endpoint.
