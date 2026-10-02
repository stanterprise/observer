# Core Components

## 1. Ingestion Gateway

- Accepts gRPC calls from reporters.
- Validates protobuf payloads.
- Publishes messages to NATS (`tests.events.v1`).
- Handles backpressure and transient errors.
- Durable run data is persisted by the processor to PostgreSQL; MongoDB is limited to live-step buffering.

## 2. Event Router / Stream

- Default: **NATS JetStream** (lightweight, simple).
- Alternative: Kafka for higher scale (planned; not implemented).
- Topics:
	- `tests.events.v1` (current)
	- `tests.summaries.v1` (planned/design topic)
	- `tests.errors.v1` (planned/design topic)

## 3. Processor / Indexer

- Consumes test events.
- Writes structured data to PostgreSQL.
- Uses MongoDB for `live_step_buffers` live-step buffering.
- Uploads artifacts through the configured artifact storage driver.
- Summary emission for fast UI queries remains a design goal, not a separate current stream.

## 4. Databases

| Mode        | Engine               | Notes                                                                   |
| ----------- | -------------------- | ----------------------------------------------------------------------- |
| AIO         | MongoDB + PostgreSQL | Embedded MongoDB live buffer plus embedded relational DB; PostgreSQL is canonical for durable runs. |
| Distributed | MongoDB + PostgreSQL | Embedded or external services; MongoDB buffers live steps and PostgreSQL stores durable runs. |

## 5. Artifact Storage

| Mode        | Storage             | Path                                    |
| ----------- | ------------------- | --------------------------------------- |
| AIO         | Local FS            | `/data/artifacts`                       |
| Distributed | MinIO / S3-compatible | Configurable bucket through the storage driver. |

## 6. API / GraphQL

- Serves the UI and external integrations through REST.
- **WebSocket endpoint (`/ws`) for real-time event streaming** ✅
- NATS JetStream consumer for event relay to connected clients.
- Provides authentication middleware (future; not implemented).
- Exposes `/api/graphql` and `/metrics` endpoints (planned; not registered by the current API).
- Multiple WebSocket clients supported with automatic connection management.

## 7. Web UI

- Built with React + Tailwind + shadcn/ui.
- Displays runs, tests, steps, and artifacts.
- Receives persisted data through REST and live updates through WebSocket.

## 8. Auth Layer

| Mode        | Method                                  |
| ----------- | --------------------------------------- |
| AIO         | Single dev token (planned; not implemented) |
| Distributed | OIDC (GitHub, Okta, etc.; planned, not implemented) |
