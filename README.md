# Observer Service

[![Open in GitHub Codespaces](https://github.com/codespaces/badge.svg)](https://codespaces.new/stanterprise/observer?quickstart=1)

A test observability system for automated test execution. Observer currently integrates with Playwright and operates in two deployment modes:

- 🧩 **All-in-One (AIO)** — Single container with embedded MongoDB, PostgreSQL, and NATS for local/dev use
- ⚙️ **Distributed Mode** — Multi-container deployment for production/CI

> 💡 **Quick Start with Codespaces:** Click the badge above to launch a fully configured development environment in seconds! See [CODESPACES.md](CODESPACES.md) for details.

## Use Observer

The [Observer documentation website](https://observer.stanterprise.com/) is the canonical source for end-user guidance:

- [Getting Started](https://observer.stanterprise.com/docs/getting-started/)
- [Installation and deployment](https://observer.stanterprise.com/docs/install/)
- [Playwright reporter integration](https://observer.stanterprise.com/docs/integrations/playwright-reporter/)
- [Hosted demo](https://observer.rocks)

## Architecture at a glance

- Ingestion receives reporter events over gRPC and publishes them to NATS JetStream.
- The processor persists durable run data to PostgreSQL and uses MongoDB only for live in-flight step buffering.
- The API provides REST queries and an optional NATS-backed WebSocket event relay.
- The web UI consumes the API and live event stream.

For component internals, see [`docs/architecture/`](docs/architecture/) and the service READMEs in `cmd/`. GraphQL, authentication/OIDC, metrics, Kafka, and non-Playwright reporters are not currently supported interfaces; see the [public roadmap](https://observer.stanterprise.com/docs/roadmap/) for planned work.

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
docker compose up -d postgres
make nats-up       # Start NATS

# Ingestion (stateless, publishes to NATS)
NATS_URL='nats://localhost:4222' ./bin/ingestion

# API (requires PostgreSQL)
POSTGRES_DSN='postgres://observer:password@localhost:5432/observer?sslmode=disable' \
NATS_URL='nats://localhost:4222' \
./bin/api

# Processor (requires PostgreSQL and MongoDB)
POSTGRES_DSN='postgres://observer:password@localhost:5432/observer?sslmode=disable' \
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

| Variable        | Default                 | Description                                        |
| --------------- | ----------------------- | -------------------------------------------------- |
| `POSTGRES_DSN`  | -                       | PostgreSQL connection string for relational writes |
| `MONGODB_URI`   | -                       | MongoDB connection string for live step buffering  |
| `NATS_URL`      | `nats://localhost:4222` | NATS server URL                                    |
| `NATS_STREAM`   | `tests_events`          | JetStream stream name                              |
| `NATS_CONSUMER` | `processor`             | Durable consumer name for JetStream consumer       |

### API Service

| Variable           | Default        | Description                                   |
| ------------------ | -------------- | --------------------------------------------- |
| `PORT`             | `8080`         | HTTP listening port                           |
| `POSTGRES_DSN`     | -              | PostgreSQL connection string (required)       |
| `NATS_URL`         | -              | NATS server URL (optional, for WebSocket)     |
| `NATS_STREAM`      | `tests_events` | JetStream stream name for WebSocket relay     |
| `NATS_WS_CONSUMER` | `websocket`    | Consumer name for WebSocket NATS subscription |

### MongoDB Configuration

MongoDB can be configured using either a connection URI or split environment variables:

| Variable            | Default    | Description                   |
| ------------------- | ---------- | ----------------------------- |
| `MONGODB_URI`       | -          | Full MongoDB connection URI   |
| `MONGO_URI`         | -          | Alias for `MONGODB_URI`       |
| `MONGO_HOST`        | -          | MongoDB server host           |
| `MONGO_PORT`        | `27017`    | MongoDB server port           |
| `MONGO_USER`        | -          | MongoDB username              |
| `MONGO_PASSWORD`    | -          | MongoDB password              |
| `MONGO_DATABASE`    | `observer` | MongoDB database name         |
| `MONGO_AUTH_SOURCE` | `admin`    | MongoDB authentication source |

**Connection URI format:**

```
mongodb://[user:pass@]host[:port]/database[?options]
mongodb+srv://[user:pass@]host/database[?options]
```

## Data Stores

PostgreSQL is the authoritative store for durable run data and API queries. MongoDB is limited to transient in-flight step buffering (`live_step_buffers`). Deployment-specific connection and storage configuration is documented in the [deployment guide](DEPLOYMENT.md).

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

## Service Architecture

The current event path is gRPC ingestion → NATS JetStream → processor. PostgreSQL stores durable run data; MongoDB is limited to live in-flight step buffering. The API reads from PostgreSQL and can relay live events over WebSocket when NATS is configured. The AIO image bundles services for local evaluation; distributed mode runs them separately. A legacy monolithic binary is retained for compatibility.

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

For end-user installation and supported deployment paths, use the [public installation guide](https://observer.stanterprise.com/docs/install/). This repository retains chart implementation and operator details in [DEPLOYMENT.md](DEPLOYMENT.md) and [charts/observer/README.md](charts/observer/README.md).

## Roadmap

Current product capability and integration status is maintained in the [public roadmap](https://observer.stanterprise.com/docs/roadmap/). Implementation-specific work remains in this repository's `docs/` and issue tracker.

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

## Development Documentation

This project includes comprehensive development documentation:

- **[CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)** - Community guidelines
- **[CONTRIBUTING.md](CONTRIBUTING.md)** - How to contribute
- **[CHANGELOG.md](CHANGELOG.md)** - Version history and changes

### Internal Documentation

The repository contains development workflow documentation used during the project's evolution:

- **`.github/agents/`** - AI agent configurations for development automation (see [.github/README.md](.github/README.md))
- **`.specify/`** - Project specification and governance documentation (see [.specify/README.md](.specify/README.md))
- **`docs/archive/`** - Historical development notes and implementation summaries (see [docs/archive/README.md](docs/archive/README.md))

These files document our development process and provide context for architectural decisions. Contributors don't need to interact with them directly.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for setup, workflow, and testing guidance.

## Security

See [SECURITY.md](SECURITY.md) for vulnerability reporting.

## License

Licensed under the Apache License 2.0. See [LICENSE](LICENSE).
