# Observer Service

A test observability system that collects test execution events via gRPC. The system can operate in two modes:

- 🧩 **All-in-One (AIO)** — Single container with embedded services for local/dev use
- ⚙️ **Distributed Mode** — Multi-container deployment for production/CI

## Architecture

The Observer system is composed of three main components:

### 1. **Ingestion Service** (`cmd/ingestion`)
- gRPC endpoint for test event collection
- Stateless and horizontally scalable
- Validates protobuf payloads
- Future: Publishes to NATS JetStream

### 2. **Processor Service** (`cmd/processor`)
- Consumes events and persists to database
- Handles database migrations
- Future: NATS consumer, artifact storage, summary generation

### 3. **API Service** (`cmd/api`)
- HTTP/GraphQL API for web UI and integrations
- Read-only database access
- Future: WebSocket for real-time updates, authentication

See detailed documentation in each component's README:
- [Ingestion Service](./cmd/ingestion/README.md)
- [Processor Service](./cmd/processor/README.md)
- [API Service](./cmd/api/README.md)

## Quick Start

### Build All Components

```bash
make build-all
```

This builds:
- `bin/observer` - Legacy monolithic server
- `bin/ingestion` - Ingestion service
- `bin/processor` - Processor service
- `bin/api` - API service

### Run Individual Components

```bash
# Ingestion (stateless, no DB required)
./bin/ingestion

# API (optional DB connection)
./bin/api

# Processor (requires DB)
make db-up  # Start PostgreSQL
DATABASE_URL='postgres://postgres:postgres@localhost:5432/observer?sslmode=disable' ./bin/processor
```

### Run Legacy Monolithic Server

```bash
make run
# or with database
make run-dev
```

## Tests

```bash
make test
```

The test suite uses an in-process `bufconn` listener (no external ports) and validates argument handling.

## Make Targets

### Building
- `make build` – Build legacy monolithic server
- `make build-ingestion` – Build ingestion service
- `make build-processor` – Build processor service
- `make build-api` – Build API service
- `make build-all` – Build all components

### Running
- `make run` – Run legacy server (depends on build)
- `make run-dev` – Run with PostgreSQL database

### Database
- `make db-up` – Start PostgreSQL container
- `make db-down` – Stop containers and remove volumes
- `make db-psql` – Open psql against the database
- `make db-reset` – Reset database

### Testing & Quality
- `make test` – Run all tests
- `make test-race` – Run tests with race detector
- `make test-cover` – Run tests with coverage
- `make fmt` – Format code
- `make vet` – Vet code
- `make lint` – Run golangci-lint

### Tools
- `make proto` – Generate gRPC stubs
- `make tools` – Install dev tools

## Configuration

### Ingestion Service

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `50051` | gRPC listening port |

### Processor Service

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `50052` | gRPC listening port |
| `DATABASE_URL` | - | PostgreSQL connection string (required) |
| `APPLY_MIGRATIONS` | - | Set to `1` to enable auto-migrations |

### API Service

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP listening port |
| `DATABASE_URL` | - | PostgreSQL connection string (optional) |

## Logging

Uses Go 1.21+ `slog` with text handler. Interceptors log RPC method, duration, peer, status code, and errors. Panic recovery interceptor converts panics to `Internal` status and logs stack traces.

## Validation

Handlers validate presence of `TestId`. Missing / empty IDs return `InvalidArgument`.

## Migration from Monolithic to Distributed

The repository maintains backward compatibility with the monolithic `server/main.go` deployment while supporting the new distributed architecture:

1. **Legacy Mode**: Run `./bin/observer` for single-process deployment
2. **Distributed Mode**: Run `ingestion`, `processor`, and `api` services independently

Future enhancements will add NATS JetStream integration to enable true event-driven architecture.

## Architecture Documentation

Detailed architecture documentation is available in [`docs/architecture/`](./docs/architecture/):
- [00-overview.md](./docs/architecture/00-overview.md) - System overview
- [01-components.md](./docs/architecture/01-components.md) - Component descriptions
- [02-dataflow.md](./docs/architecture/02-dataflow.md) - Data flow diagrams
- [03-modes.md](./docs/architecture/03-modes.md) - AIO vs Distributed modes

## Roadmap

- [x] Separate components into distinct services
- [ ] NATS JetStream integration for event bus
- [ ] Object storage for artifacts (MinIO/S3)
- [ ] GraphQL API implementation
- [ ] Web UI (React + Tailwind + shadcn/ui)
- [ ] Authentication layer (dev token, OIDC)
- [ ] Metrics (Prometheus) and tracing (OpenTelemetry)
- [ ] Docker Compose profiles (AIO and distributed)
- [ ] Kubernetes Helm charts

## License

(Choose and add a license file if needed.)
