# Test Observer System Architecture Overview

This project provides **test observability** across multiple frameworks (Playwright, pytest, JUnit, etc.) using a gRPC-based event protocol.  
The system can operate in two modes:

- 🧩 **All-in-One (AIO)** — a compact, single-container deployment for local/dev use.
- ⚙️ **Distributed Mode** — scalable multi-container deployment similar to Selenium Grid, used in CI/CD or production.

The architecture follows the same principles in both modes — identical binaries, different configuration via environment variables.

---

## Core Goals

- Unified schema and ingestion protocol (protobuf)
- Real-time event streaming via NATS or Kafka
- Flexible storage backends (SQLite, Postgres, ClickHouse)
- Pluggable artifact storage (local, S3, MinIO)
- Extensible APIs for custom dashboards, analytics, and alerting
- Simple onboarding (AIO) + horizontal scalability (distributed)

---

## High-Level Flow

```text
Reporter (Playwright, Pytest, etc.)
        ↓
gRPC ingestion (observer server)
        ↓
NATS JetStream (event bus)
        ↓
Processor (event consumer → DB + object store)
        ↓
API / GraphQL
        ↓
Web UI
```
