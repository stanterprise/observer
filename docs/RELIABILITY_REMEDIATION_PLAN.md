# Processor Reliability Remediation Plan

## Problem Statement

Residual event mismatches were traced primarily to ordering/timing issues in processor persistence (for example, `step.*` arriving before `test.begin`) and insufficient terminal visibility for repeatedly failing messages.

## Completed in This Change

1. Orphan step defer queue

- Added an in-memory defer queue for orphan `step.begin` and `step.end` events.
- Orphan detection currently keys on repository errors such as `parent test not found` and `step not found`.
- Deferred events are replayed automatically when the corresponding `test.begin` succeeds.
- Queue behavior is bounded by configurable max attempts and TTL.

2. Configurable JetStream retry semantics

- Added processor configuration for `MaxDeliver` and `AckWait`.
- Values now flow from processor env/flags into consumer creation instead of using hardcoded values.

3. DLQ on terminal failure

- Added DLQ publish path when message delivery attempts reach `MaxDeliver`.
- DLQ payload includes reason, run ID, original subject, delivery count, and original event envelope.
- Message is acknowledged after DLQ publish attempt to prevent endless redelivery loops.

4. Integrity counters and run completeness summary

- Added per-run counters for received, processed, deferred, replayed, failed, DLQ, and pending deferred events.
- Added run-level completeness summary log emitted on `run.end` processing.

## Runtime Configuration

Processor now supports:

- `NATS_MAX_DELIVER` (default: `5`)
- `NATS_ACK_WAIT` (default: `30s`)
- `NATS_DLQ_SUBJECT` (default: `tests.events.v1.dlq`)
- `DEFER_QUEUE_MAX_ATTEMPTS` (default: `5`)
- `DEFER_QUEUE_TTL` (default: `5m`)

Equivalent flags:

- `-max-deliver`
- `-ack-wait`
- `-dlq-subject`
- `-defer-max-attempts`
- `-defer-ttl`

## Rollout Checklist

1. Deploy processor with explicit environment configuration in each environment.
2. Confirm JetStream stream subject pattern includes DLQ subject (default stream already uses `tests.events.v1.>`).
3. Verify run summaries appear in processor logs for completed runs.
4. Monitor DLQ volume and reasons; tune `NATS_MAX_DELIVER` and `NATS_ACK_WAIT` as needed.
5. Validate deferred queue replay behavior on known out-of-order test fixtures.

## Follow-up Work (Recommended)

1. Persist deferred queue state in MongoDB for crash-safe recovery.
2. Add dedicated DLQ consumer and API endpoint/UI for DLQ triage.
3. Export integrity counters as metrics (Prometheus) and alert on low completeness ratio.
4. Add integration tests that force out-of-order delivery and terminal failure scenarios.
