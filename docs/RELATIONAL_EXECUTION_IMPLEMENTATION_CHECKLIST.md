# Relational Execution Implementation Checklist

This checklist operationalizes the migration plan in `docs/RELATIONAL_EXECUTION_MIGRATION_PLAN.md`.

Use this as the execution tracker for implementation, rollout, and cutover.

## 0. Decision Gates (Must Be Closed First)

- [ ] Confirm PostgreSQL hosting model (managed vs self-hosted).
- [ ] Confirm whether object storage offload is Phase 1 or Phase 2.
- [ ] Confirm terminal status taxonomy (passed, failed, timed_out, cancelled, aborted).
- [ ] Confirm max expected test duration for MongoDB TTL sizing.
- [ ] Confirm backfill policy (none, recent-only, full).
- [ ] Confirm SLO targets and load profile assumptions.

## 1. Foundation and Guardrails

### 1.1 Feature flags
- [ ] Add `PG_EXECUTION_ENABLED` flag.
- [ ] Add `MONGO_LIVE_STEPS_ENABLED` flag.
- [ ] Add `PG_FINALIZE_FROM_MONGO_ENABLED` flag.
- [ ] Add `OBJECT_STORAGE_STEPS_ENABLED` flag.
- [ ] Add startup logs printing effective values for all migration flags.

### 1.2 Config validation
- [ ] Add boot-time validation for incompatible flag combinations.
- [ ] Add boot-time validation that required DSNs/endpoints exist when flags are enabled.
- [ ] Add safe defaults for local/dev mode.

### 1.3 Telemetry scaffolding
- [ ] Add counters for step buffer create/update/delete.
- [ ] Add counters for flush success/failure/retry.
- [ ] Add counters for reconciliation action types.
- [ ] Add histograms for flush latency and active step read latency.
- [ ] Add gauges for active buffer count and stale flush count.

## 2. PostgreSQL Data Layer

### 2.1 Connectivity and lifecycle
- [ ] Add PostgreSQL connection module with pooling and health checks.
- [ ] Add graceful shutdown integration for PG clients.
- [ ] Add configuration options for connection pool sizes and timeouts.

### 2.2 Schema migrations
- [ ] Create migration for `runs` table.
- [ ] Create migration for `run_shards` table.
- [ ] Create migration for `suites` table.
- [ ] Create migration for `tests` table.
- [ ] Create migration for `test_attempts` table.
- [ ] Create migration for `attachments` metadata table.
- [ ] Create required indexes and uniqueness constraints.
- [ ] Add migration rollback scripts where applicable.

### 2.3 Repository interfaces and implementations
- [ ] Define PG repository interfaces for runs/suites/tests/test_attempts.
- [ ] Implement idempotent upsert semantics for event redelivery safety.
- [ ] Implement attempt begin/end transitions with explicit state machine checks.
- [ ] Implement read queries for dashboard and detail endpoints.

## 3. Shard Stitching and Logical Run Identity

- [ ] Define canonical logical run key composition.
- [ ] Implement key derivation in ingestion/processor path.
- [ ] Implement run shard attach/update behavior in PG.
- [ ] Implement run completion policy for strict + fallback timeout modes.
- [ ] Add reconciliation for orphaned/incomplete shard groups.
- [ ] Add tests for split-shard inputs representing one logical run.

## 4. MongoDB Live Step Buffer

### 4.1 Collection and indexes
- [ ] Create `live_step_buffers` collection initialization.
- [ ] Add TTL index on `ttl_at`.
- [ ] Add supporting index for run-end sweep (`run_id`) if needed.

### 4.2 Document contract
- [ ] Implement document shape with `_id=test_id` and `attempt_index` attribute.
- [ ] Add `status` and `flush_in_progress` fields.
- [ ] Add `first_event_at` and `last_event_at` fields.
- [ ] Add `flush_started_at` for stale flush recovery.

### 4.3 Write path
- [ ] Implement create/reset behavior at attempt start.
- [ ] Implement atomic step begin/step end mutations.
- [ ] Enforce invariants on attempt index progression.
- [ ] Handle duplicate deliveries idempotently.

## 5. Flush Protocol (Mongo -> PG)

- [ ] Implement transition to `flush_in_progress=true` prior to finalization write.
- [ ] Implement final payload assembly from Mongo step tree.
- [ ] Implement payload size guard before PG write.
- [ ] Write to `test_attempts.steps` JSONB when under threshold.
- [ ] Write to object storage + `test_attempts.steps_ref` when above threshold (if enabled).
- [ ] Confirm PG transaction success before deleting Mongo buffer.
- [ ] Implement retry-safe behavior when flush fails.
- [ ] Emit structured logs with attempt IDs and flush outcomes.

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

## 8. Legacy Path De-risking

- [ ] Add kill switch for legacy raw-message retention append behavior.
- [ ] Disable oversized legacy document growth path behind feature flag.
- [ ] Keep read-only compatibility for existing historical data during migration window.
- [ ] Document deprecation and removal timeline for legacy path.

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

## 10. Rollout Plan

### 10.1 Environment progression
- [ ] Deploy to dev with feature flags disabled by default.
- [ ] Enable PG sidecar writes in dev and validate parity.
- [ ] Enable Mongo live buffer in dev and validate API routing.
- [ ] Enable finalization in staging with canary load.
- [ ] Enable production canary for selected projects.
- [ ] Expand to full production after SLO/pass criteria.

### 10.2 Observability and alerts
- [ ] Dashboard: active buffers, flush latency, flush failures, stale flush count.
- [ ] Dashboard: NATS consumer lag and processing throughput.
- [ ] Alert on stale flush growth beyond threshold.
- [ ] Alert on reconciliation forced-finalization spikes.
- [ ] Alert on any document-size limit errors (should be zero).

## 11. Cutover and Cleanup

- [ ] Switch primary terminal-step read path to PG.
- [ ] Confirm active-step reads still served by Mongo.
- [ ] Disable legacy retention writes permanently.
- [ ] Remove dead code paths and obsolete flags.
- [ ] Update runbooks and deployment docs.
- [ ] Final post-cutover verification report.

## 12. Exit Criteria

- [ ] Zero 16MB document-limit errors in production over agreed observation window.
- [ ] Flush success rate meets target (>= 99.99%).
- [ ] Active and terminal read p95 latency targets are met.
- [ ] Reconciliation resolves stale/orphan cases automatically.
- [ ] Team sign-off from backend, ops, and product stakeholders.

## 13. Recommended Execution Order

1. Decision Gates -> Foundation -> PostgreSQL schema and repositories.
2. Logical run stitching and attempt state machine.
3. Mongo live buffer implementation.
4. Flush protocol and reconciliation.
5. API/WebSocket read routing.
6. Performance validation and canary rollout.
7. Legacy path shutdown and cleanup.
