# Logical Run Aggregation PRD

## Overview

Observer should support one logical run composed of multiple unique execution scopes that share the same `run_id`.

This capability must support all of the following under one logical run:

- Unsharded Playwright calls
- Sharded Playwright calls
- Sequential execution scopes
- Concurrent execution scopes
- Repeated logical tests whose attempts should be combined into one logical test history

The core model is:

- `run_id` identifies the logical aggregate run
- `execution_id` identifies a unique execution scope within that logical run
- Shared `execution_id` across shard calls is optional and acts as batch association
- Uniqueness is mandatory; batch association is optional but helpful

This model keeps the input surface area small while solving the critical data integrity problems caused by reusing `run_id` alone.

## Problem Statement

Observer currently overloads `run_id`.

The same field is effectively used for:

- Logical run identity
- Execution identity
- Shard namespace
- In-memory reconciliation scope
- Attempt collision scope
- Buffer and deferred-queue scope

That works for one Playwright invocation and partially works for a single, globally coordinated shard batch, but it breaks when multiple distinct invocations intentionally contribute to the same logical run.

When separate executions reuse `run_id` today, Observer risks:

- Shard collisions
- Test row collisions
- Attempt collisions
- Live buffer collisions
- Premature run finalization
- Deferred-state cleanup while sibling executions are still active
- Suite hierarchy corruption
- Last-writer-wins metadata overwrites

This blocks an important workflow:

- Split a suite into multiple categories or lanes
- Run each category independently, optionally with its own sharding behavior
- Present all outcomes as one coherent logical run in Observer

## Feature Thesis

Observer should treat `run_id` as the identifier of a logical aggregate run and `execution_id` as the identifier of a unique execution scope beneath it.

That execution scope model should be flexible enough to support either of these equally valid modes:

- One `execution_id` per unsharded Playwright invocation
- One `execution_id` shared across sibling shard calls when the caller wants batch association

If shard calls do not share `execution_id`, Observer should still function correctly by treating each shard call as a distinct execution scope under the same logical run.

This prioritizes correctness and uniqueness over perfect knowledge of batching intent.

## Product Goals

1. Allow multiple execution scopes to contribute to one logical run keyed by a shared `run_id`.
2. Guarantee uniqueness of run calls even when `run_id` is intentionally reused.
3. Support every mix of sharded and unsharded execution scopes under the same logical run.
4. Keep batch association optional rather than mandatory.
5. Preserve schema integrity through additive changes where possible.
6. Merge repeated logical tests into one logical test record with a complete attempt history.
7. Preserve provenance so users can see which execution scope and shard produced each attempt.
8. Keep single-invocation behavior working for legacy and simple reporter workflows.

## Non-Goals

- Merging different `run_id` values into one logical run in the first release
- Fuzzy matching of unrelated tests by title alone
- Requiring end users to manually provide UUIDs in routine workflows
- Making batch association mandatory for correctness
- Full logical suite deduplication when suite identity is ambiguous
- Cross-repository or cross-branch aggregation without an explicit product decision

## User Scenarios

The feature must support the following cleanly under one logical run `X`:

1. Run one non-overlapping sharded suite under `X`, then another non-overlapping sharded suite under `X`.
2. Run one non-overlapping sharded suite under `X`, then another non-overlapping unsharded suite under `X`.
3. Run one non-overlapping unsharded suite under `X`, then another non-overlapping sharded suite under `X`.
4. Run one non-overlapping unsharded suite under `X`, then another non-overlapping unsharded suite under `X`.
5. Run tests `X1`, `Y1`, `Z1` under `X` with multiple attempts, then later run `X1`, `Y2`, `Z2` under `X` with multiple attempts, and have the attempts for `X1` combined into one logical test history.

The solution should also work when these execution scopes overlap in time.

## Target Users

- CI owners splitting a large suite into independently orchestrated lanes
- QA engineers grouping related suites into category-based runs
- Developers who want one coherent run page for the total result of several orchestrated jobs
- Teams investigating flaky tests that were retried across more than one execution scope

## Terminology

### Logical Run

The aggregate run identified by `run_id`.

This is the top-level run entity users think of as the overall run.

### Execution Scope

The entity identified by `execution_id` within one `run_id`.

An execution scope may represent:

- One unsharded Playwright invocation
- One shard call when shard calls do not share `execution_id`
- One intentionally grouped shard batch when sibling shard calls share the same `execution_id`

Observer should not assume batch grouping exists unless the data explicitly indicates it.

### Batch Association

Optional grouping intent created when multiple shard calls under the same `run_id` intentionally share the same `execution_id`.

Batch association is helpful for fidelity and lifecycle interpretation, but it is not required for correctness.

### Shard Call

One concrete `npx playwright test --shard=...` process invocation.

### Logical Test

The aggregate representation of a test within one logical run.

### Concrete Attempt

One actual executed attempt of a logical test.

Each concrete attempt belongs to one execution scope and may also carry shard provenance.

## Core Product Model

The product model should become:

- One logical run per `run_id`
- Many execution scopes under that logical run
- Optional shard data attached to execution scopes
- Many logical tests under the logical run
- Many concrete attempts under each logical test

This model should support both:

- Accurate aggregate run summaries
- Provenance-aware drill-down when users need to know where a result came from

## Key Product Principles

### 1. Uniqueness First

Observer must guarantee unique persisted and in-memory identity for each execution scope contributing to a logical run.

Batch association is secondary.

### 2. Additive Schema Evolution

The feature should preserve current schema integrity through additive changes where practical.

That means:

- Keep `runs` as the aggregate logical run keyed by `run_id`
- Add execution-scoped identity beneath it rather than redefining `run_id`
- Keep existing simple workflows functioning with minimal required input

### 3. Explicit Over Inferred Identity

Observer should not attempt to infer execution grouping from timing alone when explicit identity is available.

### 4. Merge Only Where Identity Is Stable

Logical tests should merge across execution scopes only when their logical identity is stable.

`execution_id` must separate attempts, not redefine the logical test itself.

### 5. Preserve Provenance

Every concrete attempt should retain:

- Which `execution_id` produced it
- Which shard, if any, produced it
- Which source retry index it had
- Which metadata and labels applied to that execution scope

## Identity and Generation Model

### Logical Run Identity

`run_id` is the logical run identifier.

If `run_id` is not specified for a single `npx playwright test` call, the reporter may auto-generate a UUID.

If the user wants multiple calls to aggregate into one logical run, sharing `run_id` across those calls is required.

### Execution Scope Identity

`execution_id` is the execution scope identifier.

If `execution_id` is not specified for a given execution scope, the reporter should auto-generate a UUID.

This guarantees uniqueness even when `run_id` is intentionally reused.

### Single Unsharded Call

Within a single `npx playwright test` call:

- If `run_id` is absent, auto-generate a UUID
- If `execution_id` is absent, auto-generate a UUID

Result:

- One logical run
- One execution scope

### Single Sharded Batch

Within a single batch of `npx playwright test --shard=...` calls:

- `run_id` must be shared if the user wants one logical aggregate run
- If `execution_id` is absent on each shard call, each shard call auto-generates its own `execution_id`
- If the same `execution_id` is explicitly shared across sibling shard calls, batch association exists

Both modes are valid.

If no shared `execution_id` is provided, correctness and uniqueness still hold. The only thing lost is explicit knowledge that the shard calls were intended siblings.

### Multiple Sharded Batches Under One Logical Run

When two different shard batches contribute to the same logical run:

- Shared `run_id` is mandatory if the user wants one logical aggregate run
- Each batch may either share one explicit `execution_id` across its shard calls or let each shard call auto-generate one
- Different batches must not reuse the same `execution_id` in a way that collapses distinct execution scopes into one ambiguous identity

### Summary

The intended behavior is:

- Uniqueness is guaranteed
- Batch association is optional but helpful
- Lack of batch association reduces fidelity, not correctness

## Validation Guardrails

The model needs a few explicit rules to remain safe.

### Valid

- One unsharded call with auto-generated `run_id` and `execution_id`
- Multiple unsharded calls sharing `run_id` but each with a distinct `execution_id`
- Multiple shard calls sharing `run_id` and each having distinct auto-generated `execution_id`
- Multiple sibling shard calls sharing both `run_id` and `execution_id`

### Invalid

- Reusing the same `execution_id` across multiple unsharded calls within the same `run_id`
- Reusing the same `execution_id` across separate shard batches under the same `run_id` when their shard namespace overlaps or their identity would become ambiguous
- Omitting `execution_id` from some events belonging to an otherwise identified execution scope

Observer should validate these conditions and surface clear errors or warnings rather than silently corrupting data.

## Functional Requirements

- FR-001: Observer must allow multiple execution scopes to attach to one logical run identified by a shared `run_id`.
- FR-002: Every execution scope must have a unique `execution_id` within the logical run.
- FR-003: A logical run must support any mix of sharded and unsharded execution scopes.
- FR-004: Shared `execution_id` across sibling shard calls must be optional, not mandatory.
- FR-005: When shard calls do not share `execution_id`, Observer must still preserve correctness by treating them as distinct execution scopes.
- FR-006: Logical run summaries must aggregate across all execution scopes belonging to the same `run_id`.
- FR-007: A logical run must not be finalized just because one execution scope reaches terminal state while sibling execution scopes are still active or unresolved.
- FR-008: Repeated logical tests across execution scopes must merge into one logical test record when their logical identity matches.
- FR-009: Attempts for a repeated logical test must be preserved as separate concrete attempts and exposed in a stable logical order.
- FR-010: Each concrete attempt must preserve execution provenance, including at least `execution_id`, source retry index, and shard attribution when available.
- FR-011: Non-overlapping tests across execution scopes must remain distinct and must not merge accidentally.
- FR-012: Suite execution identity collisions across execution scopes must not overwrite or corrupt each other.
- FR-013: The system must remain idempotent under message redelivery and replay.
- FR-014: Legacy single-invocation workflows must continue to work without requiring the user to manually provide IDs.
- FR-015: Composite runs must surface both aggregate run data and an execution-aware breakdown in API responses.
- FR-016: Aggregate run metrics must include both distinct logical test counts and total concrete attempt counts.
- FR-017: The system must preserve per-execution metadata and must not silently overwrite conflicting execution metadata on the logical run.
- FR-018: Steps, stdout, stderr, attachments, failures, and errors must remain attached to concrete attempts rather than being flattened into logical-test-level blobs.
- FR-019: Missing terminal events must be recoverable through reconciliation or timeout logic at the execution-scope level.
- FR-020: Composite runs must be queryable and renderable even when execution scopes overlap in time.
- FR-021: The system should surface definition drift when the same logical test appears across execution scopes with materially different identifying metadata such as title, location, or suite path.
- FR-022: The system should preserve a clean path for future execution-aware UI features such as execution filters and optional batch-level diagnostics.

## Data Model Strategy

This PRD recommends an additive model that preserves the current top-level `runs` table as the aggregate logical run.

### Aggregate Entities

#### Logical Run

Existing `runs` row, keyed by `run_id`.

It should represent the aggregate state of the logical run and contain derived fields such as:

- Aggregate status
- Earliest start time across execution scopes
- Latest end time across execution scopes
- Distinct logical test count
- Total attempt count
- Execution scope count
- Composite-run indicator

#### Logical Test

The `tests` table should continue to represent a logical test where practical.

For composite runs, that means the `tests` row should represent the aggregate test identity within the logical run, not one execution-specific copy.

### Execution-Scoped Entities

#### Run Execution

Add a first-class execution record under the logical run.

Recommended shape:

- Internal primary key
- `run_id` foreign key to logical run
- `execution_id`
- Status
- Started at
- Finished at
- Metadata
- Optional label or category
- Optional source information such as worker pool, source host, or source id
- Optional indicator that multiple shard calls shared this `execution_id`

The core requirement is uniqueness and traceability. Batch semantics may exist, but they are not required for correctness.

#### Run Shard

Preserve the current `run_shards` concept but scope it beneath `execution_id`.

Recommended additive change:

- Add `execution_id` to shard rows
- Make shard uniqueness execution-scoped rather than only `run_id + shard_index`

Recommended identity:

- `run_id + execution_id + shard_index`

`shard_total` should remain important validation metadata, but it does not need to be the primary uniqueness key.

#### Suite Execution

Suites currently conflate logical identity and execution occurrence.

To avoid corruption, suite execution data should become execution-scoped.

Two acceptable approaches are:

- Add `execution_id` to existing suite rows and treat them as suite executions
- Introduce separate suite execution records while preserving logical suite grouping at read time

The feature should favor the smallest additive change that prevents hierarchy and timing corruption when the same suite id appears in more than one execution scope.

#### Concrete Attempt

`test_attempts` should represent concrete attempts, not only reporter-local retry indices.

For aggregated runs, attempt identity must distinguish:

- Logical test identity
- Execution scope identity
- Source retry index inside that execution scope
- Stable logical attempt order for aggregate display

Recommended additive fields:

- `execution_id`
- `source_attempt_index` or equivalent
- `logical_attempt_index`

This allows one logical test such as `X1` to accumulate attempts from multiple execution scopes without collisions.

### Identity Rules

#### Logical Run Key

- `logical_run_key = run_id`

#### Execution Scope Key

- `execution_scope_key = run_id + execution_id`

#### Execution Shard Key

- `execution_shard_key = run_id + execution_id + shard_index`

#### Logical Test Key

The system needs a stable logical test key that does not depend on `execution_id`.

Preferred order:

1. Explicit reporter-provided logical test id, if introduced
2. Existing stable external test id, if guaranteed stable across execution scopes
3. A deterministic derived fingerprint using stable attributes such as project, file, suite path, and test id

The product should not rely on display title alone.

#### Concrete Attempt Key

Recommended conceptual key:

- `run_id + logical_test_key + execution_id + source_attempt_index`

Additionally, Observer should compute a `logical_attempt_index` for presentation and ordered history.

## Status Aggregation Rules

### Logical Run Status

Logical run status should be derived from all execution scopes and final logical test outcomes.

At minimum:

- `RUNNING` if any execution scope is still active
- `FAILED` if the logical run is complete and at least one logical test finished failed
- `TIMEDOUT` if a terminal timeout condition applies and no stronger failure model supersedes it
- `INTERRUPTED` if the run is explicitly interrupted
- `PASSED` only if all relevant logical tests reached passing terminal state and all known execution scopes are complete
- `PARTIAL` or equivalent should be considered when one or more execution scopes never finalized cleanly

### Logical Test Status

Logical test status should be derived from all concrete attempts across execution scopes.

Expected behavior:

- If any later attempt passes after earlier failures, the logical test may be `FLAKY` or `PASSED_WITH_RETRIES`
- If all attempts fail, logical status is `FAILED`
- If the final known state is incomplete due to execution loss or timeout, the logical test may be `INTERRUPTED`, `TIMEDOUT`, or `PARTIAL`

The exact taxonomy should align with current Observer status conventions.

## Metrics and Data Requirements

The feature should provide both aggregate and execution-aware metrics.

### Logical Run Metrics

- Distinct logical test count
- Total concrete attempt count
- Execution scope count
- Total shard count across execution scopes
- Passed, failed, flaky, skipped, interrupted, and timed out logical test counts
- Start time, end time, and duration derived from all execution scopes
- Count of repeated logical tests that appeared in more than one execution scope
- Count of metadata conflicts, if any

### Execution Scope Metrics

- `execution_id`
- Optional label or category
- Execution-local shard count
- Execution-local test count
- Execution-local attempt count
- Start time, end time, duration
- Execution status
- Source metadata
- Optional indication whether the execution scope was shared across multiple shard calls

### Attempt-Level Data

Each concrete attempt should preserve:

- Logical test reference
- `execution_id`
- Optional label or category
- Execution-local shard identifier if present
- Source retry index
- Logical attempt index
- Attempt status
- Attempt timing
- Attachments, steps, stdout, stderr, failures, and errors

## UX Expectations

This PRD is primarily about run semantics, but the user experience should make the merged model visible rather than hiding it.

### Logical Run Detail

The logical run detail page should:

- Continue to behave like the canonical run page
- Show that the run aggregates multiple execution scopes when applicable
- Surface top-level aggregate summary metrics
- Provide an execution-aware breakdown panel
- Allow filtering by execution scope or category when available

### Test Detail

Test detail should:

- Show one logical test record for repeated logical tests
- Present a complete attempt history across execution scopes
- Preserve provenance so users can see where each attempt came from

### Timeline and Other Run-Level Views

Execution-aware views should be able to show:

- Aggregate run timeline
- Execution scope boundaries where identifiable
- Optional grouped shard behavior when shared `execution_id` is present across shard calls
- Repeated tests appearing in different execution scopes

If shard calls do not share `execution_id`, the view should remain correct even if it cannot reconstruct user intent about batching.

## Reporter and Ingestion Expectations

### Required Runtime Behavior

- The reporter must emit `run_id` on all relevant events
- The reporter must emit `execution_id` on all relevant events belonging to the same execution scope
- If `run_id` is absent for a simple one-off call, the reporter may auto-generate it
- If `execution_id` is absent for an execution scope, the reporter may auto-generate it

### Event Coverage

`execution_id` must be present on run, test, step, output, failure, and terminal events for the execution scope it identifies.

Observer should not rely on out-of-band inference for this field.

### Shared `execution_id` for Shard Calls

If sibling shard calls intentionally share the same `execution_id`, Observer should interpret that as optional batch association.

If sibling shard calls do not share `execution_id`, Observer should treat them as distinct execution scopes under the same logical run.

### Backward Compatibility

Legacy single-invocation workflows should continue to work with reporter-side auto-generation of missing IDs.

Users should not be required to provide UUIDs manually for normal use.

## Generation Rules

### Single `npx playwright test`

- If `run_id` is not specified, auto-generate a UUID
- If `execution_id` is not specified, auto-generate a UUID

### Single Batch of `npx playwright test --shard=...`

- Shared `run_id` is required if the caller wants one logical run
- If `execution_id` is not specified, each shard call auto-generates its own `execution_id`
- If `execution_id` is specified and reused across sibling shard calls, batch association exists

### Two Different Shard Batches Contributing to the Same Logical Run

- Shared `run_id` is required
- Each batch may either provide a shared `execution_id` or allow per-call auto-generation
- Reusing the same explicit `execution_id` across distinct batches must be prevented when it would make execution identity ambiguous

## Corner Cases

### 1. Same shard numbers reused in different execution scopes

Expected behavior:

- No collisions
- Shards are execution-scoped

### 2. Unsharded execution scope added to an already sharded logical run

Expected behavior:

- The unsharded execution scope remains distinct
- Aggregate run metrics update cleanly

### 3. Sharded execution scope added to an already unsharded logical run

Expected behavior:

- Sharded behavior is represented without forcing the earlier unsharded execution into a conflicting shard namespace

### 4. Same logical test appears in multiple execution scopes

Expected behavior:

- One logical test record
- Many concrete attempts
- Stable logical attempt ordering
- Per-attempt provenance retained

### 5. Same reporter-local retry index reused in different execution scopes

Expected behavior:

- No collision because attempt identity includes execution provenance

### 6. Same logical test appears with changed title, file, or suite path

Expected behavior:

- Attempts can still merge if logical identity is explicit and stable
- Observer should preserve mismatched metadata and surface definition drift rather than hiding it

### 7. Same suite id reused in different execution scopes

Expected behavior:

- No overwrite or hierarchy corruption
- Execution-scoped suite records or equivalent protection

### 8. Execution scopes overlap in time

Expected behavior:

- All remain active under the same logical run
- Run-level completion waits for all known execution scopes

### 9. One execution scope never emits terminal state

Expected behavior:

- Reconciliation or inactivity timeout should finalize or mark that execution scope independently
- The logical run may become `PARTIAL`, `TIMEDOUT`, or equivalent until fully resolved

### 10. Late events arrive after an execution scope was considered complete

Expected behavior:

- Late events are handled at the execution-scope level
- Aggregate state is recomputed safely rather than dropped or used to corrupt sibling execution scopes

### 11. Duplicate or replayed events are received from the event bus

Expected behavior:

- Idempotent upserts remain correct for logical runs, execution scopes, shards, tests, and attempts

### 12. Conflicting execution metadata

Examples:

- Different branch
- Different commit SHA
- Different project name
- Different initiator

Expected behavior:

- Preserve per-execution metadata
- Aggregate run stores either canonical metadata plus conflict indicators, or a structured conflict summary
- Do not silently overwrite one execution scope with another

### 13. Same explicit `execution_id` reused across multiple unsharded calls in one `run_id`

Expected behavior:

- Validation error or explicit ambiguity warning
- No silent collapse of distinct execution scopes into one

### 14. Same explicit `execution_id` reused across distinct shard batches in one `run_id`

Expected behavior:

- Validation error or explicit ambiguity warning when the identity would collapse distinct batches into one execution scope

### 15. Composite run created by auto-generated per-shard `execution_id` values only

Expected behavior:

- Behavior remains correct
- Uniqueness is preserved
- Batch fidelity is reduced because no shared batch association exists

### 16. Entire execution scope is re-run intentionally

Expected behavior:

- If the rerun is meant to be a new execution scope, it uses a new `execution_id`
- If it is a continuation or replay of the same execution scope, it reuses the same `execution_id` and remains idempotent

## Acceptance Scenarios

### Scenario 1: Sharded plus sharded, no shared batch association

Given logical run `X`
And one shard batch contributes multiple shard calls with distinct auto-generated `execution_id` values
And a second shard batch later contributes additional shard calls with distinct auto-generated `execution_id` values
When all events are ingested
Then Observer stores one logical run
And multiple distinct execution scopes without collisions
And one aggregate run summary across all of them.

### Scenario 2: Sharded plus sharded, shared batch association

Given logical run `X`
And sibling shard calls share one explicit `execution_id`
When all events are ingested
Then Observer preserves one execution scope with shard-aware provenance
And batch association is available for higher-fidelity interpretation.

### Scenario 3: Sharded plus unsharded

Given logical run `X` with one sharded contribution and one unsharded contribution
When both complete
Then Observer shows one logical run with both execution types represented cleanly
And no shard collision or run-finalization corruption occurs.

### Scenario 4: Unsharded plus unsharded

Given logical run `X` with two non-overlapping unsharded execution scopes
When they finish
Then Observer stores one logical run with two distinct execution scopes
And a combined logical test set.

### Scenario 5: Repeated logical test across execution scopes

Given test `X1` runs in execution scope `A` with multiple attempts
And later test `X1` runs again in execution scope `B` with multiple attempts
When the user opens test detail for `X1`
Then they see one logical test record
And all attempts across both execution scopes in stable order
And per-attempt provenance including execution and shard context.

### Scenario 6: Concurrent execution scopes

Given two execution scopes under logical run `X` are active at the same time
When one finishes early
Then Observer does not finalize the logical run prematurely
And the remaining execution scope continues safely.

### Scenario 7: Missing terminal event for one execution scope

Given one execution scope never emits an end event
When inactivity reconciliation triggers
Then Observer finalizes or marks that execution scope independently
And the logical run reflects the unresolved condition without corrupting sibling execution scopes.

## Success Criteria

- SC-001: All four sharded and unsharded mixing scenarios produce one logical run without key collisions.
- SC-002: Repeated logical tests across execution scopes render as one logical test with a complete attempt history.
- SC-003: Execution-scope completion no longer causes premature cleanup or finalization for still-active sibling execution scopes.
- SC-004: Aggregate run metrics remain correct for distinct logical tests and concrete attempt counts.
- SC-005: Auto-generated IDs allow single-invocation workflows to continue without required manual configuration.
- SC-006: Composite runs expose enough provenance for users to understand which execution scope and shard produced each attempt.
- SC-007: Lack of shared shard-batch association does not compromise data correctness.

## Rollout Considerations

### Phase 1

- Add `execution_id` to the product model and event contract
- Make execution-local state safe for shards, buffers, attempts, and reconciliation
- Preserve aggregate logical run behavior

### Phase 2

- Expose execution-aware API responses
- Surface execution-scope breakdown in run detail
- Show merged attempt history for repeated logical tests

### Phase 3

- Add richer execution-aware analytics and filtering
- Add optional batch-aware diagnostics when shared `execution_id` is present across shard calls

## Open Questions

1. Is the current Playwright test id stable enough to serve as the logical test key across split execution scopes, or should Observer introduce an explicit logical test identifier?
2. Should suite aggregation be execution-aware only in the first release, with logical suite deduplication deferred until suite identity is better defined?
3. Should the API explicitly mark whether an execution scope represented one process or an intentionally shared shard batch?
4. What user-visible status should represent a logical run where some execution scopes completed but one remained unresolved after timeout: `PARTIAL`, `INTERRUPTED`, `TIMEDOUT`, or a new aggregate-specific status?
5. Should Observer hard-fail invalid `execution_id` reuse, or should it ingest with warnings and mark the affected run ambiguous?

## Recommendation

The preferred product direction is:

- Keep `run_id` as the logical run id
- Add `execution_id` as the uniqueness-first execution scope id beneath it
- Allow shared `execution_id` across sibling shard calls as optional batch association
- Treat missing shared batch association as a fidelity gap, not a correctness failure
- Treat `tests` as logical tests and `test_attempts` as concrete attempts with execution provenance
- Preserve backward compatibility by auto-generating missing IDs in simple workflows

This is the narrowest change that satisfies the requested scenarios while maintaining schema integrity, keeping inputs minimal, and avoiding unsafe overload of `run_id` semantics.
