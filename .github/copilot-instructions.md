# Observer Service - AI Agent Instructions

## Architecture Overview

This is a **test observability system** that ingests test execution events via gRPC. The system operates in two deployment modes:

- **All-in-One (AIO)**: Single container with embedded services (SQLite, embedded NATS via s6-overlay) for dev/local use
- **Distributed**: Multi-container deployment (Postgres, separate NATS, independent services) for production/CI

**Current implementation status**: **Phase 2 Complete** - System is fully decomposed into three independent services with complete NATS JetStream integration (publisher + consumer).

### Service Architecture

```
Test Reporter → Ingestion (gRPC) → NATS JetStream ──┬→ Processor (Consumer) → Database
                                                     │
                                                     └→ API Consumer (Future) → WebSocket → Web UI

                                          Database ← API Service → Web UI (HTTP/GraphQL)
```

**Components:**

1. **Ingestion Service** (`cmd/ingestion/`) - Stateless gRPC endpoint that validates events and publishes to NATS (dual-write with optional DB)
2. **Processor Service** (`cmd/processor/`) - NATS JetStream consumer that persists events to database (requires DB and NATS)
3. **API Service** (`cmd/api/`) - HTTP/GraphQL server for UI and integrations (future implementation)
4. **Legacy Server** (`server/main.go`) - Monolithic backward-compatible server (direct DB writes, optional NATS publish)

**NATS Consumer Pattern:**

- **Processor Consumer** (✅ Implemented): Subscribes to all events, writes to database for persistence
- **API Consumer** (Future): Will subscribe to events, relay real-time updates to Web UI via WebSocket
- Multiple independent consumers can subscribe to the same NATS JetStream stream

**Key directories:**

- `cmd/{ingestion,processor,api}/` - Independent service entrypoints with signal handling
- `pkg/publisher/` - NATS JetStream publisher (wraps event serialization + stream management)
- `pkg/consumer/` - NATS consumer with event routing and DB persistence logic
- `pkg/server/` - gRPC service implementation with panic recovery + logging interceptors
- `internal/database/` - GORM connection supporting both Postgres and SQLite
- `internal/models/` - GORM models (`TestCaseRun`, `StepRun`) with JSON metadata fields
- `tests/` - In-process `bufconn` tests + NATS integration tests

## Critical Dependencies

- **Protobuf schema**: `github.com/stanterprise/proto-go/testsystem/v1` (external, versioned at v0.0.8)
- **NATS JetStream**: Event bus for service decoupling (nats.go v1.47.0)
- **GORM**: Multi-dialect ORM (Postgres + SQLite); uses `datatypes.JSONMap` for metadata
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
make db-up nats-up
./bin/ingestion &
./bin/processor &

# Run Playwright tests with reporter pointing to local Observer
cd stanterprise-playwright-reporter
npm test  # Configure reporter to use localhost:50051
```

## Database Integration Patterns

### Connection Strategy

All services support **optional database mode** via `internal/database/ConnectFromEnv()`:

```go
db, err := database.ConnectFromEnv(logger)
if db == nil {
    logger.Info("DATABASE_URL not set; running without DB")
}
```

**Two DSN formats supported** (auto-detected by driver prefix):

1. **Postgres**: `DATABASE_URL=postgres://user:pass@host:5432/dbname?sslmode=disable`
2. **SQLite**: `DATABASE_URL=/path/to/db.db` or `DATABASE_URL=file:/path/to/db.db`
3. **Split PG vars**: `PGHOST`, `PGPORT`, `PGUSER`, `PGPASSWORD`, `PGDATABASE`, `PGSSLMODE`

SQLite auto-creates parent directories. Postgres uses connection pooling (50 max open, 10 idle, 30min lifetime).

### Auto-Migration

Controlled by `APPLY_MIGRATIONS=1` (or legacy `GORM_AUTO_MIGRATE`). Always check before calling `AutoMigrateSchema()`:

```go
if err := database.AutoMigrateSchema(db, logger); err != nil {
    // Only runs if APPLY_MIGRATIONS=1
}
```

### Idempotent Upsert Pattern

All event handlers (both in-process and NATS consumer) use **GORM upsert via ON CONFLICT** for idempotency:

```go
db.Clauses(clause.OnConflict{
    Columns:   []clause.Column{{Name: "id"}},
    DoUpdates: clause.AssignmentColumns([]string{"status", "updated_at"}),
}).Create(tc)
```

This enables event replay and handles out-of-order delivery. Consumer uses this pattern in `pkg/consumer/nats.go` handlers.

### Transaction Safety

Row-level locking for read-modify-write operations (see `handleStepEnd` in `pkg/consumer/nats.go`):

```go
tx.Clauses(clause.Locking{Strength: "UPDATE"}).
    Where("test_case_run_id = ?", runID).
    Order("created_at DESC").Limit(1).Take(&step)
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
- Retention: WorkQueue policy, 24h max age
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

Located in `pkg/consumer/nats.go`. Pull-based consumer with batch fetching:

```go
consumer, err := consumer.NewNATSConsumer(consumer.NATSConsumerConfig{
    URL:          natsURL,
    StreamName:   "tests_events",
    ConsumerName: "processor",  // Durable consumer name (enables horizontal scaling)
    BatchSize:    10,            // Fetch up to 10 messages at once
    MaxWait:      5 * time.Second,
}, logger, db)

consumer.Start(ctx, cfg)  // Blocks until context cancelled
```

**Consumer configuration:**

- Durable name enables resumption after restart
- Explicit ack policy (must call `msg.Ack()` or `msg.Nak()`)
- MaxDeliver: 5 (retry up to 5 times before DLQ)
- AckWait: 30s (message redelivered if not acked)

**Event routing** (in `processMessage()`):

- Unmarshals event envelope → routes by `event.Type`
- Calls dedicated handlers: `handleTestBegin()`, `handleTestEnd()`, etc.
- Handlers use same GORM upsert patterns as legacy in-process code
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
- DB failures: `codes.Internal` (don't leak internal details)
- NATS publish failures: Log error, continue with DB write (best-effort dual-write in Phase 1)
- Always log errors with context: `logger.Error("persist failed", "id", id, "error", err)`

### Dual-Write Pattern (Phase 1)

Ingestion service optionally writes to both NATS and DB (see `pkg/server/server.go:ReportTestBegin`):

```go
// Publish to NATS if publisher is configured
if s.publisher != nil {
    if err := s.publisher.Publish(ctx, publisher.EventTypeTestBegin, in); err != nil {
        s.logger.Error("publish to NATS failed", "id", in.TestCase.Id, "error", err)
        // Continue with DB write even if NATS publish fails (best-effort)
    }
}

// Persist to DB if configured (optional in ingestion, required in processor)
if s.db != nil {
    // ... GORM upsert logic
}
```

**Future Phase 3**: Remove DB dependency from ingestion entirely (NATS-only).

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

### Run Distributed Mode (Local Development)

```bash
# Start infrastructure
make db-up    # Postgres on :5432
make nats-up  # NATS on :4222, monitoring on :8222

# Run services independently
NATS_URL=nats://localhost:4222 ./bin/ingestion  # gRPC on :50051

DATABASE_URL='postgres://postgres:postgres@localhost:5432/observer?sslmode=disable' \
NATS_URL=nats://localhost:4222 \
./bin/processor

./bin/api  # HTTP on :8080 (future implementation)
```

### Run Legacy Monolithic Mode

```bash
make run-dev  # Builds + runs with DATABASE_URL from docker-compose Postgres
```

### Docker Compose Profiles

**AIO profile** (single container with s6-overlay):

```bash
docker compose --profile aio up -d
# Exposes: :50051 (gRPC), :8080 (API), :4222 (NATS), :8222 (NATS monitoring)
```

**Distributed profile** (separate containers):

```bash
docker compose --profile dist up -d
# Starts: ingestion, processor, api, db, nats
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
- `DATABASE_URL` - Postgres or SQLite DSN (required)
- `APPLY_MIGRATIONS` - Set to `1` to enable auto-migrations

**API Service:**

- `PORT` - HTTP listen port (default: 8080)
- `DATABASE_URL` - Postgres or SQLite DSN (optional for future read-only access)

## Code Style & Patterns

### Import Aliases

Consistent alias for models: `import m "github.com/stanterprise/observer/internal/models"`

### Logger Nil-Safety

All functions accepting `*slog.Logger` must handle nil (critical for library code):

```go
if logger == nil {
    logger = slog.New(slog.NewTextHandler(&noopWriter{}, nil))
}
```

Pattern used in: `pkg/server/New()`, `pkg/publisher/NewNATSPublisher()`, `pkg/consumer/NewNATSConsumer()`

### Metadata Conversion

Protobuf `map[string]string` → GORM `datatypes.JSONMap` (map[string]any):

```go
md := map[string]any{}
for k, v := range protoMetadata {
    md[k] = v
}
tc.Metadata = md  // Stored as JSONB in Postgres, TEXT in SQLite
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

1. Update GORM model in `internal/models/models.go`
2. Test locally: `APPLY_MIGRATIONS=1 make run-dev`
3. For production, consider dedicated migration tool (Goose, golang-migrate) instead of GORM auto-migrate

### Debug Database Connection

```bash
make env-print  # Shows resolved DSN values
make db-psql    # Opens psql shell in Postgres container
```

GORM logs slow queries (>200ms) at `logger.Warn` level. Check connection pool stats in Postgres with `pg_stat_activity`.

### Debug NATS Connection

```bash
make nats-logs  # Tail NATS server logs
curl http://localhost:8222/streaming/channelsz  # JetStream stats
```

Check consumer lag: Fetch consumer info via NATS CLI or monitoring endpoint.

## Migration Roadmap

**Completed:**

- ✅ **Phase 1**: NATS JetStream publisher in ingestion service (dual-write mode) - Commit #64a0f13
- ✅ **Phase 2**: NATS JetStream consumer in processor service - Commit #87b0209
- ✅ Separate service entrypoints (`cmd/ingestion`, `cmd/processor`, `cmd/api`)
- ✅ NATS consumer with event routing and database persistence
- ✅ Docker Compose profiles (AIO + distributed)
- ✅ Multi-stage Dockerfiles for each service
- ✅ Comprehensive test suite (17 tests) with E2E NATS integration validation
- ✅ Playwright reporter integration documentation

**Future phases:**

- [ ] **Phase 3**: Remove dual-write from ingestion (NATS-only, fully stateless)
- [ ] **Phase 4**: API service GraphQL implementation
- [ ] Object storage integration (MinIO/S3) for test artifacts
- [ ] Web UI (React + Tailwind + shadcn/ui)
- [ ] Authentication layer (dev token, OIDC)
- [ ] Observability (Prometheus metrics, OpenTelemetry tracing)
