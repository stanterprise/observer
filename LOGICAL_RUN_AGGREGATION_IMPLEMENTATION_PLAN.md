# Logical Run Aggregation Implementation Plan

This plan operationalizes [LOGICAL_RUN_AGGREGATION_PRD.md](LOGICAL_RUN_AGGREGATION_PRD.md).

It translates the agreed product model into concrete implementation work across the Playwright reporter, protobuf contract, Observer ingestion, processor, persistence, API, and web UI.

## 1. Purpose and Scope

The purpose of this effort is to let Observer ingest and present one logical run identified by `run_id` while safely accepting multiple distinct execution scopes under that same logical run.

The implementation must preserve three invariants:

1. `run_id` remains the logical aggregate run key.
2. `execution_id` becomes the uniqueness-first execution scope key beneath `run_id`.
3. Shared `execution_id` across sibling shard calls is optional and improves fidelity, but correctness must not depend on it.

This effort covers:

- Protobuf and reporter contract changes
- Observer ingestion and event handling changes
- PostgreSQL schema and repository changes
- MongoDB live-step-buffer identity changes
- API and UI updates for execution-aware reads
- Reconciliation, validation, and rollout logic

Out of scope for this phase:

- Merging different `run_id` values into one logical run
- Fuzzy logical test merging based only on title similarity
- Full historical backfill of existing MongoDB or PostgreSQL rows
- Warehouse-style analytics or cross-run comparison features

## 2. Implementation Approach

### 2.1 Delivery Strategy

Use an additive, compatibility-preserving implementation strategy.

Key principles:

- New writes become execution-aware.
- Existing reads continue to tolerate legacy rows that have no `execution_id`.
- Existing single-invocation workflows continue to function.
- Composite-run correctness is guaranteed only when `execution_id` is present consistently on all relevant events.

### 2.2 Rollout Shape

Recommended rollout order:

1. Extend protobuf schema and reporter generation behavior.
2. Make Observer backend execution-aware while still tolerating missing `execution_id` in legacy flows.
3. Update API and UI to expose and render execution-aware data.
4. Tighten validation and ambiguity handling once reporter rollout is complete.

### 2.3 Migration Policy

This is a prospective schema and behavior change.

- New runs should use execution-aware identities.
- Existing historical data remains readable.
- No mandatory backfill is required for correctness of new runs.
- Existing rows without `execution_id` should be treated as legacy execution data.

## 3. Core Technical Decisions

### D1. Identity Model

- `run_id` = logical aggregate run key
- `execution_id` = unique execution scope key within a logical run
- `shard_index` = execution-local shard identifier when present
- `logical_test_key` = stable logical test identity independent of `execution_id`
- `source_attempt_index` = reporter-local retry index within one execution scope

### D2. Batch Association Semantics

Shared `execution_id` across sibling shard calls is treated as optional batch association.

This is interpretive metadata derived from shared `execution_id`, not a separate primary identifier or required persisted entity in Phase 1.

If shard calls do not share `execution_id`:

- they remain valid
- they remain unique
- aggregate run correctness remains intact
- batch-level fidelity is reduced

### D3. Validation Semantics

The backend should reject or mark ambiguous the following cases:

- same `execution_id` reused across multiple unsharded calls under one `run_id`
- same `execution_id` reused across distinct shard batches under one `run_id` when identity becomes ambiguous
- incomplete event sets where some events in the same execution scope carry `execution_id` and others do not

### D4. Logical Test Merging

Logical test aggregation must not depend on `execution_id`.

`execution_id` separates concrete attempts. It must not redefine the logical test itself.

### D5. No Additional Grouping Identifier in Phase 1

Do not introduce a second grouping identifier in the initial implementation.

Do not model batch association as its own first-class identity axis in Phase 1.

Phase 1 semantics should be:

- unique execution scope first
- optional batch association when shard calls share one `execution_id`

## 4. Cross-Repo Dependency Order

This feature spans more than the current repository.

### 4.1 External Repositories

#### Protobuf Schema Repository

Repository: `github.com/stanterprise/proto-go`

Required because Observer currently imports event and entity types from the external protobuf package in [go.mod](go.mod#L15).

#### Playwright Reporter Repository

Repository: `github.com/stanterprise/stanterprise-playwright-reporter`

Required because reporter-side ID generation and env-var behavior are part of the product contract.

### 4.2 Observer Repository Workstreams

Observer packages most affected:

- `pkg/server`
- `pkg/consumer`
- `pkg/websocket`
- `internal/models`
- `internal/repository/postgres`
- `internal/repository/mongodb`
- `pkg/api`
- `web`

### 4.3 Recommended Order

1. Protobuf schema changes
2. Reporter generation logic
3. Observer backend model and persistence changes
4. Observer read/API/UI changes
5. Validation tightening and rollout hardening

## 5. Protobuf and Reporter Workstream

### 5.1 Protobuf Schema Changes

Add `execution_id` to all relevant event payloads and entity types that participate in run/test/step lifecycle.

At minimum, add it to:

- run start
- run end
- suite begin / suite end
- test begin / test end
- step begin / step end
- stdout / stderr
- failure / error events
- any nested entities used as the canonical event carriers for run/test/step identity

Requirements:

- field must be optional at wire level for backward compatibility
- field must be documented as required for full logical-run aggregation correctness
- generated types must preserve backward compatibility for older reporters

### 5.2 Reporter Generation Logic

Implement the agreed rules in the Playwright reporter:

#### Single `npx playwright test`

- if `run_id` env var is absent, auto-generate UUID
- if `execution_id` env var is absent, auto-generate UUID

#### Single shard call in `--shard` mode

- if `run_id` env var is absent, treat as standalone logical run and auto-generate
- if `execution_id` env var is absent, auto-generate UUID for that shard call

#### Multiple shard calls intended as one logical run

- `run_id` env var must be shared across all shard calls
- `execution_id` may be:
  - omitted, producing one execution scope per shard call
  - explicitly shared, producing optional batch association

#### Reporter Validation

- prevent accidental reuse of one explicit `execution_id` across obviously conflicting local calls when detectable
- log the resolved `run_id` and `execution_id` at reporter startup for diagnostics

### 5.3 Reporter Tests

Add reporter-side tests for:

- default UUID generation
- env var override behavior
- shard behavior with and without shared `execution_id`
- propagation of `execution_id` to every emitted event

## 6. Ingestion and Server Workstream

### 6.1 gRPC Server Acceptance Rules

Update [pkg/server/server.go](pkg/server/server.go) validation rules to accept the new field set.

Rules:

- accept missing `execution_id` for backward compatibility
- do not reject legacy clients outright in the first rollout phase
- preserve event payload unchanged into NATS so downstream processor sees `execution_id` when provided

### 6.2 Compatibility Classification

Introduce a runtime classification model for inbound runs:

- execution-aware: `execution_id` present consistently
- legacy: `execution_id` absent
- ambiguous: mixed or conflicting identity signals

This classification should drive warnings, not immediate hard failures, during the first rollout phase.

### 6.3 NATS Envelope Strategy

No new top-level NATS envelope field is required if `execution_id` is inside the canonical event payloads.

However, the processor must gain helpers to extract `execution_id` from all event types, similar to current `run_id` extraction logic.

## 7. Processor and Identity Plumbing Workstream

### 7.1 Shared Identity Helpers

Add execution-aware extraction and identity helpers in `pkg/consumer`.

Required helpers:

- `extractExecutionID(...)`
- execution-aware variants of step/test identity extractors
- execution-aware composite keys for deferred queues and reconciliation

Current run-only helpers in [pkg/consumer/nats_consumer.go](pkg/consumer/nats_consumer.go) should be extended rather than duplicated blindly.

### 7.2 Consumer Handler Updates

Update these handlers to read and propagate `execution_id`:

- [pkg/consumer/nats_run_handlers.go](pkg/consumer/nats_run_handlers.go)
- [pkg/consumer/nats_test_handlers.go](pkg/consumer/nats_test_handlers.go)
- [pkg/consumer/nats_step_handlers.go](pkg/consumer/nats_step_handlers.go)
- [pkg/consumer/nats_suite_handlers.go](pkg/consumer/nats_suite_handlers.go)
- [pkg/consumer/nats_std_handlers.go](pkg/consumer/nats_std_handlers.go)
- [pkg/consumer/event_classifier.go](pkg/consumer/event_classifier.go)

### 7.3 Integrity Tracking

Current run integrity tracking is keyed only by `run_id`.

Implementation change:

- split execution-scope tracking from aggregate logical-run tracking
- maintain per-execution-scope lifecycle state
- derive logical run completeness from execution scopes rather than from one flat `run_id` bucket

### 7.4 Deferred Queue Keys

Current deferred queue keys are run-scoped.

Implementation change:

- include `execution_id` in deferred queue identity
- include `execution_id` in orphan step replay paths

Without this, repeated tests in different execution scopes will collide.

## 8. Persistence Workstream

### 8.1 PostgreSQL Schema Changes

Follow the current GORM-backed migration pattern used by Observer.

#### Add New Execution Table

Introduce a `run_executions` table or equivalent model.

Recommended fields:

- internal primary key
- `run_id`
- `execution_id`
- status
- started_at
- finished_at
- metadata
- label/category nullable
- source information nullable
- ambiguity flags or legacy markers nullable

Indexes:

- unique `(run_id, execution_id)`
- btree `(run_id, status)`
- btree `(started_at desc)`

#### Extend Existing Tables

Add `execution_id` where execution-scoped identity is required:

- `run_shards`
- `suites` or suite-execution table
- `test_attempts`
- any table whose current uniqueness depends on run-only identity

Consider whether `attachments` need direct `execution_id` or can continue to inherit through `test_attempt_id`.

### 8.2 Model Updates

Update [internal/models/relational.go](internal/models/relational.go) with:

- `RunExecution` model
- `execution_id` fields on relevant models
- updated indexes and uniqueness constraints

Update [internal/models/relational_mappers.go](internal/models/relational_mappers.go) to map execution-aware event payloads into relational rows.

### 8.3 Identity Rework

Current row IDs derived only from `run_id` must be reviewed.

#### Tests

Tests should remain logical-test-scoped, not execution-scoped.

The implementation must introduce or formalize a `logical_test_key` strategy.

Candidate order:

1. explicit logical test id from reporter, if added later
2. stable external Playwright test id
3. deterministic derived fingerprint

#### Attempts

Attempt identity must include:

- logical test key
- `execution_id`
- `source_attempt_index`

Current uniqueness on `(test_id, attempt_index)` in [internal/models/relational.go](internal/models/relational.go#L174) will need to evolve.

### 8.4 Repository Changes

Update repositories in `internal/repository/postgres` to:

- upsert run execution rows
- attach shard rows beneath execution scopes
- attach suite execution rows beneath execution scopes
- upsert logical tests independently of execution scopes
- upsert attempts with execution-aware identity
- derive aggregate run status from execution-scope status

### 8.5 Legacy Data Tolerance

Existing rows without `execution_id` should remain readable.

Implementation options:

- nullable `execution_id` plus read-time legacy handling
- synthetic internal legacy execution marker for old rows at read time

Avoid forcing historical backfill in the initial rollout.

## 9. MongoDB Live Buffer Workstream

### 9.1 Buffer Identity

Current live buffer identity is run-scoped and test-scoped.

It must become execution-aware.

At minimum, buffer identity should include:

- `run_id`
- `execution_id`
- logical or external test key

Update [internal/repository/mongodb/mongodb_step_buffer.go](internal/repository/mongodb/mongodb_step_buffer.go) accordingly.

### 9.2 Flush Protocol

Flush behavior remains conceptually the same, but all lookups and deletes must be execution-aware.

This prevents one execution scope from deleting or overwriting another execution scope’s in-flight state.

### 9.3 Reconciliation Inputs

Reconciliation workers must operate at execution-scope granularity.

They should no longer assume one active state bucket per `run_id`.

## 10. Logical Run Completion and Reconciliation Workstream

### 10.1 Completion Model

Logical run completion must be derived from execution-scope completion.

Rules:

- a logical run stays active while any known execution scope is active
- one execution scope ending must not finalize the logical run if others remain active or unresolved
- timeout and partial states must be evaluated at execution-scope level first, aggregate level second

### 10.2 Shared Shard Batch Semantics

When sibling shard calls share `execution_id`, Observer may infer optional sibling association and use it for higher-fidelity status and diagnostics.

When they do not share `execution_id`, Observer must still remain correct.

This means:

- aggregate run completion works regardless
- shard-sibling completeness checks are only available when the data supports that interpretation

### 10.3 Ambiguity Handling

When invalid `execution_id` reuse is detected:

- first rollout phase: ingest with warning and mark ambiguous where feasible
- later rollout phase: optionally tighten to hard-fail invalid reuse

The plan should preserve a clear transition path between those modes.

## 11. API Workstream

### 11.1 REST Response Shape

Update API responses in `pkg/api` to expose execution-aware structure.

Recommended additions:

- logical run includes execution scope summary array
- test detail includes attempts annotated with `execution_id`
- run-level summary includes execution scope count
- attempt details include shard provenance when available

### 11.2 Query Layer

Update queries to support:

- loading aggregate run plus execution scopes
- loading logical tests with execution-aware attempts
- filtering by execution scope where useful

### 11.3 Backward Compatibility

Existing consumers of current API responses should continue to function if possible.

Prefer additive fields over breaking shape changes in phase 1.

## 12. Web and UX Workstream

### 12.1 Run Detail UI

Add execution-aware summary to run detail.

Desired UX:

- show when a run aggregates multiple execution scopes
- expose execution scope list with status and timing
- optionally label shard-batch association when shared `execution_id` exists

### 12.2 Test Detail UI

Show one logical test with attempts from multiple execution scopes.

Each attempt should expose:

- `execution_id`
- shard info if available
- retry index
- timing and status

### 12.3 Timeline and Advanced Views

Execution-aware views should read from the new model but remain correct when shard calls were not batch-associated.

Grouping by shared `execution_id` should be treated as an enhancement, not a prerequisite.

## 13. Observability and Diagnostics Workstream

Add metrics and logs for:

- generated vs explicit `execution_id` usage
- ambiguous execution reuse detections
- execution-scope creation count per logical run
- reconciliation actions at execution-scope level
- logical test merge counts across execution scopes
- legacy-mode run ingestion count

These diagnostics are necessary for rollout safety.

## 14. Testing Matrix

### 14.1 Unit Tests

- execution-aware ID extraction helpers
- execution-aware deferred queue keys
- logical test key derivation
- validation of explicit `execution_id` reuse rules
- run status aggregation from multiple execution scopes
- attempt merge ordering across execution scopes

### 14.2 Integration Tests

- one logical run with two unsharded execution scopes
- one logical run with shard calls using distinct auto-generated `execution_id` values per call
- one logical run with sibling shard calls sharing an explicit `execution_id`
- two shard batches under one `run_id`
- repeated logical test across two execution scopes
- mixed legacy and execution-aware events under one `run_id`
- out-of-order steps and reconciliation with repeated logical tests

### 14.3 API Tests

- run detail includes execution scopes
- test detail aggregates attempts across execution scopes
- legacy rows remain readable
- ambiguous runs surface expected warnings or flags

### 14.4 UI Validation

- run detail renders multi-execution summaries
- test detail renders provenance across execution scopes
- execution-aware views remain usable without shared shard batch association

## 15. Phase Plan

### Phase 1: Contract and Reporter

Deliverables:

- protobuf schema updated with `execution_id`
- reporter generation and env var behavior implemented
- reporter tests added

Exit criteria:

- reporter emits `execution_id` on all relevant events
- shard and unsharded generation rules match PRD

### Phase 2: Backend Identity Plumbing

Deliverables:

- execution-aware extraction helpers
- server and consumer paths accept and propagate `execution_id`
- ambiguity classification introduced

Exit criteria:

- all consumer handlers can access `execution_id`
- logs and warnings identify legacy vs execution-aware flows

### Phase 3: Persistence Model

Deliverables:

- new `run_executions` model/table
- `execution_id` added to execution-scoped tables
- repository writes execution-aware
- legacy reads still work

Exit criteria:

- execution-aware runs persist without collisions
- repeated logical tests can accumulate attempts across execution scopes

### Phase 4: Live Buffer and Reconciliation

Deliverables:

- Mongo live buffer keys become execution-aware
- flush paths execution-aware
- reconciliation and completion logic execution-aware

Exit criteria:

- no cross-execution live-buffer collisions
- one execution ending no longer finalizes the whole logical run prematurely

### Phase 5: API and UI

Deliverables:

- API exposes execution scope summaries and attempt provenance
- run detail and test detail become execution-aware
- advanced views tolerate missing batch association correctly

Exit criteria:

- multi-execution logical runs render correctly in the UI

### Phase 6: Rollout Hardening

Deliverables:

- telemetry and ambiguity reporting
- documented operator guidance
- decision on warnings vs hard failures for invalid reuse

Exit criteria:

- rollout diagnostics are sufficient to monitor real traffic

## 16. Concrete File Touchpoints

Expected Observer code areas to change:

- [go.mod](go.mod)
- [pkg/server/server.go](pkg/server/server.go)
- [pkg/consumer/nats_consumer.go](pkg/consumer/nats_consumer.go)
- [pkg/consumer/nats_run_handlers.go](pkg/consumer/nats_run_handlers.go)
- [pkg/consumer/nats_test_handlers.go](pkg/consumer/nats_test_handlers.go)
- [pkg/consumer/nats_step_handlers.go](pkg/consumer/nats_step_handlers.go)
- [pkg/consumer/nats_suite_handlers.go](pkg/consumer/nats_suite_handlers.go)
- [pkg/consumer/nats_std_handlers.go](pkg/consumer/nats_std_handlers.go)
- [pkg/consumer/event_classifier.go](pkg/consumer/event_classifier.go)
- [pkg/websocket/proto_to_model.go](pkg/websocket/proto_to_model.go)
- [internal/models/relational.go](internal/models/relational.go)
- [internal/models/relational_mappers.go](internal/models/relational_mappers.go)
- [internal/repository/postgres](internal/repository/postgres)
- [internal/repository/mongodb/mongodb_step_buffer.go](internal/repository/mongodb/mongodb_step_buffer.go)
- [pkg/api](pkg/api)
- [web/src](web/src)
- [tests](tests)

## 17. Exit Criteria

- A logical run can safely aggregate multiple unsharded execution scopes under one `run_id`.
- A logical run can safely aggregate multiple sharded contributions under one `run_id`.
- Repeated logical tests accumulate attempts across execution scopes without collisions.
- In-memory and persistence keys no longer collide when `run_id` is reused intentionally.
- Legacy single-invocation flows still work.
- Missing shared shard-batch association reduces only fidelity, not correctness.
- API and UI can explain provenance at execution-scope level.

## 18. Recommended Next Artifact

After this plan, the next useful artifact is an execution checklist or task tracker that breaks the work into dependency-ordered implementation tasks, ideally grouped by:

- protobuf and reporter
- backend identity plumbing
- persistence and migrations
- API and UI
- testing and rollout
