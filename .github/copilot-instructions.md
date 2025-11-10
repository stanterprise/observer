# Observer Service - AI Agent Instructions

## Architecture Overview

This is a **test observability system** that ingests test execution events via gRPC. The service operates in two modes:

- **All-in-One (AIO)**: Single container with embedded services (SQLite, local NATS) for dev/local use
- **Distributed**: Multi-container deployment (Postgres, separate NATS, MinIO) for production/CI

**Current state**: This repository implements the **gRPC ingestion server** component. The full distributed architecture (NATS event bus, processor service, API/UI) is documented in `docs/architecture/` but not yet implemented.

Key directories:

- `server/main.go` - Entrypoint with graceful shutdown and signal handling
- `pkg/server/` - gRPC service implementation with structured logging and panic recovery interceptors
- `pkg/database/` - GORM-based Postgres connection with dual DSN support (URL or PG\* env vars)
- `pkg/models/` - GORM models mapping to protobuf entities (`TestCaseRun`, `StepRun`)
- `tests/` - Integration tests using in-process `bufconn` (no external ports)

## Critical Dependencies

- **Protobuf schema**: `github.com/stanterprise/proto-go/testsystem/v1` (external, versioned at v0.0.8)
- **GORM**: ORM with Postgres driver; uses `datatypes.JSONMap` for metadata fields
- **slog**: Go 1.21+ structured logging (use `slog.Logger`, not `log.Printf`)

## Database Integration Patterns

### Connection Strategy

The service supports **optional database mode**:

```go
db, err := database.ConnectFromEnv(logger)
if db == nil {
    logger.Info("DATABASE_URL not set; running without DB")
}
```

**Two DSN formats supported** (see `pkg/database/database.go`):

1. `DATABASE_URL=postgres://user:pass@host:5432/dbname?sslmode=disable`
2. Split env vars: `PGHOST`, `PGPORT`, `PGUSER`, `PGPASSWORD`, `PGDATABASE`, `PGSSLMODE`

### Auto-Migration

Controlled by `APPLY_MIGRATIONS=1` (or legacy `GORM_AUTO_MIGRATE`). Always check this env var before running migrations.

### Upsert Pattern

All event handlers use **GORM upsert via ON CONFLICT**:

```go
db.Clauses(clause.OnConflict{
    Columns:   []clause.Column{{Name: "id"}},
    DoUpdates: clause.AssignmentColumns([]string{"status", "updated_at"}),
}).Create(tc)
```

This allows idempotent event replay and handles out-of-order delivery.

### Transaction Safety

`ReportStepEnd` uses row-level locking to prevent races:

```go
tx.Clauses(clause.Locking{Strength: "UPDATE"}).
    Where("test_case_run_id = ?", runID).
    Order("created_at DESC").Limit(1).Take(&step)
```

Use transactions for read-modify-write operations on shared resources.

## gRPC Service Conventions

### Interceptor Chain

All servers use this chain (order matters):

1. `recoveryInterceptor` - Catches panics, logs stack trace, returns `Internal` status
2. `loggingInterceptor` - Logs RPC method, duration, peer, status code

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

- Client errors: `codes.InvalidArgument`
- DB failures: `codes.Internal` (don't leak internal details)
- Always log errors with context: `logger.Error("persist failed", "id", id, "error", err)`

## Testing Strategy

**All tests use in-process `bufconn`** - no TCP ports, no race conditions from port allocation. See `tests/helper_test.go`:

```go
testBufListener = bufconn.Listen(bufSize)
testGRPCServer.Serve(testBufListener)
```

Client connections use `grpc.WithContextDialer` pointing to `bufconn.Listener.Dial`.

**TestMain pattern**: Setup server in `TestMain`, ensure graceful shutdown on test exit.

## Build & Run Workflows

### Local Development

```bash
make run-dev          # Starts with DATABASE_URL pointing to docker-compose Postgres
make db-up            # Starts only Postgres container
make db-psql          # Opens psql shell in container
```

### Environment Variables

- `PORT` / `-port`: gRPC listen port (default 50051)
- `DATABASE_URL` or `PG*` split vars: Postgres connection
- `APPLY_MIGRATIONS=1`: Enable GORM auto-migrate on startup

### Docker Compose

Single service defined: `db` (Postgres 16 Alpine). Init scripts in `docker/db/init/` run on first startup. Use `make db-reset` to wipe volume and restart.

## Code Style & Patterns

### Import Aliases

Use `m` for models: `import m "github.com/stanterprise/observer/pkg/models"`

### Logger Nil-Safety

All functions accepting `*slog.Logger` must handle nil:

```go
if logger == nil {
    logger = slog.New(slog.NewTextHandler(&noopWriter{}, nil))
}
```

### Metadata Conversion

Protobuf `map[string]string` → GORM `datatypes.JSONMap` (map[string]any):

```go
md := map[string]any{}
for k, v := range protoMetadata {
    md[k] = v
}
```

## Component Separation Strategy

### Current State → Target Architecture

**Now**: Monolithic `pkg/server/server.go` that does gRPC ingestion + direct DB writes.

**Target**: Three separate components per `docs/architecture/01-components.md`:

1. **Ingestion Gateway** - gRPC → NATS publisher (no DB dependency)
2. **Processor** - NATS consumer → DB + object storage
3. **API** - HTTP/GraphQL server reading from DB and WebSocket for UI

### Implementation Phases

#### Phase 1: Extract NATS Publisher

- Create `pkg/publisher/nats.go` wrapping NATS JetStream client
- Add `NATS_URL` env var support (default: `nats://localhost:4222`)
- Keep DB writes in `pkg/server/server.go` temporarily (dual-write for safety)
- Pattern: gRPC handler validates → publishes to NATS → writes DB (if configured)

#### Phase 2: Create Processor Service

- New entrypoint: `cmd/processor/main.go`
- Subscribe to `tests.events.v1` stream
- Implement same DB persistence logic from current `pkg/server/server.go`
- Use NATS consumer groups for horizontal scaling
- Reuse existing `pkg/models/` and `pkg/database/` packages

#### Phase 3: Remove DB from Ingestion

- Remove `db` parameter from `RegisterServices()`
- Delete DB persistence logic from `pkg/server/server.go`
- Ingestion becomes stateless (scales horizontally without coordination)

#### Phase 4: Add API Service

- New entrypoint: `cmd/api/main.go`
- HTTP server with GraphQL endpoint (consider gqlgen)
- Read-only DB access via `pkg/database/`
- Serve static UI assets from `web/dist/`

### Docker Compose Profiles

Per `docs/architecture/04-docker-compose.md`, support two profiles:

**Profile: aio** (development)

```yaml
services:
  aio:
    build: .
    environment:
      - MODE=aio
      - DB_DRIVER=sqlite
      - STORAGE_DRIVER=local
```

**Profile: dist** (production)

```yaml
services:
  nats:
    image: nats:2.10-alpine
    command: ["-js"]

  ingestion:
    build: .
    environment:
      - MODE=service
      - SERVICE_TYPE=ingestion
      - NATS_URL=nats://nats:4222

  processor:
    build: .
    environment:
      - MODE=service
      - SERVICE_TYPE=processor
      - NATS_URL=nats://nats:4222
      - DATABASE_URL=postgres://...

  api:
    build: .
    environment:
      - MODE=service
      - SERVICE_TYPE=api
      - DATABASE_URL=postgres://...
```

### Multi-Stage Dockerfile

Per `docs/architecture/05-dockerfile.md`, use single Dockerfile with runtime switching:

```dockerfile
# Builder stage
FROM golang:1.23-alpine AS builder
WORKDIR /build
COPY go.* ./
RUN go mod download
COPY . .
RUN go build -o /bin/observer ./server
RUN go build -o /bin/processor ./cmd/processor
RUN go build -o /bin/api ./cmd/api

# Runtime stage
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates sqlite3
COPY --from=builder /bin/observer /bin/processor /bin/api /usr/local/bin/
COPY docker/entrypoint.sh /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
```

Entrypoint script routes by `SERVICE_TYPE` env var:

- `ingestion` → exec `/usr/local/bin/observer`
- `processor` → exec `/usr/local/bin/processor`
- `api` → exec `/usr/local/bin/api`
- `aio` → use s6-overlay to run all three + embedded NATS

### NATS Integration Patterns

**Publisher (Ingestion)**:

```go
js, _ := nats.Connect(natsURL)
js.Publish("tests.events.v1", eventBytes)
```

**Consumer (Processor)**:

```go
sub, _ := js.PullSubscribe("tests.events.v1", "processors")
for {
    msgs, _ := sub.Fetch(10)
    for _, m := range msgs {
        // Persist to DB, handle idempotency via upsert
        m.Ack()
    }
}
```

Use **JetStream** (not core NATS) for durability and replay. Configure stream retention and consumer delivery policies in NATS setup.

### Migration Checklist

When separating components:

- [ ] Preserve existing `bufconn` tests (ingestion only, no DB)
- [ ] Add NATS integration tests (use nats-server in-process or testcontainers)
- [ ] Update `Makefile` with targets: `build-ingestion`, `build-processor`, `build-api`
- [ ] Add `docker-compose.yml` profiles per architecture docs
- [ ] Document environment variables in README
- [ ] Keep backward compatibility: allow `observer` binary to run in legacy mode (direct DB writes) if `NATS_URL` not set

## Common Tasks

**Add new gRPC method**:

1. Update protobuf in `stanterprise/proto-go` repo
2. Run `make proto` to regenerate stubs
3. Implement handler in `pkg/server/server.go` following validation → log → persist pattern
4. Add bufconn test in `pkg/server/server_test.go`

**Add DB migration**:

1. Update GORM model in `pkg/models/models.go`
2. Test with `APPLY_MIGRATIONS=1 make run-dev`
3. For production, consider switching to schema migration tool (Goose, golang-migrate)

**Debug DB connection**:
Run `make env-print` to see resolved DSN values. Check GORM logs (configured at `logger.Warn` level, logs slow queries >200ms).
