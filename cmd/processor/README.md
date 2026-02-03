# Processor Service

The processor service consumes test events from NATS JetStream and persists them to MongoDB. It's responsible for durable storage of test runs.

## Architecture

The processor service handles:

1. Consuming test events from NATS JetStream
2. Writing structured data to MongoDB with idempotent upserts
3. (Future) Uploading artifacts to object store
4. (Future) Emitting summaries for fast UI queries

## Current State

Currently, the processor service runs as a NATS consumer with MongoDB persistence. It requires both NATS and MongoDB.

## Running

### With MongoDB + NATS (required)

Start infrastructure:

```bash
make mongo-up nats-up
```

Then run the processor:

```bash
make build-processor
MONGODB_URI='mongodb://root:change-me@localhost:27017/observer?authSource=admin' \
NATS_URL='nats://localhost:4222' \
./bin/processor
```

## Environment Variables

| Variable                                                                                          | Default        | Description                                       |
| ------------------------------------------------------------------------------------------------- | -------------- | ------------------------------------------------- |
| `MONGODB_URI` or `MONGO_URI`                                                                      | -              | MongoDB connection string (required)              |
| `MONGO_HOST`, `MONGO_PORT`, `MONGO_USER`, `MONGO_PASSWORD`, `MONGO_DATABASE`, `MONGO_AUTH_SOURCE` | -              | Split MongoDB vars (alternative to `MONGODB_URI`) |
| `NATS_URL`                                                                                        | -              | NATS server URL (required)                        |
| `NATS_STREAM`                                                                                     | `tests_events` | JetStream stream name                             |
| `NATS_CONSUMER`                                                                                   | `processor`    | Durable consumer name                             |

## Database

The processor uses MongoDB and does not run SQL migrations.

## Future Enhancements

- [ ] Object storage integration (MinIO/S3)
- [ ] Summary generation and caching
- [ ] Consumer group scaling
- [ ] Metrics and tracing
