# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Initial public release
- All-in-One (AIO) deployment mode with embedded services
- Distributed deployment mode for production
- gRPC ingestion service for test event collection
- NATS JetStream-based event processing
- PostgreSQL persistence layer for run history
- MongoDB-backed live step buffering during in-flight execution
- REST API with WebSocket support for real-time updates
- React + TypeScript + Tailwind CSS web UI
- Docker and Kubernetes/Helm deployment support
- Comprehensive documentation (README, QUICKSTART, DEPLOYMENT, SECURITY)
- GitHub Codespaces support
- Apache 2.0 license

### Changed

- Helm chart documentation now reflects the distributed-first contract, downstream-managed exposure model, and `runtime.existingSecret` production flow.
- Chart examples now prefer immutable image tags and out-of-band runtime secret management instead of inline credentials in values files.
- Distributed runtime connection overrides via `distributed.*.env` are documented as unsupported and rejected during validation.

### Fixed

- The distributed migration hook no longer depends on a chart-managed runtime Secret that has not been created yet during install or upgrade.

### Removed

- Chart-managed ingress and Gateway API examples from Helm documentation. Exposure, TLS, and certificates are now documented as downstream infrastructure concerns.

### Architecture

- Phase 1: NATS JetStream publisher in ingestion service
- Phase 2: NATS JetStream consumer in processor service
- Phase 3: WebSocket real-time streaming and Web UI

## Notes

This changelog tracks major features and changes. For detailed commit history,
see the [GitHub repository](https://github.com/stanterprise/observer).
