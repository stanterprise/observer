# Observer Service - AI Agent Instructions

## Purpose

This file is for high-impact coding guidance only.

Keep this file focused on:

- Architecture boundaries
- Data-layer rules
- Reliability and safety constraints
- Coding/testing conventions that prevent regressions

Avoid duplicating full runbooks that already exist in service READMEs.

## Architecture Overview

Observer is a test observability system that ingests test execution events via gRPC and processes them through NATS JetStream.

### Deployment Modes

- All-in-One (AIO): single container for local/dev workflows
- Distributed: multi-container deployment for CI/production

### High-Level Flow

```text
Test Reporter -> Ingestion (gRPC) -> NATS JetStream ---> Processor -> PostgreSQL (authoritative)
                                                  \--> API WS relay -> Web UI

                                      MongoDB (secondary): live step buffer only
                                      API Service reads from PostgreSQL
```

### Components

1. Ingestion Service (`cmd/ingestion/`)
   - Stateless gRPC endpoint
   - Validates events and publishes to NATS
   - No direct persistence

2. Processor Service (`cmd/processor/`)
   - NATS JetStream consumer
   - Writes relational execution data to PostgreSQL via `internal/repository/postgres/`
   - Uses MongoDB only for live in-flight step buffering (`live_step_buffers`)

3. API Service (`cmd/api/`)
   - REST + WebSocket
   - PostgreSQL is required for REST API queries
   - Optional NATS subscription for WebSocket event relay

4. Web UI (`web/`)
   - React + TypeScript + Tailwind CSS
   - Consumes REST and WebSocket endpoints

## Data Layer Authority

### Primary Data Store: PostgreSQL

PostgreSQL is the source of truth for persisted run data and API reads.

Use these locations for relational work:

- Connection: `internal/database/postgres.go`
- Relational models: `internal/models/relational.go`
- Repository: `internal/repository/postgres/`
- REST handlers: `pkg/api/rest_postgres.go`, `pkg/api/attachments_postgres.go`
- SQL migrations: `migrations/*.sql` via `cmd/migrate/`

### Secondary Data Store: MongoDB (Limited Scope)

MongoDB is retained for limited transient workflows, primarily active step buffering.

Use these locations only for that scope:

- Connection/index setup: `internal/database/mongodb.go`
- Repository: `internal/repository/mongodb/`
- Main collection: `live_step_buffers`

Do not introduce new durable read models in MongoDB unless explicitly requested.

### Data-Path Rules

- API read/query surfaces should target PostgreSQL.
- Durable persistence logic should target PostgreSQL.
- MongoDB updates should stay scoped to transient live-buffer behavior.
- If behavior differs between stores, treat PostgreSQL as canonical.

## Database Connection Conventions

### PostgreSQL

`ConnectPostgresFromEnv()` reads `POSTGRES_DSN` first, then `DATABASE_URL`.

```go
pgDB, err := database.ConnectPostgresFromEnv(logger)
if err != nil {
    logger.Error("postgres connect failed", "error", err)
    os.Exit(1)
}
if pgDB == nil {
    logger.Error("POSTGRES_DSN / DATABASE_URL not set")
    os.Exit(1)
}
defer pgDB.Close()
```

Service expectations:

- API: PostgreSQL required
- Processor: PostgreSQL is primary target; current code allows nil for compatibility, but production should run with PostgreSQL configured

### MongoDB

`ConnectMongoDBFromEnv()` reads `MONGODB_URI` or `MONGO_URI` (or split vars).

Use MongoDB only where the code path is explicitly for step buffer workflows.

## Migrations and Schema Changes

PostgreSQL migrations are mandatory for schema changes. When you modify GORM models in `internal/models/relational.go`, you must create corresponding SQL migrations.

### Migration Workflow

1. **Modify GORM Model**: Update struct in `internal/models/relational.go`
   - Add/remove fields, change types, update tags
   - Example: Add a new nullable field `RawMetadata` to `TestRun` struct

2. **Create Migration Files** in `migrations/` folder:
   - File format: `000NNN_description.{up,down}.sql`
   - Example: `000003_add_raw_metadata_to_run.up.sql` and `.down.sql`
   - Increment the number; check existing migrations for the next available number

3. **Up Migration (forward)**: Add your schema changes in `.up.sql`
   ```sql
   -- migrations/000003_add_raw_metadata_to_run.up.sql
   ALTER TABLE test_runs ADD COLUMN raw_metadata JSONB;
   CREATE INDEX idx_test_runs_raw_metadata ON test_runs USING GIN (raw_metadata);
   ```

4. **Down Migration (rollback)**: Reverse the changes in `.down.sql`
   ```sql
   -- migrations/000003_add_raw_metadata_to_run.down.sql
   DROP INDEX IF EXISTS idx_test_runs_raw_metadata;
   ALTER TABLE test_runs DROP COLUMN IF EXISTS raw_metadata;
   ```

5. **Test Migrations Locally**:
   - Spin up PostgreSQL: `make nats-up` (or use `docker compose up -d`; ensure `POSTGRES_DSN` is set)
   - Run migration forward: `cmd/migrate` binary or custom migration runner
   - Verify schema change via `psql` or migration state logs
   - Test rollback: Revert and re-apply to confirm `.down.sql` works
   - Test backward compatibility: Ensure old code can still connect to new schema

6. **Guidelines**:
   - Prefer additive, backward-compatible changes (adding columns, creating new tables)
   - Avoid destructive changes (drop column/table) unless migration/rollback plan is explicit
   - Use `IF EXISTS` / `IF NOT EXISTS` clauses for safety
   - Index frequently queried fields (e.g., run_id, suite_id)
   - Avoid large data transformations in migrations; use processor/API to backfill if needed

7. **Deployment**:
   - Migrations run automatically at service startup via connection initialization or explicit `cmd/migrate` call
   - CI/deployment pipelines validate migrations before service restart
   - Monitor migration logs for errors; rollback to previous migration on failure

### MongoDB Changes

MongoDB changes should be minimal and scoped to transient buffer documents/indexes only. Do not create durable schema or read models in MongoDB.

## NATS Integration Patterns

### Publisher

Located in `pkg/publisher/nats.go`.

- Ingestion validates gRPC request
- Publishes event envelope to JetStream

### Consumers

Processor consumes events and routes them by type.

- Persist durable execution state in PostgreSQL
- Use MongoDB only for in-flight step buffering
- Acknowledge unknown or irrelevant events to avoid redelivery loops

## gRPC Service Conventions

### Interceptor Chain

Use `pkg/server/NewGRPCServer()` chain order:

1. recovery interceptor
2. logging interceptor

### Validation Pattern

Return `InvalidArgument` for client input errors.

```go
if in == nil || in.TestCase == nil {
    return nil, status.Error(codes.InvalidArgument, "test_case is required")
}
```

### Error Handling

- Client errors: `codes.InvalidArgument`
- Internal failures (DB/NATS/etc): `codes.Internal`
- Log with context; do not leak sensitive internals

## Testing Strategy

### Backend

- Use `bufconn` for gRPC unit tests (no TCP port flakiness)
- Add/maintain repository tests for PostgreSQL paths (`internal/repository/postgres/*_test.go`)
- Use NATS integration tests for publish/consume behavior

### Frontend

For UI-affecting changes under `web/src/**`, run:

```bash
cd web && npm run build
```

Verify:

1. No TypeScript errors
2. No unintended fallback fonts
3. Clear tonal contrast in applicable themes

## Build and Run Guidance (Condensed)

Use root-level Make targets and service READMEs for detailed commands.

- Build all: `make build-all`
- Run tests: `make test`
- NATS integration: `make test-nats-integration`

For service-specific runtime env examples, prefer:

- `cmd/api/README.md`
- `cmd/processor/README.md`
- `README.md`

## Environment Variables (Important)

### Ingestion

- `PORT` (default `50051`)
- `NATS_URL`
- `NATS_STREAM`
- `NATS_SUBJECT_PREFIX`

### Processor

- `NATS_URL`, `NATS_STREAM`, `NATS_CONSUMER`
- `POSTGRES_DSN` or `DATABASE_URL` (primary persistence path)
- `MONGODB_URI` or `MONGO_URI` (live step buffer path)

### API

- `PORT` (default `8080`)
- `POSTGRES_DSN` or `DATABASE_URL` (required)
- `NATS_URL` (optional, for WebSocket relay)
- `NATS_STREAM`, `NATS_WS_CONSUMER`
- `CORS_ALLOWED_ORIGINS`

## Code Style and Patterns

### Logger Nil-Safety

Library-style constructors/functions that accept `*slog.Logger` should handle nil safely.

```go
if logger == nil {
    logger = slog.Default()
}
```

### Import Aliases

Use `m` for models imports where already established:

```go
import m "github.com/stanterprise/observer/internal/models"
```

### Graceful Shutdown Pattern

Use signal handling with bounded shutdown timeout in service entrypoints.

```go
sigCh := make(chan os.Signal, 1)
signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
```

## Frontend Implementation Guardrails (Stitch)

When editing UI under `web/src/**`, treat Stitch guidance as source-of-truth:

- Tokens/usage: `web/src/lib/styleGuide.ts`
- Policy: `web/STYLE_GUIDE.md`
- Fonts/variables: `web/src/index.css`

Follow existing project visual language and accessibility requirements.

## Common Tasks

### Add New gRPC Method

1. Update protobuf in `stanterprise/proto-go`
2. Bump dependency in `go.mod`
3. Implement ingestion handler in `pkg/server/server.go`
4. Add/update processor event handling in `pkg/consumer/`
5. Add `bufconn` tests in `tests/`

### Add Database Change

1. Add SQL migration(s) under `migrations/`
2. Update relational models/repository (`internal/models/relational.go`, `internal/repository/postgres/`)
3. Update API response/query behavior in `pkg/api/rest_postgres.go`
4. Add/adjust tests for migration + repository + API paths

### Debug PostgreSQL Connection

- Verify DSN env (`POSTGRES_DSN` or `DATABASE_URL`)
- Confirm API startup logs for postgres connect success/failure
- Use migration tool (`cmd/migrate`) to verify schema state

### Debug MongoDB Buffer Path

- Verify Mongo env (`MONGODB_URI` / `MONGO_URI`)
- Confirm `live_step_buffers` indexes are created at startup
- Scope debugging to active step buffering only

### Debug NATS

- Check NATS logs and stream/consumer state
- Validate ack/NAK behavior and DLQ routing where configured
