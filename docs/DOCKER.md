# Observer Docker Deployment Guide

This guide explains how to build and run Observer using Docker and Docker Compose.

> **⚡ Performance Note**: For optimized multi-architecture builds, see [Build Optimization Guide](BUILD_OPTIMIZATION.md)

## Docker Images

Observer provides separate Docker images for each deployment scenario:

- **`Dockerfile.aio`** - All-in-one image with embedded NATS and all services
- **`Dockerfile.ingestion`** - Ingestion service only
- **`Dockerfile.processor`** - Processor service only
- **`Dockerfile.api`** - API service only

## Quick Start

### Build Docker Images

```bash
# Build all images (standard build)
make docker-build-all

# Or build individual images
make docker-build-aio        # AIO image
make docker-build-ingestion  # Ingestion service
make docker-build-processor  # Processor service
make docker-build-api        # API service

# For faster multi-platform builds with caching:
make docker-buildx-setup     # One-time setup
make docker-buildx-aio       # Optimized multi-arch build (60-90% faster)
```

**Build Performance:**

- Standard build: ~20 min for ARM64, ~15 min for AMD64
- Optimized buildx (first): ~8 min for both architectures
- Optimized buildx (cached): ~2 min for code changes

See [Build Optimization Guide](BUILD_OPTIMIZATION.md) for details.

### Run with Docker Compose

#### AIO Mode (All-in-One)

Single container with embedded NATS, ingestion, processor, and API:

```bash
# Build and start AIO mode
make docker-up-aio

# Or manually
docker compose --profile aio up -d

# View logs
docker compose --profile aio logs -f

# Stop
docker compose --profile aio down
```

**Ports:**

- `50051` - gRPC ingestion endpoint
- `8080` - HTTP API and UI
- `27017` - MongoDB
- `4222` - NATS client connections
- `8222` - NATS monitoring

#### Distributed Mode

Multi-container deployment with separate services:

```bash
# Build and start distributed mode
make docker-up-dist

# Or manually
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
- `mongodb` - MongoDB database (port 27017)

## Configuration

### Environment Variables

All services support these environment variables:

| Variable       | Description                                                 | Default                           | Required                 |
| -------------- | ----------------------------------------------------------- | --------------------------------- | ------------------------ |
| `MODE`         | Deployment mode: `aio` or `service`                         | `service`                         | Yes                      |
| `SERVICE_TYPE` | Service to run: `ingestion`, `processor`, `api`, `observer` | `ingestion`                       | When `MODE=service`      |
| `PORT`         | Service listen port                                         | `50051` (ingestion), `8080` (api) | No                       |
| `MONGODB_URI`  | MongoDB connection string                                   | -                                 | For processor, api       |
| `NATS_URL`     | NATS server URL                                             | `nats://localhost:4222`           | For ingestion, processor |
| `NATS_STREAM`  | NATS stream name                                            | `tests_events`                    | No                       |

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
  MONGODB_URI: mongodb://user:pass@mongodb:27017/observer?authSource=admin
```

#### API Service

```yaml
environment:
  MODE: service
  SERVICE_TYPE: api
  MONGODB_URI: mongodb://user:pass@mongodb:27017/observer?authSource=admin
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
│               │Processor│      │MongoDB ││
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
                                      │ MongoDB  │
                                      │ :27017   │
                                      └─────┬────┘
                                        │
                                        ▼
                                  ┌──────────┐
                                  │   API    │
                                  │  :8080   │
                                  └──────────┘
```

## Building from Source

Each service has its own Dockerfile:

- **`Dockerfile.aio`**: Includes s6-overlay, NATS server, and all service binaries
- **`Dockerfile.ingestion`**: Minimal image with only the ingestion binary
- **`Dockerfile.processor`**: Minimal image with only the processor binary
- **`Dockerfile.api`**: Minimal image with only the API binary

All Dockerfiles require pre-built binaries in the `bin/` directory:

```bash
# Build Go binaries
make build-all

# Build specific Docker image
docker build -f Dockerfile.aio -t observer:aio .
docker build -f Dockerfile.ingestion -t observer:ingestion .
docker build -f Dockerfile.processor -t observer:processor .
docker build -f Dockerfile.api -t observer:api .
```

## Healthchecks

All services include healthchecks:

- **AIO**: Checks NATS monitoring endpoint
- **Ingestion**: Checks if process is running
- **Processor**: Checks if process is running
- **API**: Checks `/health` endpoint
- **NATS**: Checks `/healthz` endpoint
- **MongoDB**: Checks `db.adminCommand({ ping: 1 })`

## Makefile Targets

```bash
make docker-build-all        # Build all Docker images
make docker-build-aio        # Build AIO image
make docker-build-ingestion  # Build ingestion image
make docker-build-processor  # Build processor image
make docker-build-api        # Build API image
make docker-up-aio           # Start AIO profile
make docker-up-dist          # Start distributed profile
make docker-down             # Stop all services
```

## Troubleshooting

### Processor fails in AIO mode

The processor requires a MongoDB connection. Ensure `MONGODB_URI` is set correctly for your deployment mode.

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
make mongo-up

# Run services locally
NATS_URL='nats://localhost:4222' \
  ./bin/ingestion &

MONGODB_URI='mongodb://root:change-me@localhost:27017/observer?authSource=admin' \
  NATS_URL='nats://localhost:4222' \
  ./bin/processor &

MONGODB_URI='mongodb://root:change-me@localhost:27017/observer?authSource=admin' \
  NATS_URL='nats://localhost:4222' \
  ./bin/api &
```

## Next Steps

- [ ] Add MinIO for artifact storage in distributed mode
- [ ] Add multi-arch builds (amd64, arm64)
- [ ] Add container image scanning in CI
