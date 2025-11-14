# Observer Service

[![Open in GitHub Codespaces](https://github.com/codespaces/badge.svg)](https://codespaces.new/stanterprise/observer?quickstart=1)

A test observability system that collects test execution events via gRPC. The system can operate in two modes:

- 🧩 **All-in-One (AIO)** — Single container with embedded services for local/dev use
- ⚙️ **Distributed Mode** — Multi-container deployment for production/CI

> 💡 **Quick Start with Codespaces:** Click the badge above to launch a fully configured development environment in seconds! See [CODESPACES.md](CODESPACES.md) for details.

## Architecture

The Observer system is composed of three main components:

### 1. **Ingestion Service** (`cmd/ingestion`)

- gRPC endpoint for test event collection
- Stateless and horizontally scalable
- Validates protobuf payloads
- **Phase 1 Complete**: Publishes to NATS JetStream (dual-write with optional DB)

### 2. **Processor Service** (`cmd/processor`)

- **Phase 2 Complete**: NATS JetStream consumer for event processing
- Persists events to database with idempotent upsert pattern
- Handles database migrations
- Supports horizontal scaling via durable consumer groups
- Future: artifact storage, summary generation

### 3. **API Service** (`cmd/api`)

- HTTP/GraphQL API for web UI and integrations
- Read-only database access
- Future: WebSocket for real-time updates, authentication

See detailed documentation in each component's README:

- [Ingestion Service](./cmd/ingestion/README.md)
- [Processor Service](./cmd/processor/README.md)
- [API Service](./cmd/api/README.md)

## Development Environment

### GitHub Codespaces (Recommended)

The fastest way to start developing is with GitHub Codespaces—a complete, pre-configured development environment in your browser:

1. Click the **Open in Codespaces** badge at the top of this README
2. Wait 2-3 minutes for automatic setup
3. Start coding immediately!

Codespaces includes:

- ✅ Go 1.23 with all dev tools (gopls, golangci-lint, delve)
- ✅ Docker and Docker Compose
- ✅ PostgreSQL and NATS auto-started
- ✅ VS Code with debugging and Go extensions
- ✅ Pre-built binaries and passing tests

See [CODESPACES.md](CODESPACES.md) for the complete guide.

### Local Development

For local development, ensure you have:

- Go 1.23+
- Docker and Docker Compose
- Protocol Buffers compiler (for code generation)
- Make

Install development tools:

```bash
make tools
```

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
# Start infrastructure services
make db-up    # Start PostgreSQL
make nats-up  # Start NATS

# Ingestion (stateless, publishes to NATS)
NATS_URL='nats://localhost:4222' ./bin/ingestion

# API (optional DB connection)
./bin/api

# Processor (requires DB)
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

### NATS

- `make nats-up` – Start NATS container
- `make nats-down` – Stop NATS container
- `make nats-logs` – Tail NATS logs

### Testing & Quality

- `make test` – Run all tests
- `make test-race` – Run tests with race detector
- `make test-cover` – Run tests with coverage
- `make test-nats-integration` – Run NATS integration tests (requires NATS running)
- `make fmt` – Format code
- `make vet` – Vet code
- `make lint` – Run golangci-lint

### Tools

- `make proto` – Generate gRPC stubs
- `make tools` – Install dev tools

## Configuration

### Ingestion Service

| Variable              | Default           | Description                                               |
| --------------------- | ----------------- | --------------------------------------------------------- |
| `PORT`                | `50051`           | gRPC listening port                                       |
| `NATS_URL`            | -                 | NATS server URL (optional, e.g., `nats://localhost:4222`) |
| `NATS_STREAM`         | `tests_events`    | JetStream stream name                                     |
| `NATS_SUBJECT_PREFIX` | `tests.events.v1` | Subject prefix for events                                 |

### Processor Service

| Variable           | Default | Description                             |
| ------------------ | ------- | --------------------------------------- |
| `PORT`             | `50052` | gRPC listening port                     |
| `DATABASE_URL`     | -       | PostgreSQL connection string (required) |
| `APPLY_MIGRATIONS` | -       | Set to `1` to enable auto-migrations    |

### API Service

| Variable       | Default | Description                             |
| -------------- | ------- | --------------------------------------- |
| `PORT`         | `8080`  | HTTP listening port                     |
| `DATABASE_URL` | -       | PostgreSQL connection string (optional) |

## Logging

Uses Go 1.21+ `slog` with text handler. Interceptors log RPC method, duration, peer, status code, and errors. Panic recovery interceptor converts panics to `Internal` status and logs stack traces.

## Validation

Handlers validate presence of `TestId`. Missing / empty IDs return `InvalidArgument`.

## Migration from Monolithic to Distributed

The repository maintains backward compatibility with the monolithic `server/main.go` deployment while supporting the new distributed architecture:

1. **Legacy Mode**: Run `./bin/observer` for single-process deployment
2. **Distributed Mode**: Run `ingestion`, `processor`, and `api` services independently

**Phase 2 Complete**: The system now supports full NATS JetStream integration with both publisher (ingestion) and consumer (processor) services. The processor service runs as a pure NATS consumer with database persistence, enabling fully distributed event-driven architecture.

## Architecture Documentation

Detailed architecture documentation is available in [`docs/architecture/`](./docs/architecture/):

- [00-overview.md](./docs/architecture/00-overview.md) - System overview
- [01-components.md](./docs/architecture/01-components.md) - Component descriptions
- [02-dataflow.md](./docs/architecture/02-dataflow.md) - Data flow diagrams
- [03-modes.md](./docs/architecture/03-modes.md) - AIO vs Distributed modes

## Roadmap

- [x] Separate components into distinct services
- [x] **Phase 1**: NATS JetStream publisher integration (dual-write)
- [x] **Phase 2**: Processor service NATS consumer with database persistence
- [x] Docker Compose profiles (AIO and distributed)
- [x] Comprehensive test suite with E2E NATS integration
- [x] Playwright reporter integration validation
- [ ] **Phase 3**: Remove DB from ingestion (NATS-only, fully stateless)
- [ ] **Phase 4**: GraphQL API implementation
- [ ] Object storage for artifacts (MinIO/S3)
- [ ] Web UI (React + Tailwind + shadcn/ui)
- [ ] Authentication layer (dev token, OIDC)
- [ ] Metrics (Prometheus) and tracing (OpenTelemetry)
- [ ] Kubernetes Helm charts

## License

(Choose and add a license file if needed.)
