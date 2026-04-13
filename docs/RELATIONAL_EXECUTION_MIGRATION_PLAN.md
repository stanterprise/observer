# Relational Execution Migration Plan

## 1. Purpose and Scope

This plan defines a full migration from the current MongoDB-centric run document model to a relational execution model centered on PostgreSQL, while retaining MongoDB as an ephemeral live-step cache for in-progress tests.

**Implementation approach:** Feature branch with complete cutover. No feature flags or incremental rollout—this is a clean migration to a new architecture, merged to master once ready.

The effort addresses these core goals:

- Eliminate MongoDB 16MB document-limit failures for large runs.
- Support runs with tens to hundreds of MB of step/log payload.
- Preserve low-latency live step updates during test execution.
- Keep run/suite/test hierarchy as the product's core mental model.
- Correctly handle retries, shard stitching, and catastrophic test termination.
- Extend existing object storage support to step payloads and large JSONB blobs.

Out of scope for this phase:

- Full historical analytics warehouse build-out.
- Replacing NATS as ingestion backbone.
- Full-text log search platform adoption (can be Phase 2+).
- **Backfilling existing MongoDB runs into PostgreSQL.** PostgreSQL schema is for NEW runs only; legacy MongoDB documents remain read-only.

---

## 1.1 Data Processing Boundary

**Important:** This is a **prospective schema change**, not a data migration:

- PostgreSQL and the new live-step-buffer pattern apply **only to runs ingested after the feature branch is deployed**.
- Existing MongoDB run documents remain in the database as read-only historical records.
- The API continues to serve both sources: PostgreSQL (new runs), MongoDB (legacy runs).
- No backfilling or data transformation of existing runs occurs.

---

## 2. Target Architecture (High Level)

### 2.1 Logical Domain Model (unchanged)

- Run -> Suite -> Test -> Step
- Attachments/log blobs stored separately from core execution rows.

### 2.2 Physical Storage Model (new)

- PostgreSQL (source of truth for completed execution state):
  - runs, run_shards, suites, tests, test_attempts, attachments metadata.
- MongoDB (ephemeral live cache only):
  - one active step buffer document per test (not per attempt ID in `_id`).
- Object storage (optional in Phase 1, recommended in Phase 2):
  - oversized step payloads/log bundles and large attachment bodies.
- NATS JetStream (existing):
  - event ingestion and delivery backbone.

### 2.3 Source-of-truth routing

- Test attempt active/in-progress:
  - steps read from MongoDB live buffer.
- Test attempt terminal (passed/failed/timed_out/cancelled):
  - steps read from PostgreSQL `test_attempts.steps` JSONB or `steps_ref` pointer.

---

## 3. Core Design Decisions

## D1. Identity and retries

- `test_attempts` become first-class rows.
- `attempt_index` remains an attribute, not part of MongoDB buffer `_id`.
- MongoDB live buffer `_id` is `test_id` (one active attempt invariant).
- Gaps in attempt indexes are tolerated and validated in code.

## D2. Step storage policy

- In-progress steps: mutable nested JSON in MongoDB buffer.
- Completed attempt steps: immutable JSONB in PostgreSQL.
- Introduce a size guard at flush time:
  - if payload exceeds threshold, write compressed blob to object storage and persist pointer in `steps_ref`.

## D3. Flush and deletion protocol

Never delete MongoDB live buffer before PostgreSQL write confirmation.

Protocol:

1. Mark `flush_in_progress=true` in MongoDB.
2. Write final step payload to PostgreSQL (`steps` JSONB or `steps_ref`).
3. Confirm transaction success.
4. Delete MongoDB buffer document.
5. On failure, clear/recover flush flag and retry via reconciliation.

## D4. Reconciliation and failure semantics

- Reconciliation runs at:
  - test end,
  - run end sweep,
  - periodic background scan.
- Handles:
  - missing end events,
  - stale `flush_in_progress`,
  - orphaned live buffers,
  - PG state mismatch.

## D5. Shard stitching

- Introduce canonical logical run identity in PostgreSQL.
- Persist shard membership and run completion policy in `run_shards`.
- Parent run closes only when shard completion criteria are met.

---

## 4. Data Model Blueprint

## 4.1 PostgreSQL tables

### runs

- `id` (PK)
- `logical_run_key` (unique)
- `source`, `project`, `pipeline`, `branch`, `commit_sha`
- `status`
- `started_at`, `finished_at`
- `metadata` JSONB

Indexes:

- unique on `logical_run_key`
- btree on `(started_at desc)`
- btree on `(status, started_at desc)`

### run_shards

- `id` (PK)
- `run_id` (FK runs.id)
- `shard_key`
- `shard_index`
- `shard_count_expected` nullable
- `status`
- `started_at`, `finished_at`

Indexes:

- unique `(run_id, shard_key)`
- btree `(run_id, status)`

### suites

- `id` (PK)
- `run_id` (FK)
- `external_suite_id`
- `name`, `status`
- `started_at`, `finished_at`
- `metadata` JSONB

Indexes:

- btree `(run_id)`
- btree `(run_id, status)`

### tests

- `id` (PK)
- `suite_id` (FK)
- `external_test_id`
- `name`, `status`
- `attempt_count`
- `started_at`, `finished_at`
- `metadata` JSONB

Indexes:

- unique `(suite_id, external_test_id)`
- btree `(suite_id, status)`

### test_attempts

- `id` (PK)
- `test_id` (FK)
- `attempt_index`
- `status`
- `started_at`, `finished_at`
- `steps` JSONB nullable
- `steps_ref` text nullable
- `step_count`
- `duration_ms`
- `failure_reason` text nullable
- `metadata` JSONB

Constraints/indexes:

- unique `(test_id, attempt_index)`
- check only one of `steps` or `steps_ref` is populated for terminal attempts
- btree `(test_id, attempt_index)`
- btree `(status, finished_at desc)`

### attachments

- `id` (PK)
- `test_attempt_id` (FK)
- `step_id` nullable
- `kind`, `name`, `content_type`
- `size_bytes`
- `storage_key`
- `checksum`
- `created_at`

Indexes:

- btree `(test_attempt_id)`
- btree `(created_at desc)`

## 4.2 MongoDB ephemeral collection

Collection: `live_step_buffers`

Document shape:

- `_id`: `test_id`
- `test_id`
- `run_id`
- `attempt_index`
- `status`: `active | flush_in_progress`
- `steps`: nested step tree JSON
- `first_event_at`, `last_event_at`
- `flush_started_at` nullable
- `ttl_at`

Indexes:

- unique `_id`
- TTL index on `ttl_at`
- optional secondary index on `(run_id)` for run-end sweep

---

## 5. Event and State Flows

## 5.1 Test begin

1. Upsert `tests` and `test_attempts` row in PG.
2. Create or reset MongoDB buffer `_id=test_id` for `attempt_index`.
3. Initialize buffer with empty steps root and timing metadata.

## 5.2 Step begin/step end

1. Persist canonical event effects to PG lightweight counters/state where needed.
2. Mutate live step tree in MongoDB buffer atomically.
3. Broadcast real-time updates via WebSocket.

## 5.3 Test end (normal flush path)

1. Resolve Mongo buffer by `test_id`.
2. Validate `attempt_index` match.
3. Set `flush_in_progress=true`, `flush_started_at=now`.
4. Serialize step tree, apply size guard.
5. Write final state into PG `test_attempts` in one transaction.
6. Delete Mongo live buffer on confirmed success.
7. Mark attempt and test terminal statuses.

## 5.4 Catastrophic failure path

Triggers:

- timeout,
- worker crash,
- missing terminal event,
- interrupted stream.

Handling:

1. Reconciliation identifies stale active attempts.
2. Attempt forced flush from Mongo buffer if available.
3. If no buffer, finalize with partial/error status and diagnostics.
4. Emit reconciliation telemetry and alert if thresholds exceeded.

---

## 6. API and Read Path Strategy

## 6.1 API contracts

- Existing run list/detail endpoints remain stable where possible.
- Add explicit attempt endpoints:
  - list attempts for a test,
  - fetch attempt details,
  - fetch attempt steps.

## 6.2 Source routing rules

- If attempt status is terminal:
  - read from PG (`steps` or dereference `steps_ref`).
- Else:
  - read from Mongo `live_step_buffers`.

## 6.3 WebSocket behavior

- On subscribe to active test:
  - send current Mongo snapshot first,
  - then stream deltas from events.
- On terminal transition:
  - send finalization event,
  - subsequent fetches served from PG.

---

---

## 8. Performance and Capacity Planning

## 8.1 Benchmarks to run

- Ingest load:
  - sustained events/sec with mixed run sizes.
- Active run live reads:
  - p95 latency of active test step retrieval.
- Finalization throughput:
  - tests/minute flush capacity.
- Large run scenarios:
  - 50MB, 250MB, 1GB logical run simulations.

## 8.2 SLO targets (initial draft)

- p95 active step fetch: < 200ms.
- p95 terminal attempt fetch: < 300ms.
- flush success rate: > 99.99%.
- orphan buffer rate: < 0.1% of attempts.
- zero document-size hard failures in production.

## 8.3 Guardrails

- Max step payload threshold before object storage offload.
- Max flush concurrency limit to protect PG.
- Mongo TTL > max expected test duration + reconciliation margin.

---

## 9. Reliability and Data Safety

## 9.1 Idempotency

- Event handlers must be idempotent by event key.
- Duplicate terminal events must not duplicate flush writes.

## 9.2 Exactly-once vs at-least-once

- System remains at-least-once at transport level.
- Application-level idempotency gives effectively-once outcomes.

## 9.3 Failure matrix (must be tested)

- Processor restart during active test.
- Crash after setting `flush_in_progress` but before PG commit.
- PG unavailable while Mongo available.
- Mongo unavailable while PG available.
- NATS redelivery storms.
- Run-end arrives with missing test-end for subset.

---

## 10. Operational Plan

## 10.1 Testing and Validation (Pre-merge)

All validation occurs on the feature branch before merge to master:

1. **Dev environment** with synthetic high-step workloads.
2. **Staging environment** with end-to-end test pipelines.
3. **Load testing** with 50MB, 250MB, 1GB+ run profiles.
4. **Chaos testing** covering failure scenarios (PG down, Mongo down, mid-flush crash, NATS redelivery).
5. All SLOs must pass; error budget exhausted → fix before merge.

## 10.2 Runtime dashboards (for post-deployment)

Once deployed to production after master merge, track at minimum:

- Active Mongo buffer count.
- Flush latency (p50, p95, p99) and failure rate.
- Reconciliation actions/minute.
- PostgreSQL write TPS, lock waits, connection pool utilization.
- NATS consumer lag.
- Orphan buffer detection and cleanup rate.

## 10.3 Alerting (for production post-deployment)

Configure alerts for:

- Stale `flush_in_progress` older than timeout threshold.
- Orphan buffer growth rate > threshold.
- PostgreSQL transaction retry spike.
- Reconciliation forced-finalization spike (indicates data quality issues).
- Mongo buffer memory pressure (if tracking collection size).

---

## 11. Implementation Sequence (Branch-based, Single Cutover)

Implementation occurs on a feature branch (`feature/relational-execution`) in sequential phases. Each phase builds on the previous; they are **not** optional or feature-flagged. The full branch is merged to master once all phases pass testing.

### Phase 1: Foundation and PostgreSQL Connectivity

- Define PostgreSQL schema for all tables (runs, shards, suites, tests, test_attempts) as DDL or Go structs.
- Implement PostgreSQL connection module (`internal/database/postgres.go`) with idempotent schema initialization (**create tables if not exist; no backfill of legacy data**).
- Add PG repository interface and base operations (create test_run, get test_run, etc.).
- Verify PG connects in dev, staging, and AIO/distributed configs.
- No write path changes yet; all writes still go to MongoDB.
- Legacy MongoDB run documents remain untouched and read-only.

### Phase 2: PostgreSQL Schema and Repositories

- Implement idempotent upsert repositories for all PG tables (runs, shards, suites, tests, test_attempts).
- Add integration tests for each repository method.
- Verify schema handles duplicate begins, test_id collisions, resequenced attempt_indexes.
- Prepare for Phase 3 (begins writing to PG in parallel).

### Phase 3: MongoDB live_step_buffers and Shard Stitching

- Implement MongoDB `live_step_buffers` repository with upsert, read, delete, TTL management.
- Implement run-shard stitching logic (map multi-shard ingestion to canonical run).
- Add tests for per-test buffer mutation and shard discovery.
- Verify MongoDB TTL deletion and Mongo buffer consistency under replay.

### Phase 4: Dual-write Testing (MongoDB ↔ PostgreSQL parity checks)

- Begin dual-writing terminal events to both MongoDB (legacy) and PostgreSQL.
- Add parity checks in tests: verify PG and Mongo states match for terminal data.
- Run end-to-end test scenarios (begin → steps → end) and validate both stores.
- Catch schema/mapping mismatches before Phase 5.

### Phase 5: Flush Protocol and Two-phase Commit

- Implement flush handler: read Mongo buffer → serialize → size check → write PG or object storage.
- Implement state machine: active → flush_in_progress → (success: delete Mongo; failure: retry).
- Add object storage integration for payloads > ~4MB (reuse existing `pkg/storage` drivers).
- Add idempotency by flush ID and event key to handle retries safely.
- Test crashes and partial failures during flush.

### Phase 6: Reconciliation and Orphan Recovery

- Implement reconciliation worker: scan for stale `flush_in_progress` and orphaned buffers.
- Implement failed-flush recovery and forced-finalization for orphans.
- Add run-end sweep: ensure no active buffers outlive their run_end event by reconciliation window.
- Test recover scenarios (PG unavailable, mid-flush restart, duplicate run-ends).

### Phase 7: API Read Path and Status-based Routing

- Update API read handlers to route by attempt terminal status:
  - Active (RUNNING, IN_PROGRESS) → read Mongo buffer.
  - Terminal (others) → read PostgreSQL.
- Update WebSocket handler to route events correctly.
- Verify API backward compatibility for existing callers.
- Test read consistency (active state transitions to terminal, verify read path switches).

### Phase 8: Comprehensive Testing and Performance Validation

- Run load tests with 50MB, 250MB, 1GB+ logical runs.
- Run chaos tests (PG down, concurrent flushes, NATS redelivery storms).
- Verify SLOs under load: p95 latency, flush throughput, error rate.
- Verify orphan rate stays < threshold.
- Run multi-day soak test for reliability.

### Phase 9: Final Cutover to Master

- All phases complete. All tests passing. SLOs validated.
- Merge feature branch to master.
- Deploy to production (no feature flags, full cutover to new system).
- Monitor SLOs and orphan rate post-deployment.

---

## 12. Decision Gates (Confirmed)

✅ **PostgreSQL deployment model (DECIDED):**

- AIO image: PostgreSQL embedded alongside MongoDB (optional external connection override).
- Distributed mode: PostgreSQL connection configured at startup.
- Rationale: Simplifies AIO deployment; distributed mode users already manage connections.

✅ **Object storage (DECIDED: Phase 1):**

- Object storage is already implemented in `pkg/storage` for attachments (>= 100KB).
- Extend same infrastructure to step payloads when flushing to PG.
- Size threshold at flush: payload exceeds ~4MB → redirect to object storage, store pointer in `steps_ref`.
- Supported backends: local (file), S3, extensible.

✅ **Terminal status taxonomy (DECIDED):**

- Terminal statuses: `PASSED`, `FAILED`, `TIMED_OUT`, `CANCELLED`, `ABORTED`, `SKIPPED`.
- Non-terminal (active): `RUNNING`, `IN_PROGRESS`.
- Rationale: Anything other than `RUNNING` or `IN_PROGRESS` is terminal.

✅ **MongoDB live buffer TTL (DECIDED):**

- Default: 15 minutes.
- Configurable via `MONGO_STEP_BUFFER_TTL` env var.
- Rationale: 15 min > typical test window; auto-cleanup for orphaned buffers.

✅ **Backfill policy (DECIDED):**

- No backfill of legacy run documents.
- New schema applies prospectively on migration branch cutover.

---

## 13. Acceptance Criteria

The implementation is complete when all conditions are true:

- ✅ PostgreSQL schema initialized at startup (new tables created; no backfill of legacy MongoDB data).
- ✅ **New runs** are written to PostgreSQL; **legacy MongoDB runs remain read-only**.
- ✅ MongoDB `live_step_buffers` collection with TTL working correctly for new attempts.
- ✅ Dual-write parity validated: terminal data in PG and legacy Mongo match (for new runs only).
- ✅ Flush protocol successfully terminates active buffers and persists to PG.
- ✅ Object storage offload works for payloads > ~4MB.
- ✅ Reconciliation worker detects and recover stale flushes and orphans.
- ✅ API read path correctly switches between Mongo (active) and PG (terminal) by status.
- ✅ API serves both new runs (PG) and legacy runs (Mongo) correctly.
- ✅ All end-to-end tests pass (begin → steps → end; crash scenarios; NATS redelivery).
- ✅ Load tests pass: 50MB, 250MB, 1GB+ new runs complete without size errors.
- ✅ SLOs met: p95 latency < 200ms (active), < 300ms (terminal); flush success > 99.99%.
- ✅ Orphan buffer rate < 0.1% in load tests.
- ✅ Feature branch merged to master; deployed to production.
- ✅ Post-deployment monitoring confirms SLOs and orphan rate for new runs in production.

---

## 14. Suggested Immediate Next Actions

1. **Create feature branch**: `git checkout -b feature/relational-execution`
2. **Phase 1 (Foundation)**:
   - Define PostgreSQL schema for all tables (runs, shards, suites, tests, test_attempts) as DDL or Go struct tags.
   - Implement `internal/database/postgres.go` connection helper with idempotent schema initialization (paralleling `internal/database/mongodb.go`).
   - Add PostgreSQL connection configuration (embedded in AIO Dockerfile, external connection string in distributed mode).
   - Test in dev and Docker Compose configurations.
3. **Phase 2 (Repositories)**:
   - Implement idempotent upsert repository methods for each table.
   - Add comprehensive repository tests (duplicate keys, resequenced attempts, etc.).
4. **Synthetic load profile**:
   - Build high-step run simulator (50MB, 250MB, 1GB profiles).
   - Use to validate both MongoDB and PostgreSQL paths independently.
5. **Parallel workstreams**:
   - Schema finalization and code implementation.
   - Docker/Helm updates for PostgreSQL embedding (AIO).
   - Monitoring dashboard scaffolding (pre-deployment readiness).
