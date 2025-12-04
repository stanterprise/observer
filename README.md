# Observer Service

[![Open in GitHub Codespaces](https://github.com/codespaces/badge.svg)](https://codespaces.new/stanterprise/observer?quickstart=1)

A test observability system that collects test execution events via gRPC. The system can operate in two modes:

- 🧩 **All-in-One (AIO)** — Single container with embedded services for local/dev use
- ⚙️ **Distributed Mode** — Multi-container deployment for production/CI

> 💡 **Quick Start with Codespaces:** Click the badge above to launch a fully configured development environment in seconds! See [CODESPACES.md](CODESPACES.md) for details.

## Quick Start

Get Observer running in 2 minutes! Choose your preferred method:

**Docker (Fastest)**

```bash
docker run -d -p 3000:80 -p 50051:50051 -v observer-data:/data \
  ghcr.io/stanterprise/observer/aio:latest
```

**Kubernetes/Helm**

```bash
helm install observer oci://ghcr.io/stanterprise/observer/charts/observer --version 0.1.0
kubectl port-forward svc/observer-web 3000:80
```

Access the Web UI at http://localhost:3000 and gRPC at localhost:50051

📖 See [QUICKSTART.md](QUICKSTART.md) for detailed instructions and more deployment options.

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

- HTTP REST/GraphQL API for web UI and integrations (✅ Implemented)
- **WebSocket endpoint for real-time event streaming** (`/ws`)
- NATS JetStream consumer for event relay to WebSocket clients
- REST endpoints for test listing, run statistics, and details
- GraphQL support with interactive playground
- Read-only database access

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
- ✅ MongoDB and NATS auto-started
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
make mongodb-up    # Start MongoDB
make nats-up       # Start NATS

# Ingestion (stateless, publishes to NATS)
NATS_URL='nats://localhost:4222' ./bin/ingestion

# API (requires MongoDB)
MONGODB_URI='mongodb://root:password@localhost:27017/observer?authSource=admin' ./bin/api

# Processor (requires MongoDB)
MONGODB_URI='mongodb://root:password@localhost:27017/observer?authSource=admin' ./bin/processor
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

**Docker Images:**

- `make docker-build-all` – Build all Docker images (standard)
- `make docker-build-aio` – Build AIO image
- `make docker-buildx-aio` – **Optimized multi-platform build (60-90% faster)** ⚡

> 💡 **Build Performance**: Multi-architecture builds optimized from ~20min to ~2-8min using BuildKit cache mounts.  
> See [Build Optimization Guide](docs/BUILD_OPTIMIZATION.md) for details.

**Setup for optimized builds:**

```bash
./scripts/setup-buildx.sh  # One-time setup
make docker-buildx-aio      # Fast cached builds
```

### Running

- `make run` – Run legacy server (depends on build)
- `make run-dev` – Run with MongoDB database

### Database

- `make mongodb-up` – Start MongoDB container
- `make mongodb-down` – Stop containers and remove volumes
- `make mongodb-shell` – Open mongosh against the database
- `make mongodb-reset` – Reset database

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

| Variable       | Default | Description                                 |
| -------------- | ------- | ------------------------------------------- |
| `MONGODB_URI`  | -       | MongoDB connection string (required)        |
| `NATS_URL`     | -       | NATS server URL (required)                  |
| `NATS_STREAM`  | -       | JetStream stream name                       |

### API Service

| Variable           | Default        | Description                                   |
| ------------------ | -------------- | --------------------------------------------- |
| `PORT`             | `8080`         | HTTP listening port                           |
| `MONGODB_URI`      | -              | MongoDB connection string (required)          |
| `NATS_URL`         | -              | NATS server URL (optional, for WebSocket)     |
| `NATS_STREAM`      | `tests_events` | JetStream stream name for WebSocket relay     |
| `NATS_WS_CONSUMER` | `websocket`    | Consumer name for WebSocket NATS subscription |

### MongoDB Configuration

MongoDB can be configured using either a connection URI or split environment variables:

| Variable            | Default    | Description                           |
| ------------------- | ---------- | ------------------------------------- |
| `MONGODB_URI`       | -          | Full MongoDB connection URI           |
| `MONGO_URI`         | -          | Alias for `MONGODB_URI`               |
| `MONGO_HOST`        | -          | MongoDB server host                   |
| `MONGO_PORT`        | `27017`    | MongoDB server port                   |
| `MONGO_USER`        | -          | MongoDB username                      |
| `MONGO_PASSWORD`    | -          | MongoDB password                      |
| `MONGO_DATABASE`    | `observer` | MongoDB database name                 |
| `MONGO_AUTH_SOURCE` | `admin`    | MongoDB authentication source         |

**Connection URI format:**
```
mongodb://[user:pass@]host[:port]/database[?options]
mongodb+srv://[user:pass@]host/database[?options]
```

## Database Backends

The Observer service supports two database backends:

### MongoDB (Recommended for new deployments)

MongoDB provides a document-based data model that aligns well with test run hierarchies:

- **Test runs are stored as single documents** containing embedded tests, suites, and steps
- **Flexible schema** for storing metadata and custom attributes
- **Efficient queries** for retrieving complete test run data
- **Better suited** for hierarchical test structures (suites → tests → steps)

**Docker Compose with MongoDB:**
```bash
docker compose --profile dist up -d
```

## WebSocket Real-Time Events

The API service exposes a WebSocket endpoint at `/ws` for real-time test event streaming.

### Connecting to WebSocket

```javascript
const ws = new WebSocket("ws://localhost:8080/ws");

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log("Event:", data.type, data);
};
```

### Event Format

All events follow this structure:

```json
{
  "type": "test.begin|test.end|step.begin|step.end",
  "timestamp": "2025-11-14T05:00:00Z",
  "data": {
    /* event-specific data */
  }
}
```

### Test Client

A simple HTML test client is available at [`docs/websocket-test-client.html`](./docs/websocket-test-client.html). Open it in a browser to connect to the WebSocket endpoint and view real-time events.

**Note**: The WebSocket functionality requires `NATS_URL` to be configured. Without NATS, the WebSocket endpoint will accept connections but won't relay events.

## Logging

Uses Go 1.21+ `slog` with text handler. Interceptors log RPC method, duration, peer, status code, and errors. Panic recovery interceptor converts panics to `Internal` status and logs stack traces.

## Validation

Handlers validate presence of `TestId`. Missing / empty IDs return `InvalidArgument`.

## Migration from Monolithic to Distributed

The repository maintains backward compatibility with the monolithic `server/main.go` deployment while supporting the new distributed architecture:

1. **Legacy Mode**: Run `./bin/observer` for single-process deployment
2. **Distributed Mode**: Run `ingestion`, `processor`, and `api` services independently

**Phase 2 Complete**: The system now supports full NATS JetStream integration with both publisher (ingestion) and consumer (processor, WebSocket) services. The processor service runs as a pure NATS consumer with database persistence, enabling fully distributed event-driven architecture. The API service includes WebSocket support for real-time event streaming to web clients, and the Web UI provides a modern React-based interface with live updates.

## Architecture Documentation

Detailed architecture documentation is available in [`docs/architecture/`](./docs/architecture/):

- [00-overview.md](./docs/architecture/00-overview.md) - System overview
- [01-components.md](./docs/architecture/01-components.md) - Component descriptions
- [02-dataflow.md](./docs/architecture/02-dataflow.md) - Data flow diagrams
- [03-modes.md](./docs/architecture/03-modes.md) - AIO vs Distributed modes

## Web UI

The Observer service includes a modern web interface built with React, TypeScript, and Tailwind CSS.

### Features

- **Real-time Updates**: Live test execution monitoring via WebSocket
- **Test Run Listing**: View all test runs with status, timing, and metadata
- **Responsive Design**: Mobile-friendly interface
- **Configurable Endpoints**: Environment-based API and WebSocket configuration

### Access

- **AIO Mode**: `http://localhost:3000` (Web UI served by Nginx on port 80/3000)
- **Distributed Mode**: `http://localhost:3000` (Standalone Web UI service)

### Development

See [web/README.md](./web/README.md) for web UI development and [web/README-LOCAL-DEV.md](./web/README-LOCAL-DEV.md) for running web locally with Docker backend.

**Option 1: Local Web + Docker Backend** (Recommended)

```bash
# Start backend services (DB, NATS, ingestion, processor, API)
docker compose --profile web-dev up -d

# Run web dev server
cd web
npm install
npm run dev  # Opens on http://localhost:3000
```

**Option 2: Full Development Mode**

```bash
cd web
npm install
npm run dev
```

The development server includes proxying for API and WebSocket endpoints to `localhost:8080`.

## Deployment

### Docker Images

Pre-built Docker images are available on GitHub Container Registry:

```bash
# Pull AIO image
docker pull ghcr.io/stanterprise/observer/aio:latest

# Pull distributed mode images
docker pull ghcr.io/stanterprise/observer/ingestion:latest
docker pull ghcr.io/stanterprise/observer/processor:latest
docker pull ghcr.io/stanterprise/observer/api:latest
docker pull ghcr.io/stanterprise/observer/web:latest
```

### Kubernetes / Helm

Install Observer on Kubernetes using Helm:

```bash
# Install from OCI registry
helm install observer oci://ghcr.io/stanterprise/observer/charts/observer --version 0.1.0

# Or add the Helm repository
helm repo add observer https://stanterprise.github.io/observer/
helm install observer observer/observer
```

See the [Deployment Guide](DEPLOYMENT.md) for detailed instructions on:

- Docker image usage
- Helm chart installation and configuration
- Production deployment
- AIO vs Distributed mode selection
- Ingress configuration
- Scaling and monitoring

## Roadmap

- [x] Separate components into distinct services
- [x] **Phase 1**: NATS JetStream publisher integration (dual-write)
- [x] **Phase 2**: Processor service NATS consumer with database persistence
- [x] **WebSocket component**: Real-time event streaming to web clients
- [x] **Web UI**: React + TypeScript + Tailwind CSS interface
- [x] **REST API**: Test listing, run statistics, and detail endpoints
- [x] Docker Compose profiles (AIO and distributed)
- [x] Comprehensive test suite with E2E NATS integration
- [x] Playwright reporter integration validation
- [x] **Docker image publishing**: GitHub Container Registry
- [x] **Kubernetes Helm charts**: Deployment templates for AIO and distributed modes
- [ ] **Phase 3**: Remove DB from ingestion (NATS-only, fully stateless)
- [ ] **Phase 4**: Complete GraphQL API implementation
- [ ] Enhanced Web UI features (test details, artifact viewer, filtering)
- [ ] Object storage for artifacts (MinIO/S3)
- [ ] Authentication layer (dev token, OIDC)
- [ ] Metrics (Prometheus) and tracing (OpenTelemetry)

## CI/CD & Build Optimization

The project uses optimized GitHub Actions workflows with BuildKit cache mounts for fast, efficient builds:

### Build Performance

- **Multi-platform builds** (AMD64 + ARM64) in ~8-12 minutes (first build)
- **Cached rebuilds** in ~2-4 minutes for code changes
- **60-90% faster** than traditional Docker builds

### Workflows

- **docker-publish.yml** - Automated image building and publishing with dual cache strategy
- **build-performance.yml** - Weekly validation of build optimization effectiveness
- **cache-cleanup.yml** - Automated registry cache management

### Documentation

- [Build Optimization Guide](docs/BUILD_OPTIMIZATION.md) - Complete optimization details
- [GitHub Actions Integration](docs/GITHUB_ACTIONS_BUILDS.md) - CI/CD examples and best practices
- [Quick Reference](BUILD_QUICK_REF.md) - Essential commands and troubleshooting

## License

(Choose and add a license file if needed.)
