# GitHub Codespaces Quick Start Guide

This repository is fully configured for [GitHub Codespaces](https://github.com/features/codespaces), providing an instant, cloud-based development environment with all tools and dependencies pre-installed.

## 🚀 Getting Started

### Launch a Codespace

1. **From GitHub Web UI:**
   - Navigate to this repository on GitHub
   - Click the **Code** button (green)
   - Select the **Codespaces** tab
   - Click **Create codespace on main**

2. **From VS Code Desktop:**
   - Install the [GitHub Codespaces extension](https://marketplace.visualstudio.com/items?itemName=GitHub.codespaces)
   - Open Command Palette (`Ctrl+Shift+P` / `Cmd+Shift+P`)
   - Select "Codespaces: Create New Codespace"
   - Choose this repository

3. **From GitHub CLI:**
   ```bash
   gh codespace create -r stanterprise/observer
   gh codespace code
   ```

### Initial Setup

The Codespace will automatically:

1. ✅ Build the Go 1.24 development container with Node.js LTS
2. ✅ Install development tools (golangci-lint, protoc, gopls, delve, TypeScript)
3. ✅ Download Go dependencies
4. ✅ Build all service components
5. ✅ Run tests to verify setup
6. ✅ Start MongoDB and NATS containers
7. ✅ Create `.env` file from template

This process takes **2-3 minutes** on first launch. Subsequent launches are faster.

## 🛠️ What's Pre-Configured

### Infrastructure Services (Auto-Started)

- **MongoDB** - Database on port 27017
- **NATS JetStream** - Message broker on port 4222

Check status:

```bash
docker compose ps
```

### Development Tools

**Backend:**

- **Go 1.24** with gopls language server
- **Delve** debugger
- **golangci-lint** for code quality
- **protoc** for gRPC code generation

**Frontend:**

- **Node.js LTS** with npm and Yarn
- **TypeScript** compiler and language tools

**Infrastructure:**

- **Docker and Docker Compose**
- **Make** for build automation

### Environment Variables

All required variables are pre-set:

```bash
MONGODB_URI=mongodb://root:change-me@localhost:27017/observer?authSource=admin
NATS_URL=nats://localhost:4222
```

View all variables:

```bash
make env-print
```

### VS Code Extensions

Pre-installed and configured:

- Go language support with debugging
- TypeScript and JavaScript language support
- ESLint for JavaScript/TypeScript linting
- Tailwind CSS IntelliSense
- Docker management
- GitHub Copilot
- Protobuf syntax
- Makefile tools

## 📝 Common Tasks

### Build All Components

```bash
make build-all
```

Builds:

- `bin/observer` - Legacy monolithic server
- `bin/ingestion` - Ingestion service
- `bin/processor` - Processor service
- `bin/api` - API service

### Run Tests

```bash
# All tests
make test

# With coverage
make test-cover

# With race detector
make test-race

# NATS integration tests
make test-nats-integration
```

### Start Services

**Legacy Monolithic Mode:**

```bash
make run-dev
```

**Distributed Mode:**

```bash
# Terminal 1: Ingestion
./bin/ingestion

# Terminal 2: Processor
./bin/processor

# Terminal 3: API
./bin/api
```

### Database Operations

```bash
# Open MongoDB shell
make mongo-shell

# View logs
make mongo-logs

# Reset database
make mongo-reset
```

### Code Quality

```bash
# Format code
make fmt

# Lint code
make lint

# Vet code
make vet
```

## 🐛 Debugging

### Using VS Code Debugger

1. **Open Debug Panel** (Ctrl+Shift+D / Cmd+Shift+D)
2. **Select a configuration:**
   - Debug Ingestion Service
   - Debug Processor Service
   - Debug API Service
   - Debug Legacy Server
   - Debug All Services (runs all three)
3. **Press F5** to start debugging

Breakpoints, variable inspection, and step debugging work out of the box.

### Debugging Tests

1. **Open a test file**
2. **Set breakpoints**
3. **Select debug configuration:**
   - Debug Current Test (runs selected test)
   - Debug Package Tests (runs all tests in package)
   - Debug All Tests
4. **Press F5**

## 🔧 VS Code Tasks

Access via **Terminal → Run Task** or `Ctrl+Shift+P` → "Tasks: Run Task":

### Build Tasks

- **Build All Components** - `Ctrl+Shift+B`
- Build Ingestion Service
- Build Processor Service
- Build API Service

### Test Tasks

- **Run Tests** - `Ctrl+Shift+T`
- Run Tests with Coverage
- Run Tests with Race Detector

### Infrastructure Tasks

- Start Database
- Start NATS
- Start All Infrastructure
- Stop All Infrastructure

## 🌐 Accessing Services

Codespaces automatically forwards ports. Click the **Ports** panel at the bottom to see:

| Port  | Service         | Access                                 |
| ----- | --------------- | -------------------------------------- |
| 50051 | gRPC Ingestion  | Use gRPC client or grpcurl             |
| 8080  | HTTP API        | Click "Open in Browser" in Ports panel |
| 27017 | MongoDB         | mongosh or any MongoDB client          |
| 4222  | NATS            | NATS client or CLI                     |
| 8222  | NATS Monitoring | Open in browser for NATS dashboard     |

### Testing gRPC Endpoints

```bash
# Install grpcurl
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

# List services
grpcurl -plaintext localhost:50051 list

# Make a request (example)
grpcurl -plaintext -d '{"test_case": {"id": "test-123"}}' \
  localhost:50051 testsystem.v1.TestObserver/ReportTestStart
```

## 💡 Pro Tips

### Multiple Terminals

Codespaces supports split terminals:

- `Ctrl+Shift+5` - Split terminal
- `Ctrl+` ` - Focus terminal

Run different services in separate terminals for easier monitoring.

### Custom Environment Variables

Edit `.env` file:

```bash
export MONGODB_URI='mongodb://root:change-me@localhost:27017/observer?authSource=admin'
export NATS_URL='nats://localhost:4222'
```

For faster iteration:
Start MongoDB and NATS:

# Use cached test results when appropriate

go test -count=1 ./... # Skip cache
make mongo-up nats-up

# Focus on specific package

go test ./pkg/server -v

```

### Docker Compose Logs

MONGODB_URI=mongodb://root:change-me@localhost:27017/observer?authSource=admin
# All services
docker compose logs -f

# Specific service
docker compose logs -f db
docker compose logs -f nats
```

## 🔄 Updating Dependencies

```bash
# Update all Go dependencies
go get -u ./...
go mod tidy

# Update specific dependency
go get -u github.com/nats-io/nats.go@latest
go mod tidy

# Install new dev tools
make tools
```

## 🧹 Cleanup

### Reset Environment

```bash
# Clean build artifacts
make clean

# Stop and remove all containers
docker compose down -v

# Restart infrastructure
docker compose up -d db nats
```

### Rebuild Codespace

If you need a fresh start:

1. Open Command Palette
2. "Codespaces: Rebuild Container"
3. Wait for rebuild to complete

## 📚 Additional Resources

- [Codespaces Documentation](.devcontainer/README.md)
- [Architecture Documentation](docs/architecture/)
- [Main README](README.md)
- [VS Code Go Documentation](https://code.visualstudio.com/docs/languages/go)

## 🆘 Troubleshooting

### Services Not Starting

```bash
# Check Docker status
docker ps

# Restart services
docker compose down
docker compose up -d db nats

# Check service health
docker compose ps
```

### Build Failures

```bash
# Clean everything
make clean
make clean-cache

# Rebuild
make build-all
```

### Port Conflicts

```bash
# Find process using port
lsof -i :50051

# Kill process if needed
kill -9 <PID>
```

### Go Tools Not Working

```bash
# Reinstall tools
make tools

# Install specific tool
go install golang.org/x/tools/gopls@latest
```

### Database Connection Issues

```bash
# Check MongoDB is running
docker compose ps mongodb

# Test connection
docker compose exec mongodb mongosh --username root --password change-me --authenticationDatabase admin --eval "db.adminCommand({ ping: 1 })" observer

# Reset database
make mongo-reset
```

## 💬 Getting Help

- Check the [.devcontainer/README.md](.devcontainer/README.md) for detailed configuration
- Review [Makefile](Makefile) for available commands
- See [Architecture Documentation](docs/architecture/) for system design

---

**Happy Coding! 🎉**

Your complete development environment is ready. Start coding immediately—no setup required!
