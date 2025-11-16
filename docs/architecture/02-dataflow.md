# Data Flow

```mermaid
flowchart TD
  A[Reporter Plugin] -->|gRPC| B[Ingestion Server]
  B -->|Publish| C[NATS JetStream]
  C --> D[Processor Service]
  C --> I[WebSocket Consumer]
  D --> E[(Database)]
  D --> F[(Object Storage)]
  E --> G[API Service]
  F --> G
  I --> G
  G -->|HTTP/GraphQL| H[Web UI]
  G -->|WebSocket /ws| H
```

---

## Event Lifecycle

1. Reporter sends events over gRPC (`TestStarted`, `Step`, `AttachmentAdded`, etc.).
2. Ingestion publishes validated events to NATS.
3. Processor consumes, persists data and artifacts.
4. **WebSocket consumer (part of API service) relays events to connected web clients in real-time.**
5. Processor emits summaries → API caches or indexes them.
6. UI displays data via the API and receives real-time updates via WebSocket.

---

## Observability & Reliability

- Backpressure managed via NATS JetStream.
- DLQ (dead letter queue) for failed events.
- OpenTelemetry spans across all services.
- Prometheus metrics on `/metrics` endpoints.
