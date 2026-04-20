# Deployment Modes

## 🧩 All-in-One (AIO)

A single container with multiple internal services managed via **s6-overlay**.

### Includes

- `ingestion`, `processor`, and `api` services managed by `s6-overlay`
- Embedded `nats-server`
- Embedded MongoDB
- Embedded PostgreSQL
- Nginx-served Web UI
- Local artifact store

### Ports

- `80` → Web UI
- `50051` → gRPC ingestion
- `8080` → Internal API
- `4222` → NATS
- `8222` → NATS monitoring
- `27017` → Internal MongoDB
- `5432` → Internal PostgreSQL, optionally published by Compose as `AIO_POSTGRES_PORT`

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
- `mongodb` (live step buffer DB)
- `postgres` (relational run storage)

### Scaling

- Processors scale horizontally (consumer groups).
- API scaled behind a load balancer.
- NATS clustered independently.
