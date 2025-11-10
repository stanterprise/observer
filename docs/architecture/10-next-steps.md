# Next Steps

## 1. Code Refactor
- Add Cobra subcommands: `ingest`, `processor`, `api`, `serve-aio`.

## 2. Event Bus Integration
- Integrate NATS JetStream as publisher/consumer.

## 3. Storage Layer
- Start with SQLite/local FS.
- Add Postgres + S3 support.

## 4. AIO Runtime
- Use s6-overlay to orchestrate `nats-server` + observer.

## 5. Compose Setup
- Add `aio` and `dist` profiles to `docker-compose.yaml`.

## 6. UI MVP
- React dashboard for runs, tests, and artifacts.

## 7. CI/CD Pipeline
- Multi-arch build, SBOM, Trivy scan, Helm packaging.

## 8. Observability
- Add OpenTelemetry traces and Prometheus metrics.
