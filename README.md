# Observer Service

A gRPC service that collects test execution events (start, step, finish). This repository includes:

- Structured server implementation with validation and interceptors (logging + panic recovery)
- Graceful shutdown with signal handling
- Deterministic in-process bufconn based tests

## Quick Start

Build:

```bash
make build
```

Run (default port 50051):

```bash
make run
```

Override port:

```bash
PORT=6000 go run ./server
# or
go run ./server -port 6000
```

## Tests

```bash
make test
```

The test suite uses an in-process `bufconn` listener (no external ports) and validates argument handling.

## Make Targets

- `make build` – Compile all packages
- `make run` – Run the server (depends on build)
- `make test` – Run all tests
- `make proto` – Generate gRPC stubs (requires `proto/service.proto` present)
- `make tools` – Install/upgrade protobuf plugins
- `make lint` – Placeholder for future lint integration

## Configuration

| Variable | Flag    | Default | Description           |
| -------- | ------- | ------- | --------------------- |
| `PORT`   | `-port` | 50051   | TCP port to listen on |

## Logging

Uses Go 1.21+ `slog` with text handler. Interceptors log RPC method, duration, peer, status code, and errors. Panic recovery interceptor converts panics to `Internal` status and logs stack traces.

## Validation

Handlers validate presence of `TestId`. Missing / empty IDs return `InvalidArgument`.

## Roadmap / Suggestions

- Add metrics (Prometheus) and tracing (OpenTelemetry)
- Integrate health checking service
- Add linting (`golangci-lint`) and CI workflow
- Add TLS configuration & optional authentication
- Expand test coverage (step events edge cases, deadlines, cancellation)

## License

(Choose and add a license file if needed.)
