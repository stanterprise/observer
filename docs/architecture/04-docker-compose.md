# Docker Compose Architecture

## Profiles

- `aio` → Single all-in-one container with embedded MongoDB, PostgreSQL, and NATS.
- `dist` → Multi-container distributed stack.

### Usage

```bash
docker compose --profile aio up -d
docker compose --profile dist up -d
```

### Services

| Service       | Description |
| ------------- | ----------- |
| `aio`         | Bundled application, broker, PostgreSQL, MongoDB, and web UI |
| `nats`        | NATS JetStream event transport |
| `postgres`    | Authoritative durable run-data database |
| `mongodb`     | Transient live in-flight step buffer only |
| `ingestion`   | gRPC event receiver and NATS publisher |
| `processor`   | Event consumer; persists durable data to PostgreSQL |
| `api`         | REST API and optional WebSocket event relay |
| `web`         | Separate UI container for distributed mode |

### AIO Published Ports

- `AIO_WEB_PORT` → Nginx / Web UI (`3000`)
- `AIO_GRPC_PORT` → gRPC ingestion (`50051`)
- `AIO_API_PORT` → REST API (`8080`)
- `AIO_NATS_PORT` → NATS client (`4222`)
- `AIO_NATS_HTTP_PORT` → NATS monitoring (`8222`)
- `AIO_POSTGRES_PORT` → PostgreSQL for local debugging (`5432`)

The `8080` API mapping is available in the Compose AIO profile. The common `docker run` quick start publishes the Nginx web port, which proxies `/api/` and `/ws` to the internal API. See the [public installation guide](https://observer.stanterprise.com/docs/install/) for end-user instructions.
