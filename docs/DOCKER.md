# Observer Docker Deployment Guide

This guide explains how to build and run Observer using Docker and Docker Compose.

## Quick Start

### Build Docker Image

```bash
# Build all binaries and Docker image
make docker-build

# Or build explicitly
make build-all
docker build -t observer:latest .
```

### Run with Docker Compose

#### AIO Mode (All-in-One)

Single container with embedded NATS, ingestion, processor, and API:

```bash
# Start AIO mode
docker compose --profile aio up -d

# View logs
docker compose --profile aio logs -f

# Stop
docker compose --profile aio down
```

**Ports:**
- `50051` - gRPC ingestion endpoint
- `8080` - HTTP API and UI
- `4222` - NATS client connections
- `8222` - NATS monitoring

**Note:** AIO mode currently requires SQLite support which is not yet implemented in the database layer. The processor service will fail to start until SQLite support is added to `internal/database/`.

#### Distributed Mode

Multi-container deployment with separate services:

```bash
# Start distributed mode
docker compose --profile dist up -d

# View logs
docker compose --profile dist logs -f

# Stop
docker compose --profile dist down
```

**Services:**
- `ingestion` - gRPC endpoint (port 50051)
- `processor` - NATS consumer and database writer
- `api` - HTTP API and UI (port 8080)
- `nats` - Message broker (ports 4222, 8222)
- `db` - PostgreSQL database (port 5432)

## Configuration

### Environment Variables

All services support these environment variables:

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `MODE` | Deployment mode: `aio` or `service` | `service` | Yes |
| `SERVICE_TYPE` | Service to run: `ingestion`, `processor`, `api`, `observer` | `ingestion` | When `MODE=service` |
| `PORT` | Service listen port | `50051` (ingestion), `8080` (api) | No |
| `DATABASE_URL` | Database connection URL | - | For processor, api |
| `NATS_URL` | NATS server URL | `nats://localhost:4222` | For ingestion, processor |
| `NATS_STREAM` | NATS stream name | `tests_events` | No |
| `APPLY_MIGRATIONS` | Run database migrations on startup | - | No |

### Distributed Mode Configuration

#### Ingestion Service

```yaml
environment:
  MODE: service
  SERVICE_TYPE: ingestion
  NATS_URL: nats://nats:4222
  NATS_STREAM: tests_events
```

#### Processor Service

```yaml
environment:
  MODE: service
  SERVICE_TYPE: processor
  NATS_URL: nats://nats:4222
  DATABASE_URL: postgres://user:pass@db:5432/observer?sslmode=disable
  APPLY_MIGRATIONS: "1"
```

#### API Service

```yaml
environment:
  MODE: service
  SERVICE_TYPE: api
  DATABASE_URL: postgres://user:pass@db:5432/observer?sslmode=disable
```

## Architecture

### AIO Mode

```
┌─────────────────────────────────────────┐
│           Observer AIO Container        │
│  ┌─────────┐  ┌──────────┐  ┌───────┐  │
│  │  NATS   │  │Ingestion │  │  API  │  │
│  │ Server  │  │          │  │       │  │
│  └────┬────┘  └─────┬────┘  └───────┘  │
│       │            │                    │
│       └────────────┼────────────────┐   │
│                    │                │   │
│               ┌────▼────┐      ┌────▼──┐│
│               │Processor│      │SQLite ││
│               └─────────┘      └───────┘│
└─────────────────────────────────────────┘
```

### Distributed Mode

```
┌───────────┐    ┌───────────┐    ┌──────────┐
│ Ingestion │───▶│   NATS    │◀───│Processor │
│  :50051   │    │ :4222     │    │          │
└───────────┘    └───────────┘    └─────┬────┘
                                         │
                                         ▼
                                  ┌──────────┐
                                  │Postgres  │
                                  │  :5432   │
                                  └─────┬────┘
                                        │
                                        ▼
                                  ┌──────────┐
                                  │   API    │
                                  │  :8080   │
                                  └──────────┘
```

## Building from Source

The Dockerfile uses a multi-stage build:

1. **Builder stage:** Compiles Go binaries
2. **Runtime stage:** Debian slim with s6-overlay, NATS server, and binaries

For environments with restricted network access, pre-build the binaries:

```bash
make build-all
docker build -t observer .
```

## Healthchecks

All services include healthchecks:

- **AIO**: Checks NATS monitoring endpoint
- **Ingestion**: Checks if process is running
- **Processor**: Checks if process is running
- **API**: Checks `/health` endpoint
- **NATS**: Checks `/healthz` endpoint
- **Postgres**: Checks `pg_isready`

## Makefile Targets

```bash
make docker-build       # Build Docker image
make docker-build-aio   # Build and tag for AIO mode
make docker-build-dist  # Build and tag for distributed mode
make docker-up-aio      # Start AIO profile
make docker-up-dist     # Start distributed profile
make docker-down        # Stop all services
```

## Troubleshooting

### Processor fails in AIO mode

The processor requires a database connection. AIO mode is designed to use SQLite, but SQLite support is not yet implemented in `internal/database/database.go`. Until then, use distributed mode with PostgreSQL.

### Services can't connect to NATS

Ensure NATS is healthy before starting dependent services. Docker Compose handles this automatically with `depends_on` and health checks.

### Port conflicts

If ports are already in use, set custom ports in `.env`:

```bash
INGESTION_PORT=50052
API_PORT=8081
NATS_PORT=4223
NATS_HTTP_PORT=8223
```

## Development

For local development, use the development database:

```bash
# Start just the database
make db-up

# Run services locally
DATABASE_URL='postgres://postgres:postgres@localhost:5432/observer?sslmode=disable' \
  ./bin/ingestion &

DATABASE_URL='postgres://postgres:postgres@localhost:5432/observer?sslmode=disable' \
  APPLY_MIGRATIONS=1 \
  ./bin/processor &

DATABASE_URL='postgres://postgres:postgres@localhost:5432/observer?sslmode=disable' \
  ./bin/api &
```

## Next Steps

- [ ] Implement SQLite support for AIO mode
- [ ] Add MinIO for artifact storage in distributed mode
- [ ] Add Helm charts for Kubernetes deployment
- [ ] Add multi-arch builds (amd64, arm64)
- [ ] Add container image scanning in CI
