# Core Components

## 1. Ingestion Gateway

- Accepts gRPC calls from reporters.
- Validates protobuf payloads.
- Publishes messages to NATS (`tests.events.v1`).
- Handles backpressure and transient errors.

## 2. Event Router / Stream

- Default: **NATS JetStream** (lightweight, simple).
- Alternative: Kafka for higher scale.
- Topics:
  - `tests.events.v1`
  - `tests.summaries.v1`
  - `tests.errors.v1`

## 3. Processor / Indexer

- Consumes test events.
- Writes structured data to database.
- Uploads artifacts to object store.
- Emits summaries for fast UI queries.

## 4. Databases

| Mode        | Engine  | Notes                                |
| ----------- | ------- | ------------------------------------ |
| AIO         | MongoDB | Embedded for local/dev               |
| Distributed | MongoDB | External service in distributed mode |

## 5. Artifact Storage

| Mode        | Storage    | Path                |
| ----------- | ---------- | ------------------- |
| AIO         | Local FS   | `/data/artifacts`   |
| Distributed | MinIO / S3 | Configurable bucket |

## 6. API / GraphQL

- Serves UI and external integrations.
- **WebSocket endpoint (`/ws`) for real-time event streaming** ✅
- NATS JetStream consumer for event relay to connected clients
- Provides authentication middleware (future).
- Exposes `/api/graphql` and `/metrics` endpoints (future).
- Multiple WebSocket clients supported with automatic connection management

## 7. Web UI

- Built with React + Tailwind + shadcn/ui.
- Displays runs, tests, steps, and artifacts.

## 8. Auth Layer

| Mode        | Method                    |
| ----------- | ------------------------- |
| AIO         | Single dev token          |
| Distributed | OIDC (GitHub, Okta, etc.) |
