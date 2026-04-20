# Processor Service

The processor service consumes test events from NATS JetStream and persists relational run data to PostgreSQL. MongoDB is retained only for the live step buffer used while tests are still in flight.

## Architecture

The processor service handles:

1. Consuming test events from NATS JetStream
2. Writing structured run, suite, test, attempt, and attachment data to PostgreSQL
3. Maintaining a MongoDB-backed live step buffer until test completion
4. (Future) Uploading artifacts to object store
5. (Future) Emitting summaries for fast UI queries

## Current State

Currently, the processor service runs as a NATS consumer with PostgreSQL persistence and MongoDB-backed live buffering. It requires NATS, PostgreSQL, and MongoDB.

## Running

### With PostgreSQL + MongoDB + NATS (required)

Start infrastructure:

```bash
docker compose up -d postgres mongodb nats
```

Then run the processor:

```bash
make build-processor
DATABASE_URL='postgres://observer:observer@localhost:5432/observer?sslmode=disable' \
MONGODB_URI='mongodb://root:password@localhost:27017/observer?authSource=admin' \
NATS_URL='nats://localhost:4222' \
./bin/processor
```

## Environment Variables

| Variable                                                                                          | Default        | Description                                                   |
| ------------------------------------------------------------------------------------------------- | -------------- | ------------------------------------------------------------- |
| `DATABASE_URL` or `POSTGRES_DSN`                                                                  | -              | PostgreSQL connection string (required for relational writes) |
| `MONGODB_URI` or `MONGO_URI`                                                                      | -              | MongoDB connection string (required for live step buffering)  |
| `MONGO_HOST`, `MONGO_PORT`, `MONGO_USER`, `MONGO_PASSWORD`, `MONGO_DATABASE`, `MONGO_AUTH_SOURCE` | -              | Split MongoDB vars (alternative to `MONGODB_URI`)             |
| `NATS_URL`                                                                                        | -              | NATS server URL (required)                                    |
| `NATS_STREAM`                                                                                     | `tests_events` | JetStream stream name                                         |
| `NATS_CONSUMER`                                                                                   | `processor`    | Durable consumer name                                         |

## Database

The processor writes relational data through the PostgreSQL repository. MongoDB is used only for the live step buffer collection.

## Future Enhancements

- [ ] Object storage integration (MinIO/S3)
- [ ] Summary generation and caching
- [ ] Consumer group scaling
- [ ] Metrics and tracing
