# Deployment Modes

User-facing installation steps are maintained in the [installation guide](https://observer.stanterprise.com/docs/install/). This page describes deployment topology for implementation and operations.

## 🧩 All-in-One (AIO)

The Docker AIO image runs multiple internal services managed via **s6-overlay**.

### Includes

- `ingestion`, `processor`, and `api` services managed by `s6-overlay`
- Embedded `nats-server`
- Embedded MongoDB for live-step buffering
- Embedded PostgreSQL for durable run data
- Nginx-served Web UI
- Local artifact store

### Ports

- `80` → Web UI / Nginx (proxies `/api/` and `/ws` to the internal API)
- `50051` → gRPC ingestion
- `8080` → Internal API (also publishable through the Compose AIO profile)
- `4222` → NATS
- `8222` → NATS monitoring
- `27017` → Internal MongoDB
- `5432` → Internal PostgreSQL, optionally published by Compose as `AIO_POSTGRES_PORT`

### Use Case

Ideal for local development, demos, or single-node CI. PostgreSQL stores durable run data; MongoDB is limited to the live-step buffer.

---

## ⚙️ Distributed Mode

Each component runs as an independent container or Kubernetes Deployment.

### Components

- `ingestion` (gRPC entrypoint and NATS publisher)
- `processor` (event consumer and durable PostgreSQL writer)
- `api` (REST API and optional WebSocket relay)
- `web` (separate web UI service)
- `nats` (message broker)
- `postgres` (canonical durable run storage)
- `mongodb` (transient live-step buffer only)

Dependencies may be embedded or external according to the Compose profile or Helm values.

### Scaling

- Processors scale horizontally through durable consumer work.
- API and web services can run behind downstream load balancers.
- NATS can be clustered independently.
- PostgreSQL remains the source of truth for durable run data.

## Helm AIO distinction

Helm AIO is a chart mode, not the same dependency layout as the Docker AIO image. Shipped Helm AIO presets disable the PostgreSQL, MongoDB, and NATS dependency subcharts. See the [Helm chart README](../../charts/observer/README.md) for current chart values and external dependency behavior.
