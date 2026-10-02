# Test Observer System Architecture Overview

This project provides **test observability** using a gRPC-based event protocol. The Playwright reporter is currently available; Pytest and Mocha reporter work is in progress, and JUnit and other framework integrations remain planned.
The system can operate in two modes:

- 🧩 **All-in-One (AIO)** — a compact, single-container deployment with embedded MongoDB, PostgreSQL, and NATS for local/dev use.
- ⚙️ **Distributed Mode** — scalable multi-container deployment used in CI/CD or production.

The architecture follows the same service boundaries in both modes, with different packaging and configuration. The Helm AIO preset is distinct from the Docker AIO image: its shipped presets disable the PostgreSQL, MongoDB, and NATS dependency subcharts.

---

## Core Goals

- Unified schema and ingestion protocol (protobuf).
- Real-time event streaming via NATS JetStream. Kafka is a proposed alternative for higher scale; it is not implemented.
- MongoDB-backed live-step buffering plus PostgreSQL-backed relational run storage.
- Pluggable artifact storage (local, S3, MinIO).
- Extensible APIs for custom dashboards, analytics, and alerting. REST and WebSocket are current; GraphQL is planned.
- Simple onboarding (AIO) + horizontal scalability (distributed).

---

## High-Level Flow

```text
Reporter (Playwright available; Pytest/Mocha in progress; other frameworks planned)
    ↓
gRPC ingestion
    ↓
NATS JetStream (event bus)
    ↓
Processor (event consumer → PostgreSQL + MongoDB live-step buffer + artifact storage)
    ↓
REST API and WebSocket event relay
    ↓
Web UI
```

Authentication/OIDC, Prometheus metrics, and OpenTelemetry are planned capabilities, not current service interfaces.
