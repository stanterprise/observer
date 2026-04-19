# Relational Execution Implementation Checklist

This checklist operationalizes the migration plan in `docs/RELATIONAL_EXECUTION_MIGRATION_PLAN.md`.

Use this as the execution tracker for implementation, rollout, and cutover.

## 0. Decision Gates (Confirmed ✅)

✅ **PostgreSQL deployment (DECIDED):**

- AIO image: PostgreSQL embedded alongside MongoDB.
- Distributed mode: PostgreSQL connection string configured at startup.

✅ **Object storage (DECIDED: Phase 1):**

- Already implemented in `pkg/storage` for attachments (>= 100KB).
- Extend to step payloads: use same infrastructure and drivers (local, S3).
- Threshold: ~4MB default (configurable).

✅ **Terminal status taxonomy (DECIDED):**

- Terminal: `PASSED`, `FAILED`, `TIMED_OUT`, `CANCELLED`, `ABORTED`, `SKIPPED`.
- Active (non-terminal): `RUNNING`, `IN_PROGRESS`.

✅ **MongoDB TTL (DECIDED):**

- Default: 15 minutes (env var `MONGO_STEP_BUFFER_TTL`).

✅ **Backfill policy (DECIDED):**

- No backfill; new schema applies prospectively on branch cutover.

## 0.1 Important Scope Clarification: PostgreSQL Schema Initialization ≠ Data Migration

**This plan only applies to NEW runs ingested after the feature branch is deployed:**

- PostgreSQL tables are created at startup (idempotent `CREATE TABLE IF NOT EXISTS`).
- **No backfill or transformation of existing MongoDB runs occurs.**
- Existing MongoDB run documents remain in the database as read-only historical records.
- The API serves both data sources correctly:
  - New runs: written to PostgreSQL, read from PostgreSQL (active) or PostgreSQL (terminal).
  - Legacy runs: remain in MongoDB, served from MongoDB for backward compatibility.
- Run isolation is enforced: PostgreSQL runs are independent logical entities.

---

## 1. Foundation and Configuration

### 1.1 Configuration

- [ ] Add PostgreSQL connection config (`POSTGRES_DSN`, pooling, timeouts).
- [ ] Add `MONGO_STEP_BUFFER_TTL` env var (default: 15 minutes).
- [ ] Add object storage size threshold for step payloads (default: ~4MB).
- [x] Add boot-time validation that PG DSN is reachable (ping on connect).
- [ ] Add safe defaults for local/dev mode (POSTGRES_DSN optional; MongoDB-only fallback).

### 1.2 PostgreSQL integration

- [x] Add PostgreSQL connection module with pooling and health checks (`internal/database/postgres.go`).
- [x] Add graceful shutdown integration for PG clients (`defer pgDB.Close()` in processor).
- [ ] Integrate with existing observability for query performance.

### 1.3 Telemetry scaffolding

- [ ] Add counters for step buffer create/update/delete.
- [ ] Add counters for flush success/failure/retry.
- [ ] Add counters for reconciliation action types.
- [ ] Add histograms for flush latency and active step read latency.
- [ ] Add gauges for active buffer count and stale flush count.

## 2. PostgreSQL Data Layer

### 2.1 Connectivity and lifecycle

- [x] Add PostgreSQL connection module with pooling and health checks.
- [x] Add graceful shutdown integration for PG clients.
- [ ] Add configuration options for connection pool sizes and timeouts (via DSN + pgxpool config).

### 2.2 Schema definition and initialization

- [x] Define `runs` table schema (DDL or Go structs).
- [x] Define `run_shards` table schema.
- [x] Define `suites` table schema.
- [x] Define `tests` table schema.
- [x] Define `test_attempts` table schema with unique constraint on `(test_id, attempt_index)`.
- [x] Implement idempotent schema initialization in PG connection module: `CREATE TABLE IF NOT EXISTS` (no backfill of legacy MongoDB data).
- [x] Add indexes for query optimization: (run_id), (suite_id), (test_id), (status, created_at).

### 2.3 Repository interfaces and implementations

- [ ] Define PG repository interfaces for runs/suites/tests/test_attempts.
- [ ] Implement idempotent upsert semantics for event redelivery safety (ON CONFLICT DO UPDATE).
- [ ] Implement attempt begin/end transitions with explicit state machine checks (NOT IN terminal statuses guard).
- [x] Implement read queries for dashboard and detail endpoints (GetRun, ListRuns, GetSuite, ListTestsBySuite, etc.).

## 3. Shard Stitching and Logical Run Identity

- [ ] Define canonical logical run key composition.
- [ ] Implement key derivation in ingestion/processor path.
- [x] Implement run shard attach/update behavior in PG.
- [ ] Implement run completion policy for strict + fallback timeout modes.
- [ ] Add reconciliation for orphaned/incomplete shard groups.
- [ ] Add tests for split-shard inputs representing one logical run.

## 4. MongoDB Live Step Buffer

### 4.1 Collection and indexes

- [ ] Create `live_step_buffers` collection initialization (`LiveStepBuffersCollection()` accessor).
- [ ] Add TTL index on `ttl_at` with `expireAfterSeconds` derived from `MONGO_STEP_BUFFER_TTL`.
- [ ] Add supporting index for run-end sweep (`run_id`).

### 4.2 Document contract

- [ ] Implement document shape with `_id=test_id` (unique per active test).
- [ ] Add `attempt_index` as a regular attribute (not part of ID).
- [ ] Add `status` field (`active | flush_in_progress`).
- [ ] Add `first_event_at`, `last_event_at`, `flush_started_at` for tracking.
- [ ] Add `ttl_at` timestamp for TTL cleanup.

### 4.3 Write path

- [ ] Implement create/reset behavior at attempt start, setting `ttl_at = now() + MONGO_STEP_BUFFER_TTL`.
- [ ] Implement atomic step begin/step end mutations using MongoDB `$set` and `$push` (Phase 5).
- [ ] Enforce invariants on attempt index progression in code (Phase 5).
- [ ] Handle duplicate NATS event deliveries idempotently via event key (Phase 5).

## 5. Flush Protocol (Mongo -> PG) and Object Storage Integration

- [ ] Implement transition to `flush_in_progress=true` prior to finalization write.
- [ ] Implement final payload assembly from Mongo step tree (serialize to JSON).
- [ ] Implement payload size guard before PG write (threshold: ~4MB default, configurable).
- [ ] Write to `test_attempts.steps` JSONB when under threshold.
- [ ] When above threshold: compress payload, use existing `pkg/storage` drivers (local, S3) to persist, store pointer in `test_attempts.steps_ref`.
- [ ] Confirm PG transaction success before deleting Mongo buffer.
- [ ] Implement retry-safe behavior on flush failure: clear `flush_in_progress`, retry via reconciliation.
- [ ] Emit structured logs with attempt IDs, flush outcome, flush latency, and storage location.

## 6. Reconciliation and Recovery

### 6.1 Periodic reconciliation worker

- [ ] Implement periodic scan for stale `flush_in_progress` buffers.
- [ ] Implement retry for failed/stuck flushes.
- [ ] Implement orphan buffer detection and handling.

### 6.2 Run-end sweep

- [ ] Implement run-end triggered sweep for non-terminal attempts.
- [ ] Force finalize timed-out or missing-end attempts with diagnostics.
- [ ] Ensure test and attempt statuses are consistent after sweep.

### 6.3 Safety and idempotency

- [ ] Ensure reconciliation actions are idempotent.
- [ ] Ensure no duplicate finalization writes for same attempt.
- [ ] Add safeguards against deleting buffer before durable final write.

## 7. API and WebSocket Read Routing

### 7.1 REST API routing

- [ ] Add attempt-aware endpoint(s) if missing.
- [ ] Route active attempts to Mongo live buffer.
- [ ] Route terminal attempts to PG (`steps` or `steps_ref`).
- [ ] Add fallback behavior for missing sources with clear error states.

### 7.2 WebSocket behavior

- [ ] On subscribe, send active snapshot from Mongo when attempt is active.
- [ ] Continue streaming deltas from event path.
- [ ] Emit terminal transition message when attempt finalizes.
- [ ] Ensure post-terminal fetches resolve from PG.

## 8. Legacy Path Deactivation

- [ ] Disable oversized raw-message retention append behavior in processor.
- [ ] Stop creating new giant run documents for retention.
- [ ] Ensure read-only compatibility for existing historical data (no breaking changes).
- [ ] Document deprecation timeline for legacy path.

## 9. Testing Matrix

### 9.1 Unit tests

- [ ] Buffer lifecycle tests (create/update/finalize/delete).
- [ ] Attempt index invariants and gap handling tests.
- [ ] Flush failure/retry state transition tests.
- [ ] Reconciliation idempotency tests.

### 9.2 Integration tests

- [ ] End-to-end flow: begin -> steps -> end -> PG finalized -> Mongo deleted.
- [ ] Crash between `flush_in_progress` and PG commit.
- [ ] Missing test-end with run-end sweep recovery.
- [ ] Duplicate NATS event delivery handling.

### 9.3 Performance/load tests

- [ ] Simulate large runs at 50MB payload profile.
- [ ] Simulate large runs at 250MB payload profile.
- [ ] Simulate high-step cardinality nested scenarios.
- [ ] Verify p95 latencies and flush success SLOs.

## 10. Testing and Validation (Pre-merge)

Since this is a feature branch with complete cutover on merge (no incremental rollout):

### 10.1 Integration testing

- [ ] Run full test suite against new PG + Mongo architecture locally.
- [ ] Run load tests with high-step-count scenarios (50MB, 250MB, 1GB profiles).
- [ ] Validate all SLOs before merging to master.

### 10.2 Production observability setup (post-merge)

- [ ] Dashboard: active buffers, flush latency, flush failures, stale flush count.
- [ ] Dashboard: NATS consumer lag and processing throughput.
- [ ] Alert on stale flush growth beyond threshold.
- [ ] Alert on reconciliation forced-finalization spikes.
- [ ] Alert on any write failures to PG or object storage.

## 11. Cutover (Merge to Master – Full Deployment)

- [ ] All 8 branch implementation phases complete and tested.
- [ ] Verify all tests pass (unit, integration, load, chaos).
- [ ] Confirm active-step reads served by Mongo buffer (active terminal status).
- [ ] Confirm terminal-step reads served by PG (post-terminal status).
- [ ] Remove legacy raw-messages oversized retention code path (no longer needed).
- [ ] Update deployment documentation for PostgreSQL configuration in AIO and distributed modes.
- [ ] Code review passed; ready for master merge.
- [ ] **No feature flags** — merge commits full new architecture; previous code removed.

## 12. Exit Criteria

- [ ] Zero 16MB document-limit errors in production.
- [ ] Flush success rate >= 99.99%.
- [ ] Active and terminal read p95 latency targets met.
- [ ] Reconciliation auto-resolves stale/orphan cases.
- [ ] Team sign-off before master merge.

## 13. Execution Sequence (Branch Implementation)

1. Foundation + PG schema + Repositories.
2. Logical run stitching + Attempt state machine.
3. Mongo live buffer + TTL.
4. Flush protocol + Object storage integration (via pkg/storage).
5. Reconciliation loop.
6. API/WebSocket read routing.
7. Full integration testing + Load tests.
8. Code review + Master merge.
