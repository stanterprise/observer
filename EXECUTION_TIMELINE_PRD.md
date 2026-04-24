# Execution Timeline PRD

## Overview

Execution Timeline is a run-level visualization for understanding how test execution was distributed over time across shards and workers. It is a spiritual sibling to Test Map: where Test Map helps users understand status and distribution of tests as a set, Execution Timeline helps users understand concurrency, ordering, utilization, and imbalance.

The page should present a time-based view, similar in spirit to a Gantt chart, composed of swimlanes that represent run shards and their workers. Test executions appear as time-bounded segments placed into the correct lane and positioned relative to the run start.

Primary value:

- Reveal shard imbalance
- Reveal worker imbalance
- Surface idle gaps and bottlenecks
- Make long-running or blocking tests obvious
- Help users reason about parallelization quality

## Problem Statement

Observer already captures test runs, attempts, steps, and shard metadata, but there is no dedicated view that explains how a run unfolded over time.

Without an execution-timeline view, users cannot quickly answer questions such as:

- Did one shard finish far later than the others?
- Did one worker do most of the work while others were idle?
- Which tests occupied the critical path?
- Did retries or long-running tests create visible bottlenecks?
- Was a supposedly parallel run actually balanced in practice?

This feature should turn raw execution timing into an operational view for diagnosing inefficient sharding and worker allocation.

## Feature Thesis

Execution Timeline should become the fastest way to understand run scheduling behavior for a single test run.

Test Map answers: what ran, what failed, what is clustered.

Execution Timeline answers: when things ran, where concurrency was used, and which shard or worker became the bottleneck.

## Product Goals

1. Allow users to understand the temporal structure of a single run at a glance.
2. Make shard-level and worker-level imbalance visually obvious.
3. Let users inspect a single test segment and navigate to deeper detail.
4. Support both unsharded and sharded Playwright runs.
5. Work for completed runs first, with a clear path to in-progress visualization.

## Non-Goals

- Cross-run comparison in the first release
- Timeline editing or scheduling recommendations in the first release
- Artifact viewing inside the timeline itself
- Full critical-path analysis or automatic optimization recommendations in the first release
- Replacing Test Map; this is a complementary view

## Domain Analysis

The user-provided constraints imply the following:

1. A test run may represent either a single command invocation or a sharded execution spread across multiple run shards.
2. Each Playwright invocation may run multiple workers in parallel.
3. Test records retain timing, worker identity, and shard information either directly or through associated execution data.
4. The primary user mental model is hierarchical:
   - Run
   - Shard
   - Worker
   - Test execution segment
5. The main product question is not just what happened, but how evenly work was distributed.

This means the feature is not merely another detail page. It is a diagnostic tool for concurrency efficiency.

## Target Users

- Developers investigating unexpectedly slow runs
- QA engineers analyzing test suite parallelization behavior
- CI owners tuning shard counts and worker counts
- Engineering teams diagnosing bottlenecks, retries, and idle capacity

## Core User Stories

### P1. Understand run distribution

As a user viewing a single test run, I want to see a time-based lane view of shard and worker activity so I can immediately tell whether execution was balanced.

### P1. Find bottlenecks

As a user, I want long-running tests, idle gaps, and overloaded workers to stand out so I can identify the critical path and likely causes of slow completion.

### P1. Drill into a test execution

As a user, I want to click a timeline segment and open the relevant test detail so I can move from visual diagnosis to investigation.

### P2. Understand retries and repeated execution

As a user, I want retries or repeated attempts to be distinguishable on the timeline so I can see whether instability distorted runtime distribution.

### P2. Filter the view

As a user, I want to filter by shard, worker, status, and optionally suite or tag so I can narrow the timeline to the slice I care about.

### P3. Monitor in-progress runs

As a user, I want the page to refresh or stream updates for active runs so I can watch concurrency unfold while a run is still executing.

## User Experience Concept

### Entry Point

- Accessible from the Test Run Detail page alongside the existing Test Map entry point
- Suggested route: `/suite_runs/:runId/timeline`

### Layout

- Horizontal axis represents elapsed run time
- Vertical stack represents swimlanes
- Swimlanes are grouped by shard
- Each shard contains one or more worker lanes
- Each test execution appears as a colored time segment in its assigned worker lane

### Visual Model

- Unsharded runs display a single shard group, labeled clearly as the only shard or default shard
- Sharded runs display one group per shard, ordered by shard index
- Worker lanes are ordered numerically or by stable worker identifier
- Test segments are placed by start and end timestamps relative to run start
- Segment color indicates status
- Segment width indicates duration
- Hover reveals details
- Click navigates to test detail

### Suggested Summary Strip

At the top of the page, include a compact summary strip with:

- Total run duration
- Shard count
- Worker count
- Longest shard duration
- Busiest worker utilization
- Largest idle gap

This should support quick diagnosis before the user inspects the chart.

## Functional Requirements

- FR-001: The system must render a timeline for a single run using time-bounded execution segments placed relative to a common run clock.
- FR-002: The timeline must organize swimlanes hierarchically by shard and then by worker.
- FR-003: The system must support unsharded runs by presenting a single shard group.
- FR-004: Each segment must represent one test execution instance, including retries when applicable.
- FR-005: Each segment must expose at least test title, status, duration, start time, end time, shard identifier, worker identifier, and retry/attempt information when available.
- FR-006: Users must be able to click a segment and navigate to the corresponding test detail view.
- FR-007: Users must be able to visually distinguish statuses such as passed, failed, flaky, running, skipped, timed out, and interrupted.
- FR-008: The page must provide a clear representation of idle periods, either by visible gaps in lanes or summary metrics.
- FR-009: The page must allow filtering by shard and worker.
- FR-010: The page should allow filtering by status.
- FR-011: The page should support a fit-to-run view and one or more zoom levels for dense runs.
- FR-012: The page must handle retries without obscuring the fact that a test executed multiple times.
- FR-013: For active runs, the page should refresh or stream updates without requiring a full navigation reload.
- FR-014: The page must degrade gracefully when shard or worker attribution is incomplete by grouping segments into a clearly labeled fallback lane such as Unknown Worker or Unassigned.
- FR-015: The page must surface imbalance-oriented summary metrics at shard and worker level.

## Data Requirements

### Required Timeline Entity Shape

The feature needs a normalized execution structure with at least the following logical entities:

- RunTimeline
  - runId
  - runStartTime
  - runEndTime or current time for active runs
  - summary metrics
- ShardLane
  - shardId or shardIndex
  - shard label
  - shard start/end
  - shard summary metrics
- WorkerLane
  - workerId or workerIndex
  - parent shard
  - worker summary metrics
- ExecutionSegment
  - testId
  - test title
  - attempt or retry index
  - status
  - shard attribution
  - worker attribution
  - start time
  - end time
  - duration

### Data Expectations

The PRD assumes the backend can provide or derive:

- Run start and end timestamps
- Shard identity and shard timing
- Test execution start and end timestamps
- Retry or attempt identity
- Worker identity for the execution segment

### Observed Implementation Constraint

Current repository evidence suggests:

- Run shard metadata is already modeled and persisted in the relational path.
- Worker identity is clearly present on step-level data and frontend step types.
- Worker identity does not yet appear to be a first-class field on the relational test or attempt models used for run-level querying.

Product implication:

- The timeline can likely ship shard-aware behavior on top of existing relational run data.
- Worker-accurate lanes may require either a backend API enrichment step or promotion of worker attribution into a query-friendly run-level execution record.

This should be treated as a delivery constraint, not a reason to weaken the product definition.

## Derived Metrics

The page should compute or receive the following metrics:

- Total run duration
- Duration per shard
- Duration per worker
- Test count per shard
- Test count per worker
- Busy time per worker
- Idle time per worker
- Idle percentage per worker
- Longest segment per shard
- Longest segment overall
- Optional critical-path approximation in later phases

## Acceptance Scenarios

### Scenario 1: Unsharded run

Given a run with one logical execution group and multiple workers,
When the user opens Execution Timeline,
Then they see one shard group containing worker lanes with test segments placed over time.

### Scenario 2: Sharded run

Given a run distributed across multiple shards,
When the user opens Execution Timeline,
Then they see one shard group per shard and can compare finish times, gaps, and worker usage across them.

### Scenario 3: Imbalanced worker load

Given a run where one worker handled substantially more duration than peer workers,
When the user views the timeline,
Then the overloaded lane is visibly denser or longer-running and summary metrics reflect the imbalance.

### Scenario 4: Retry visibility

Given a test that executed more than once,
When the user inspects the relevant lane,
Then each attempt is distinguishable and does not collapse into a misleading single segment.

### Scenario 5: Missing worker attribution

Given a run where some execution records lack worker identity,
When the timeline renders,
Then those segments are still shown in a clearly labeled fallback lane rather than being dropped or silently misassigned.

### Scenario 6: Active run

Given a run that is still in progress,
When the user opens or stays on the timeline page,
Then new segments and status changes appear through polling or streaming without losing zoom state or filter state.

## Edge Cases

- A run has shard metadata but some tests do not have reliable shard attribution.
- A run has worker metadata only for part of the execution.
- Start time exists but end time is missing because the run is still active.
- A test has zero or near-zero duration.
- Clock precision differences create tiny overlaps or negative-looking gaps.
- Retries occur on different workers than the original attempt.
- Retries occur on different shards than the original attempt.
- A shard exists but no worker lanes are attributable.
- Very dense runs create too many segments to render comfortably without progressive rendering or virtualization.
- A test segment spans a long interval while nested step data suggests intermittent internal activity.

## UX Requirements

- The chart must remain legible for both small and large runs.
- Dense runs must not devolve into unreadable pixel noise without controls to zoom, fit, or filter.
- Hover details must appear quickly and not block panning or zooming.
- The visual system should be consistent with Test Map where appropriate, especially status colors and navigation conventions.
- Empty, loading, and partial-data states must explain what is missing.

## MVP Scope Recommendation

### MVP

- Single-run timeline page
- Shard grouping
- Worker lanes where attribution is available
- Segment rendering for test executions and retries
- Hover tooltip and click-through navigation
- Basic filters: shard, worker, status
- Summary metrics for imbalance analysis
- Support for completed runs

### Next Phase

- Live updates for in-progress runs
- Tag and suite filtering
- Zoom presets and manual zoom
- Sticky time scale and sticky lane labels
- Better retry visualization patterns

### Later Phase

- Cross-run comparison
- Automatic imbalance detection callouts
- Critical-path highlighting
- Export or shareable snapshot

## Proposed Delivery Approach

### Phase 1: Data contract validation

- Confirm authoritative source of shard attribution per execution segment.
- Confirm authoritative source of worker attribution per execution segment.
- Decide whether the timeline API should be built from relational execution tables, enriched documents, or a dedicated projection.

### Phase 2: Timeline API

- Introduce a run-scoped API response tailored for timeline rendering rather than forcing the frontend to reconstruct lanes from deep nested run detail payloads.
- Include normalized lane and segment data plus summary metrics.

### Phase 3: Frontend timeline page

- Add navigation from Run Detail.
- Render grouped lanes and time segments.
- Add summary strip, hover state, click-through, and filtering.

### Phase 4: Active-run support

- Add polling or WebSocket updates with stable viewport behavior.

## Success Criteria

- SC-001: A user can identify the longest shard and the busiest worker within 30 seconds of opening the page for a representative run.
- SC-002: A user can determine whether a run was balanced across shards and workers without leaving the timeline page.
- SC-003: For representative sharded runs, at least 95% of execution segments with available attribution are placed into the correct shard and worker lane.
- SC-004: For representative large runs, the initial timeline becomes interactable quickly enough that users do not perceive the page as stalled.
- SC-005: Users investigating slow CI runs can trace at least one suspected bottleneck test directly from the timeline into test detail.

## Open Questions

1. What is the canonical source of worker attribution for a test execution segment in the current architecture?
2. Should lane identity be based on worker index, worker id, or a synthesized stable label?
3. Should retries appear as separate segments in the same lane, or should the UI optionally collapse them?
4. Is shard attribution always derivable per test execution, or only at run-level metadata today?
5. Should the first release support only completed runs if live lane placement for in-flight tests is not yet reliable?
6. Should the page be positioned as a tab, button, or secondary navigation sibling to Test Map?

## Recommendation

Proceed with this feature as a run-level diagnostic view modeled as a sibling to Test Map, but treat API normalization for worker-attributed execution segments as the first implementation checkpoint.

If worker attribution is not queryable at the right granularity yet, do not force the frontend to infer it from nested step trees for the initial product. Prefer a dedicated backend projection or response tailored to timeline rendering.
