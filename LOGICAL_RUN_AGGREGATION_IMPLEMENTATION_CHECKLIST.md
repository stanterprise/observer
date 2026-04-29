# Logical Run Aggregation Implementation Checklist

This checklist operationalizes [LOGICAL_RUN_AGGREGATION_IMPLEMENTATION_PLAN.md](LOGICAL_RUN_AGGREGATION_IMPLEMENTATION_PLAN.md).

Use this as the execution tracker for implementing logical run aggregation with execution-aware identity.

## 0. Decision Gates (Confirmed)

✅ `run_id` remains the logical aggregate run key.

✅ `execution_id` is the uniqueness-first execution scope key beneath `run_id`.

✅ Shared `execution_id` across sibling shard calls is optional batch association, not a correctness requirement or a separate identity layer.

✅ Phase 1 does not introduce an additional grouping identifier beyond `run_id` and `execution_id`.

✅ New writes become execution-aware; historical data remains readable without mandatory backfill.

✅ Missing `execution_id` is tolerated initially for backward compatibility, but mixed or conflicting usage must be classified.

## 0.1 Scope Clarification

- [ ] Keep the implementation additive and compatibility-preserving for legacy runs.
- [ ] Avoid any design that merges different `run_id` values into one logical run.
- [ ] Avoid making logical test identity depend on `execution_id`.
- [ ] Avoid modeling shard batch association as a separate first-class entity in Phase 1.
- [ ] Treat missing shared shard-batch association as reduced fidelity, not incorrectness.

## 1. External Contract and Reporter

### 1.1 Protobuf contract

- [ ] Add `execution_id` to all relevant event messages in the external `proto-go` repository.
- [ ] Add `execution_id` to any nested entity types that are the canonical carriers of run, suite, test, or step identity.
- [ ] Keep the field optional on the wire for backward compatibility.
- [ ] Document that full logical-run aggregation correctness requires consistent `execution_id` propagation.
- [ ] Release a new `proto-go` version that Observer can consume.
- [ ] Bump the `github.com/stanterprise/proto-go` dependency in [go.mod](go.mod).

### 1.2 Reporter generation rules

- [ ] Implement auto-generation of `run_id` for standalone runs when absent.
- [ ] Implement auto-generation of `execution_id` for standalone runs when absent.
- [ ] Implement shard-call behavior so a shard without explicit `execution_id` gets its own execution scope.
- [ ] Preserve the ability to share `run_id` across multiple shard calls to form one logical run.
- [ ] Preserve the ability to share `execution_id` across sibling shard calls as optional batch association.
- [ ] Log resolved `run_id` and `execution_id` values at reporter startup for diagnostics.

### 1.3 Reporter validation and tests

- [ ] Add reporter-side guardrails for obviously conflicting explicit `execution_id` reuse when detectable.
- [ ] Add tests for default UUID generation.
- [ ] Add tests for env-var override behavior.
- [ ] Add tests for shard behavior with distinct auto-generated `execution_id` values.
- [ ] Add tests for shard behavior with shared explicit `execution_id`.
- [ ] Add tests proving `execution_id` is present on every emitted event type.

## 2. Ingestion and Identity Plumbing

### 2.1 gRPC server acceptance

- [ ] Update validation in [pkg/server/server.go](pkg/server/server.go) to accept execution-aware payloads.
- [ ] Keep legacy payloads without `execution_id` accepted during the initial rollout.
- [ ] Ensure event payloads continue through NATS without dropping `execution_id`.

### 2.2 Shared identity helpers in the processor

- [ ] Add `extractExecutionID(...)` helpers in [pkg/consumer/nats_consumer.go](pkg/consumer/nats_consumer.go) or the appropriate shared helper location.
- [ ] Update step identity extraction to include `execution_id`.
- [ ] Update test identity extraction to include `execution_id` where concrete attempt identity depends on it.
- [ ] Update deferred-queue and reconciliation keys to include `execution_id`.
- [ ] Separate execution-scope lifecycle state from aggregate logical-run lifecycle state.

### 2.3 Consumer handler propagation

- [ ] Update [pkg/consumer/nats_run_handlers.go](pkg/consumer/nats_run_handlers.go) to extract and persist `execution_id`.
- [ ] Update [pkg/consumer/nats_test_handlers.go](pkg/consumer/nats_test_handlers.go) to propagate `execution_id` into test and attempt writes.
- [ ] Update [pkg/consumer/nats_step_handlers.go](pkg/consumer/nats_step_handlers.go) to use execution-aware step identity.
- [ ] Update [pkg/consumer/nats_suite_handlers.go](pkg/consumer/nats_suite_handlers.go) to propagate execution-aware suite identity.
- [ ] Update [pkg/consumer/nats_std_handlers.go](pkg/consumer/nats_std_handlers.go) so stdout and stderr events remain execution-aware.
- [ ] Update [pkg/consumer/event_classifier.go](pkg/consumer/event_classifier.go) to classify execution-aware, legacy, and ambiguous flows.

### 2.4 Compatibility and ambiguity classification

- [ ] Introduce a classification model for `execution-aware`, `legacy`, and `ambiguous` run ingestion.
- [ ] Detect mixed identity cases where some events in one execution scope have `execution_id` and others do not.
- [ ] Detect invalid `execution_id` reuse across conflicting unsharded executions under one `run_id`.
- [ ] Detect invalid `execution_id` reuse across conflicting shard batches when the resulting identity is ambiguous.
- [ ] Emit warnings or flags rather than hard failures in the initial rollout phase.

## 3. Relational Data Model and Repository

### 3.1 Schema changes

- [ ] Add a `RunExecution` relational model in [internal/models/relational.go](internal/models/relational.go).
- [ ] Add a backing `run_executions` table through the existing migration path.
- [ ] Add a unique constraint on `(run_id, execution_id)`.
- [ ] Add `execution_id` to execution-scoped tables that currently rely on run-only identity.
- [ ] Revisit uniqueness constraints for `run_shards`, `suites`, `test_attempts`, and any other execution-scoped rows.
- [ ] Add supporting indexes for run-level and execution-level lookup patterns.

### 3.2 Logical test identity

- [ ] Formalize a `logical_test_key` strategy that is stable across execution scopes.
- [ ] Ensure logical tests remain independent of `execution_id`.
- [ ] Prefer a stable external Playwright test ID when available.
- [ ] Define a deterministic fallback fingerprint when a stable external ID is unavailable.

### 3.3 Attempt identity

- [ ] Update attempt identity mapping in [internal/models/relational_mappers.go](internal/models/relational_mappers.go) to include `logical_test_key`, `execution_id`, and `source_attempt_index`.
- [ ] Replace or evolve the current uniqueness assumption on `(test_id, attempt_index)`.
- [ ] Ensure repeated logical tests across multiple execution scopes append attempts instead of colliding.

### 3.4 Repository write paths

- [ ] Update `internal/repository/postgres` run-start writes to upsert run-execution rows.
- [ ] Attach shard rows beneath the correct execution scope instead of only beneath `run_id`.
- [ ] Attach suite execution rows beneath the correct execution scope.
- [ ] Preserve logical test upserts independently of execution scope.
- [ ] Persist attempts with execution-aware identity.
- [ ] Derive aggregate run status from execution-scope status rather than a flat run-only model.

### 3.5 Legacy data tolerance

- [ ] Keep rows without `execution_id` readable.
- [ ] Decide whether legacy rows stay nullable or get an internal synthetic legacy execution marker at read time.
- [ ] Avoid forcing historical backfill in the initial implementation.

## 4. MongoDB Live Buffer and Reconciliation

### 4.1 Live buffer identity

- [ ] Update live step buffer identity in [internal/repository/mongodb/mongodb_step_buffer.go](internal/repository/mongodb/mongodb_step_buffer.go) to include `execution_id`.
- [ ] Ensure live step buffer identity also includes the logical or external test key needed to avoid cross-execution collisions.
- [ ] Update any helper that computes active test buffer IDs to stop assuming `run_id` plus test ID is globally unique.

### 4.2 Flush and cleanup paths

- [ ] Make all flush lookups execution-aware.
- [ ] Make all live-buffer delete paths execution-aware.
- [ ] Ensure one execution scope cannot overwrite or delete another execution scope’s active step state.

### 4.3 Reconciliation and run-completion logic

- [ ] Rework reconciliation inputs to operate at execution-scope granularity.
- [ ] Rework logical-run completion to derive from execution-scope completion.
- [ ] Ensure one execution ending does not finalize the logical run while other execution scopes remain active or unresolved.
- [ ] Make shard-sibling completeness checks conditional on data that actually supports that interpretation.
- [ ] Preserve correctness when shard calls do not share `execution_id`.

## 5. API, WebSocket, and UI

### 5.1 API response model

- [ ] Extend API responses in [pkg/api](pkg/api) to expose execution-scope summaries.
- [ ] Add execution-scope count and status information to run detail responses.
- [ ] Annotate attempt detail responses with `execution_id` and shard provenance where available.
- [ ] Keep response changes additive where possible for backward compatibility.

### 5.2 Query behavior

- [ ] Add query paths that load a logical run plus its execution scopes.
- [ ] Add query paths that load one logical test with attempts across multiple execution scopes.
- [ ] Add execution-scope filtering where it improves investigation workflows.

### 5.3 WebSocket propagation

- [ ] Update [pkg/websocket/proto_to_model.go](pkg/websocket/proto_to_model.go) and related websocket code to propagate `execution_id` in real-time payloads where needed.
- [ ] Ensure live updates for repeated logical tests remain distinguishable across execution scopes.

### 5.4 Web UI

- [ ] Add execution-aware summaries to run detail views in [web/src](web/src).
- [ ] Show when one logical run aggregates multiple execution scopes.
- [ ] Expose execution scope status, timing, and shard/batch hints where available.
- [ ] Update test detail views so attempts from multiple execution scopes render under one logical test.
- [ ] Surface `execution_id`, shard info, retry index, timing, and status for each attempt.

## 6. Observability and Rollout Controls

### 6.1 Diagnostics

- [ ] Add counters for generated versus explicit `execution_id` usage.
- [ ] Add counters for ambiguous execution reuse detections.
- [ ] Add counters or gauges for execution-scope creation count per logical run.
- [ ] Add logs or metrics for reconciliation actions taken at execution-scope level.
- [ ] Add diagnostics for logical test merges across execution scopes.
- [ ] Add a signal for legacy-mode ingestion volume.

### 6.2 Rollout policy

- [ ] Start with warnings and ambiguity flags rather than hard failures for invalid reuse.
- [ ] Document the conditions under which the system can later tighten invalid reuse to hard failures.
- [ ] Document operator expectations for mixed legacy and execution-aware traffic during transition.

## 7. Testing Matrix

### 7.1 Unit tests

- [ ] Add unit tests for execution-aware extraction helpers.
- [ ] Add unit tests for execution-aware deferred-queue keys.
- [ ] Add unit tests for logical test key derivation.
- [ ] Add unit tests for invalid `execution_id` reuse detection.
- [ ] Add unit tests for aggregate run status derivation from multiple execution scopes.
- [ ] Add unit tests for attempt ordering and merge behavior across execution scopes.

### 7.2 Integration tests

- [ ] Add an end-to-end test for one logical run with two unsharded execution scopes.
- [ ] Add an end-to-end test for a logical run where shard calls each get their own auto-generated `execution_id`.
- [ ] Add an end-to-end test for a logical run where sibling shard calls share one explicit `execution_id`.
- [ ] Add an end-to-end test for two distinct shard batches under one `run_id`.
- [ ] Add an end-to-end test for one logical test accumulating attempts across two execution scopes.
- [ ] Add an end-to-end test for mixed legacy and execution-aware events under one `run_id`.
- [ ] Add an end-to-end test for out-of-order steps and reconciliation with repeated logical tests.

### 7.3 API and UI validation

- [ ] Add API tests verifying run detail includes execution scopes.
- [ ] Add API tests verifying logical test detail aggregates attempts across execution scopes.
- [ ] Add API tests verifying legacy rows remain readable.
- [ ] Add API tests verifying ambiguous runs surface expected warnings or flags.
- [ ] Validate UI rendering for multi-execution run detail views.
- [ ] Validate UI rendering for cross-execution test attempt provenance.

## 8. Implementation Sequence

- [ ] Phase 1: finish protobuf and reporter changes first.
- [ ] Phase 2: finish server and processor identity plumbing next.
- [ ] Phase 3: land relational schema and repository changes after identity plumbing is stable.
- [ ] Phase 4: land Mongo live-buffer and reconciliation changes after persistence is execution-aware.
- [ ] Phase 5: land API, WebSocket, and UI updates after backend read models are stable.
- [ ] Phase 6: harden rollout diagnostics and ambiguity policy before production cutover.

## 9. Exit Criteria

- [ ] One logical run can safely aggregate multiple unsharded execution scopes under one `run_id`.
- [ ] One logical run can safely aggregate multiple sharded contributions under one `run_id`.
- [ ] Repeated logical tests accumulate attempts across execution scopes without collisions.
- [ ] Live in-memory and persisted identities no longer collide when `run_id` is intentionally reused.
- [ ] Legacy single-invocation flows remain functional.
- [ ] Missing shared shard-batch association reduces only fidelity, not correctness.
- [ ] API and UI can explain provenance at execution-scope level.
