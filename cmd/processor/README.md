# Processor Service

The processor service consumes test events and persists them to the database. It's responsible for data persistence and artifact storage.

## Architecture

The processor service handles:

1. Consuming test events from NATS JetStream (future)
2. Writing structured data to database (Postgres/SQLite)
3. Uploading artifacts to object store (future)
4. Emitting summaries for fast UI queries (future)

## Current State

Currently, the processor service runs as a gRPC server with database persistence. It requires a database connection to operate.

## Running

### With database (required)

Start the database first:
```bash
make db-up
```

Then run the processor:
```bash
DATABASE_URL='postgres://postgres:postgres@localhost:5432/observer?sslmode=disable' ./bin/processor
# or
make build-processor
DATABASE_URL='postgres://postgres:postgres@localhost:5432/observer?sslmode=disable' ./bin/processor
```

Default port: `50052`

### Using split environment variables

```bash
PGHOST=localhost PGPORT=5432 PGUSER=postgres PGPASSWORD=postgres PGDATABASE=observer PGSSLMODE=disable ./bin/processor
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `50052` | gRPC listening port |
| `DATABASE_URL` | - | PostgreSQL connection string (required) |
| `PGHOST` | - | PostgreSQL host (alternative to DATABASE_URL) |
| `PGPORT` | `5432` | PostgreSQL port |
| `PGUSER` | - | PostgreSQL user |
| `PGPASSWORD` | - | PostgreSQL password |
| `PGDATABASE` | - | PostgreSQL database name |
| `PGSSLMODE` | `disable` | PostgreSQL SSL mode |
| `APPLY_MIGRATIONS` | - | Set to `1` to enable auto-migrations |
| `NATS_URL` | - | NATS server URL (future) |

## Database

The processor automatically applies schema migrations on startup when `APPLY_MIGRATIONS=1` is set.

Supported databases:
- PostgreSQL (distributed mode)
- SQLite (AIO mode, future)

## Future Enhancements

- [ ] NATS JetStream consumer integration
- [ ] Object storage integration (MinIO/S3)
- [ ] Summary generation and caching
- [ ] Consumer group scaling
- [ ] Metrics and tracing
