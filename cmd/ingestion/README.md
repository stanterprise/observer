# Ingestion Service

The ingestion service is the gRPC entry point for test event collection. It receives test execution events from reporters and processes them.

## Architecture

The ingestion service is designed to be stateless and horizontally scalable. In the target distributed architecture, it will:

1. Accept gRPC calls from reporters (Playwright, pytest, etc.)
2. Validate protobuf payloads
3. Publish validated events to NATS JetStream
4. Handle backpressure and transient errors

## Current State

Currently, the ingestion service runs as a standalone gRPC server that can operate:
- **Without database**: Pure ingestion mode (future: will publish to NATS)
- **With database**: Direct persistence mode (backward compatible with monolithic setup)

## Running

### Standalone (no database)

```bash
./bin/ingestion
# or
make build-ingestion && ./bin/ingestion
```

Default port: `50051`

### Custom port

```bash
PORT=6000 ./bin/ingestion
# or
./bin/ingestion -port 6000
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `50051` | gRPC listening port |
| `NATS_URL` | - | NATS server URL (future) |

## Future Enhancements

- [ ] NATS JetStream publisher integration
- [ ] Metrics endpoint (Prometheus)
- [ ] Health check endpoint
- [ ] OpenTelemetry tracing
- [ ] TLS support
