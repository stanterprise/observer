# Deployment Modes

## 🧩 All-in-One (AIO)

A single container with multiple internal services managed via **s6-overlay**.

### Includes
- `observer` binary (ingestion + API)
- Embedded `nats-server`
- SQLite database
- Optional Meilisearch
- Local artifact store

### Ports
- `8080` → API & UI  
- `4222` → NATS  
- `4317` → OTLP tracing

### Use Case
Ideal for local development, demos, or single-node CI.

---

## ⚙️ Distributed Mode

Each component runs as an independent container or Kubernetes Deployment.

### Components
- `ingestion` (gRPC entrypoint)
- `processor` (event consumer)
- `api` (HTTP + UI)
- `nats` (message broker)
- `postgres` (DB)
- `minio` (object storage)
- `meilisearch` (optional)

### Scaling
- Processors scale horizontally (consumer groups).  
- API scaled behind a load balancer.  
- NATS clustered independently.
