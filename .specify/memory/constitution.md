# Observer Constitution

## Core Principles

### I. Architecture Boundaries (NON-NEGOTIABLE)

- Services are split: ingestion (gRPC → NATS), processor (NATS → DB + storage), API (DB + WebSocket), web UI (React).
- Ingestion MUST remain stateless and MUST NOT write to MongoDB.
- Processor MUST be the only service that persists events to MongoDB.
- API MUST be read-only against MongoDB.
- Legacy monolith compatibility MUST be preserved unless explicitly approved.

### II. Event-First Contract

- gRPC handlers MUST validate inputs and publish to NATS JetStream.
- All state mutations MUST be driven by NATS consumers.
- Event envelopes MUST preserve event type and timestamp.
- Unknown event types MUST be acked to avoid redelivery loops.

### III. MongoDB Safety Rules (NON-NEGOTIABLE)

- All nested updates MUST include root \_id in the filter.
- Use arrayFilters for nested updates; positional indexing is forbidden for attempts/steps.
- All writes MUST be idempotent and safe for redelivery.
- Schema changes MUST be additive and backward compatible.

### IV. Storage and Attachments

- Attachments MUST use the storage.Driver abstraction.
- Inline attachments MUST be base64-encoded and include content_encoding=base64.
- External storage MUST record storage_key, storage_uri, storage, size, uploaded_at.
- API retrieval MUST respect storage type (inline/local/s3/external).

### V. Logging and Errors

- Use slog only; log.Printf is forbidden.
- Library code MUST tolerate nil loggers via a no-op logger.
- Client validation failures MUST return InvalidArgument.
- Internal failures MUST return Internal and MUST NOT leak secrets.

### VI. Testing Discipline

- Contract changes MUST include tests covering ingestion → NATS → processor → DB.
- gRPC tests MUST use bufconn where practical.
- NATS integration tests MUST be gated by NATS_TEST_URL.

### VII. Contract Versioning (NON-NEGOTIABLE)

- Event schemas MUST be versioned (event_version or envelope.version).
- Breaking changes are forbidden; if unavoidable, introduce a new event type/version and support both.
- Producers MUST NOT remove or rename fields; deprecations MUST be additive (keep old fields until removal is explicitly approved).
- Consumers MUST be tolerant of unknown fields and missing optional fields.

### VIII. Delivery Semantics

- Assume at-least-once delivery from NATS; duplicates MUST not change final state.
- Processing MUST tolerate out-of-order events; do not assume monotonic timestamps.
- Idempotency MUST be based on stable event identifiers (event_id / run_id + sequence), not timestamps alone.
- If ordering is required within a partition (e.g. per run_id), consumers MUST enforce it explicitly.

### IX. MongoDB Data Model Invariants

- TestRun documents MUST stay within MongoDB document size limits; avoid unbounded arrays without retention/compaction rules.
- Any new query pattern MUST include an index plan (which fields, why).
- API queries MUST be bounded (limit, projection); avoid full-document fetches unless required.

### X. Migrations and Rollouts

- Schema changes MUST include a forward-compatible read path first, then write path, then (optional) backfill.
- Backfills MUST be resumable and safe to re-run.
- New fields MUST default safely when missing (zero values must not change meaning).

### XI. Security and Data Handling (NON-NEGOTIABLE)

- Do not log full event payloads or attachment contents.
- Never persist secrets/tokens in MongoDB or logs.
- Attachment URIs and storage credentials MUST be treated as sensitive.
- Any externally accessible endpoint MUST validate authorization and avoid IDOR (direct object reference) risks.

### XII. Observability

- All services MUST emit structured logs with run_id and event_id where available.
- Processor MUST record processing outcomes (success/failure, latency) in metrics and/or persisted status.
- NATS consumer lag/backlog visibility MUST be maintained when changing consumer behavior.

### XIII. NATS Consumer Rules

- Ack/Nak behavior MUST preserve existing retry/backoff semantics.
- Poison messages MUST not cause infinite redelivery loops; route to DLQ or mark terminal failure per existing pattern.
- Consumers MUST not block the whole stream on a single bad message.

### XIV. Dependency and Refactor Constraints

- Do not introduce new major libraries/frameworks without explicit approval.
- Avoid refactors that are not required for the spec; prefer local, minimal change sets.
- Keep service boundaries intact; do not merge services or bypass NATS to “simplify.”

## Development Workflow Rules

### Code Changes

- Prefer minimal diffs and preserve public APIs.
- Do not reformat unrelated code.
- Maintain attempt-based retries and legacy fields for compatibility.

### Web UI Rules

- UI MUST use apiUrl helper for API access.
- Attachments MUST be previewable via /api/attachments/{storageKey}.
- UI MUST be nil-safe for partial data.

### Documentation

- Update docs/architecture/ when changing architecture, data flow, or storage behavior.
- Keep documentation aligned with implementation changes.

## Quality Gates

- Backend changes MUST pass go test ./tests.
- Web changes MUST pass npm run build.
- Lint/vet failures MUST be resolved before merging.

## Developer Experience

- Local dev MUST use existing docker-compose / dev scripts; do not introduce parallel tooling.
- Configuration MUST come from env vars and existing config loaders; do not add new config systems.
- New env vars MUST be documented in the existing place (README / docs/config).

## Governance

- This constitution overrides all other guidance unless explicitly amended.
- Amendments require updating this file and documenting rationale in the PR description.

**Version**: 1.0.0 | **Ratified**: 2026-02-01 | **Last Amended**: 2026-02-01
