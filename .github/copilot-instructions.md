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

## Future Architecture (Not Yet Implemented)

The `docs/architecture/` folder describes a distributed event-driven system with:

- NATS JetStream for event streaming
- Separate processor service for DB persistence
- REST/GraphQL API + Web UI
- Artifact storage (S3/MinIO)

**Current scope**: This repo only implements the gRPC ingestion gateway. When adding processor/API components, follow the separation described in `docs/architecture/01-components.md`.

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
