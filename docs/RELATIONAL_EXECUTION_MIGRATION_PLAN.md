# Relational Execution Migration Plan

## 1. Purpose and Scope

This plan defines a full migration path from the current MongoDB-centric run document model to a relational execution model centered on PostgreSQL, while retaining MongoDB as an ephemeral live-step cache for in-progress tests.

The effort addresses these core goals:

- Eliminate MongoDB 16MB document-limit failures for large runs.
- Support runs with tens to hundreds of MB of step/log payload.
- Preserve low-latency live step updates during test execution.
- Keep run/suite/test hierarchy as the product’s core mental model.
- Correctly handle retries, shard stitching, and catastrophic test termination.
- Provide safe migration with minimal production risk.

Out of scope for this phase:

- Full historical analytics warehouse build-out.
- Replacing NATS as ingestion backbone.
- Full-text log search platform adoption (can be Phase 2+).

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

## 7. Migration Strategy

## Phase 0: Preparation

- Define SLOs and acceptance criteria.
- Add feature flags:
  - `PG_EXECUTION_ENABLED`
  - `MONGO_LIVE_STEPS_ENABLED`
  - `PG_FINALIZE_FROM_MONGO_ENABLED`
  - `OBJECT_STORAGE_STEPS_ENABLED`
- Add observability scaffolding before behavior changes.

## Phase 1: Introduce PostgreSQL sidecar writes

- Write runs/suites/tests/test_attempts in PG in parallel with existing flow.
- Keep MongoDB current source of truth for production reads.
- Validate parity with shadow checks.

## Phase 2: Enable Mongo live-step buffer mode

- Stop appending large raw message blobs to legacy run docs.
- Start writing active steps into `live_step_buffers`.
- Keep finalization disabled initially; compare behavior in dry-run mode.

## Phase 3: Enable flush-to-PG finalization

- On test end, flush Mongo buffer into PG attempt record.
- Enable API read routing by attempt status.
- Keep legacy reads as fallback behind feature flag.

## Phase 4: Reconciliation hardening

- Add periodic reconciliation worker.
- Implement stale flush recovery and orphan cleanup.
- Add run-end sweep for missed terminal events.

## Phase 5: Cutover and cleanup

- Switch primary reads to PG for terminal attempts.
- Disable legacy large run document retention path.
- Archive or remove obsolete code paths after stabilization window.

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

## 10.1 Rollout

1. Dev environment with synthetic high-step workloads.
2. Staging with canary pipelines.
3. Production canary by project/reporter subset.
4. Full rollout after error budget and SLO pass.

## 10.2 Runtime dashboards

Track at minimum:

- Active Mongo buffer count.
- Flush latency and failure rate.
- Reconciliation actions/minute.
- PG write TPS and lock waits.
- NATS consumer lag.
- API read path split (Mongo active vs PG terminal).

## 10.3 Alerting

- Stale `flush_in_progress` older than threshold.
- Orphan buffer growth.
- PG transaction retry spike.
- Reconciliation forced-finalization spike.

---

## 11. Work Breakdown and Sequencing

## Sprint 1: Foundations

- Add PG schema migrations and repository interfaces.
- Add feature flags and telemetry.
- Implement base PG write path for runs/suites/tests/test_attempts.

## Sprint 2: Live step buffer

- Implement Mongo `live_step_buffers` repository.
- Add active step mutation handlers.
- Add API read routing scaffolding.

## Sprint 3: Finalization and reconciliation

- Implement flush protocol and state transitions.
- Implement reconciliation worker and run-end sweep.
- Add end-to-end failure recovery tests.

## Sprint 4: Cutover and hardening

- Enable PG terminal read path.
- Disable legacy oversized retention path.
- Run load tests, tune indexes, verify SLOs.

## Sprint 5 (optional): Large payload offload

- Add object storage `steps_ref` support.
- Add compression, pointer dereference APIs.
- Tune retention tiers and storage lifecycle rules.

---

## 12. Open Decision Gates

Before implementation starts, confirm:

1. PostgreSQL hosting model:
   - managed service vs self-hosted.
2. Object storage in Phase 1 or Phase 2.
3. Terminal status taxonomy and timeout defaults.
4. Maximum expected test duration for TTL design.
5. Backfill policy for old runs:
   - none, partial, or full.

---

## 13. Acceptance Criteria

The migration is complete when all conditions are true:

- No production writes depend on oversized MongoDB run documents.
- Active test steps are served from MongoDB buffer reliably.
- Terminal attempt steps are persisted/read from PostgreSQL reliably.
- Reconciliation resolves missed-end and stale-flush scenarios automatically.
- Performance SLOs pass under representative high-volume workloads.
- Legacy retention path is disabled behind a removed feature flag.

---

## 14. Suggested Immediate Next Actions

1. Finalize decision gates in Section 12.
2. Create PG migration scripts for Section 4 tables.
3. Implement feature flags and telemetry scaffolding first.
4. Build a synthetic load profile that reproduces recent high-step sharded runs.
5. Start Sprint 1 with PG sidecar writes and parity checks.
