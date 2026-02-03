# Codespaces / Dev Container Configuration

This directory contains the configuration for GitHub Codespaces and VS Code Dev Containers, providing a complete development environment for the Observer Service.

## What's Included

### Container Features

- **Go 1.24** (Debian Bookworm base)
- **Node.js LTS** with npm and Yarn - For Web UI development
- **Docker-in-Docker** - Full Docker and Docker Compose support
- **Protocol Buffers** - `protoc` compiler for gRPC code generation
- **Zsh with Oh My Zsh** - Enhanced shell experience

### VS Code Extensions

**Go Development:**

- **golang.go** - Go language support with IntelliSense, debugging, and testing

**Web Development:**

- **ms-vscode.vscode-typescript-next** - TypeScript and JavaScript language support
- **dbaeumer.vscode-eslint** - JavaScript/TypeScript linting
- **bradlc.vscode-tailwindcss** - Tailwind CSS IntelliSense

**General:**

- **ms-azuretools.vscode-docker** - Docker container management
- **github.copilot** & **github.copilot-chat** - AI-powered coding assistance
- **zxh404.vscode-proto3** - Protocol Buffers syntax highlighting
- **ms-vscode.makefile-tools** - Makefile support
- **esbenp.prettier-vscode** - Code formatting
- **redhat.vscode-yaml** - YAML support

### Development Tools

The setup script (`setup.sh`) automatically installs:

**Go Tools:**

- `protoc-gen-go` and `protoc-gen-go-grpc` - Protobuf code generators
- `golangci-lint` - Go linter
- `gopls` - Go language server
- `delve` - Go debugger
- `staticcheck` - Go static analyzer

**Web Tools:**

- `TypeScript` - TypeScript compiler and language tools
- `npm` - Node package manager
- `Yarn` - Alternative package manager

### Infrastructure Services

Automatically started on container creation:

- **MongoDB** on port 27017
- **NATS JetStream** on port 4222 (monitoring on 8222)

### Pre-configured Environment Variables

All necessary environment variables are set from `.env.example`:

```bash
MONGODB_URI=mongodb://root:change-me@localhost:27017/observer?authSource=admin
NATS_URL=nats://localhost:4222
```

## Using the Dev Container

### GitHub Codespaces

1. Navigate to the repository on GitHub
2. Click the "Code" button
3. Select "Codespaces" tab
4. Click "Create codespace on main" (or your branch)
5. Wait for the container to build and initialize

The setup process will:

- Install all development tools
- Download Go dependencies
- Build all service components
- Run tests to verify the setup
- Start MongoDB and NATS containers

### VS Code with Dev Containers Extension

1. Install the [Dev Containers extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)
2. Open the repository in VS Code
3. Click "Reopen in Container" when prompted (or use Command Palette: "Dev Containers: Reopen in Container")
4. Wait for initialization to complete

## Available Ports

The following ports are forwarded and labeled:

| Port  | Service         | Auto-forward |
| ----- | --------------- | ------------ |
| 50051 | gRPC Ingestion  | Notify       |
| 8080  | HTTP API        | Notify       |
| 27017 | MongoDB         | Ignore       |
| 4222  | NATS            | Ignore       |
| 8222  | NATS Monitoring | Silent       |

## VS Code Tasks

Available via Command Palette (`Ctrl+Shift+P` or `Cmd+Shift+P`) → "Tasks: Run Task":

### Build Tasks

- **Build All Components** (default build task)
- Build Ingestion Service
- Build Processor Service
- Build API Service

### Test Tasks

- **Run Tests** (default test task)
- Run Tests with Coverage
- Run Tests with Race Detector
- NATS Integration Tests

### Infrastructure Tasks

- Start Database
- Start NATS
- Start All Infrastructure
- Stop All Infrastructure
- Database Shell

### Code Quality Tasks

- Format Code
- Lint Code
- Vet Code
- Generate Protobuf
- Clean Build Artifacts

## Debugging

Launch configurations are available in the Debug panel (F5):

### Single Service Debugging

- **Debug Ingestion Service** - Launch ingestion with NATS
- **Debug Processor Service** - Launch processor with DB
- **Debug API Service** - Launch API with DB
- **Debug Legacy Server** - Launch monolithic server

### Test Debugging

- **Debug Current Test** - Debug selected test function
- **Debug All Tests** - Debug all tests
- **Debug Package Tests** - Debug tests in current package

### Multi-Service Debugging

- **Debug All Services** - Launch ingestion, processor, and API together

### Advanced

- **Attach to Process** - Attach debugger to running process

## Common Workflows

### Building and Testing

```bash
# Build all components
make build-all

# Run all tests
make test

# Run tests with coverage
make test-cover

# Lint code
make lint
```

### Running Services

```bash
# Start infrastructure
make mongo-up
make nats-up

# Run individual services
./bin/ingestion    # Port 50051
./bin/processor    # Background consumer (no public port)
./bin/api          # Port 8080

# Or use the legacy monolithic server
make run-dev
```

### Database Operations

```bash
# Open MongoDB shell
make mongo-shell

# Reset database
make mongo-reset

# View logs
make mongo-logs
```

### NATS Operations

```bash
# View NATS logs
make nats-logs

# Run NATS integration tests
make test-nats-integration
```

## Customization

### Adding More Tools

Edit `.devcontainer/devcontainer.json` and add features:

```json
"features": {
  "ghcr.io/devcontainers/features/your-feature:1": {}
}
```

### Adding VS Code Extensions

Add to the `extensions` array in `devcontainer.json`:

```json
"customizations": {
  "vscode": {
    "extensions": [
      "your.extension-id"
    ]
  }
}
```

### Modifying Setup Script

Edit `.devcontainer/setup.sh` to add initialization steps.

## Troubleshooting

### Services not starting

Check if Docker is running:

```bash
docker ps
docker compose ps
```

Restart services:

```bash
docker compose down
docker compose up -d db nats
```

### Build failures

Clean and rebuild:

```bash
make clean
make clean-cache
make build-all
```

### Go tools missing

Reinstall tools:

```bash
make tools
```

### Port conflicts

Check for processes using ports:

```bash
lsof -i :50051
lsof -i :27017
```

## Architecture

This development environment supports the Observer Service architecture:

- **Ingestion Service** - Stateless gRPC ingestion, publishes to NATS
- **Processor Service** - NATS consumer, persists to MongoDB
- **API Service** - HTTP REST + WebSocket API for queries/streaming

See [Architecture Documentation](../docs/architecture/) for more details.

## Resources

- [Dev Containers Documentation](https://code.visualstudio.com/docs/devcontainers/containers)
- [GitHub Codespaces Documentation](https://docs.github.com/en/codespaces)
- [Go in VS Code](https://code.visualstudio.com/docs/languages/go)
- [Observer Service README](../README.md)
