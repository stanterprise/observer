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

| Service     | Description                                                 |
| ----------- | ----------------------------------------------------------- |
| `aio`       | Single compact container with embedded databases and broker |
| `nats`      | Message broker                                              |
| `mongodb`   | Main database                                               |
| `ingestion` | gRPC endpoint                                               |
| `processor` | Event consumer                                              |
| `api`       | Web UI + API service                                        |

### AIO Published Ports

- `AIO_WEB_PORT` → Nginx / Web UI (`3000`)
- `AIO_GRPC_PORT` → gRPC ingestion (`50051`)
- `AIO_API_PORT` → REST API (`8080`)
- `AIO_NATS_PORT` → NATS client (`4222`)
- `AIO_NATS_HTTP_PORT` → NATS monitoring (`8222`)
- `AIO_POSTGRES_PORT` → PostgreSQL for local debugging (`5432`)
