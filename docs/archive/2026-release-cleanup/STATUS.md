# Project Status (Short)

**Last updated**: 2025-12-17

Observer is a test observability system that ingests test execution events via gRPC, publishes them to NATS JetStream, persists them to MongoDB, and serves historical + real-time views via an API service (REST + WebSocket) and a React Web UI.

## Where to look

- Run it: [../../QUICKSTART.md](../../QUICKSTART.md)
- System design: [../architecture/00-overview.md](../architecture/00-overview.md)
- Roadmap: [../architecture/10-next-steps.md](../architecture/10-next-steps.md)

## Key capabilities

- Distributed services: ingestion (gRPC), processor (JetStream consumer), API (REST + WebSocket), web UI
- Deployment: Docker Compose profiles (AIO + distributed), Helm chart
- Real-time streaming: WebSocket endpoint backed by a JetStream consumer

## Notes

If you find older documents at the repository root that read like “implementation summary” or “status report”, they are considered historical and are archived under [archive/](../README.md).
