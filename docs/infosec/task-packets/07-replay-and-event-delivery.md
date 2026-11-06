# TP-07 - Reliable, Bounded Event Delivery

**Status:** Proposed; no owner assigned.
**Priority:** P1; stream bounds also support the P0 public-ingestion gate.
**Accountable roles:** Application and operations owners.
**Findings:** 9 and 10.
**Controls:** ING-04/05/06/09, DATA-07, OPS-08; tests V-06/07/14.
**Prerequisites:** D-07 for stream capacity and D-05 for stream retention;
D-03 and trusted scope from [TP-02](02-project-ownership.md) for full replay.
Stream bounds can proceed independently of human access. Coordinate ingress
limits with [TP-06](06-ingestion-limits.md) and recovery/deletion with
[TP-10](10-retention-and-deletion.md) and
[TP-15](15-recovery-and-response.md).

## Objective and current state

Define replay and ordering semantics, prevent duplicate durable effects, and
bound streams without silently discarding accepted events. The
[publisher](../../../pkg/publisher/nats.go#L96) creates a 24-hour stream but
does not explicitly set max bytes or message size, and returns early if a stream
already exists. The [consumer](../../../pkg/consumer/nats_consumer.go#L250)
configures retry/ack limits and a DLQ; in the
[DLQ failure path](../../../pkg/consumer/nats_consumer.go#L300) it logs a
failed DLQ publish and still acknowledges the original message.

Set and verify the stream bounds needed for the public-ingestion abuse gate at
P0. The remaining delivery/idempotency and DLQ work is P1; its priority does
not waive package controls or make data-loss risk acceptable. Record the
partial milestone separately from full packet completion.

## Work and acceptance criteria

1. Define a stable event identity, ownership scope, retry behavior, allowed age
   and ordering, conflict handling, and replay horizon. Ensure a retry of the
   same event is distinguishable from a different legitimate lifecycle event.
   Preserve current run aggregation, shards, retries, and recognized event types.
2. Implement durable, concurrency-safe idempotency for persistence and side
   effects through the approved replay horizon. Broker deduplication such as a
   NATS message ID may help, but its finite window alone is not sufficient.
   Prevent duplicates from bypassing quotas or reviving deleted data.
3. **P0:** Configure and validate stream max age, bytes, message size, capacity,
   replicas, and reject/discard behavior against approved policy. Handle
   pre-existing streams explicitly: verify compatibility and update safely or
   fail with an actionable error. Never silently keep weaker existing bounds.
4. **P1:** Ensure processor failures, exhausted retries, and DLQ failures have
   durable, observable outcomes. A failed DLQ write must not be followed by a
   success-shaped acknowledgement that silently loses the original event.
   Bound retries and document operator recovery; retain the established
   acknowledgement policy for genuinely irrelevant or unknown event types.
5. Document acceptance versus database-commit guarantees and replay/recovery
   procedures. Do not store credentials or raw sensitive payloads in diagnostics.
   Preserve supported local development and reporter workflows when introducing
   production-specific stream settings.

## Verification

- Add tests for duplicate delivery before/after the broker dedup window,
  concurrent delivery, event-ID collision with different payloads, reordering,
  legitimate repeated events, deletion/replay, stream-at-capacity, existing stream
  mismatch, and DLQ publication failure.
- Run `go test ./pkg/publisher ./pkg/consumer ./tests`. With an isolated NATS
  instance, run `NATS_TEST_URL=nats://localhost:4222 go test ./tests/... -run TestNATSIntegration -v`
  and the new integration cases; tests skipped without a broker do not count as
  integration evidence.
- Record V-06/07/14 results, including the selected retention and exact capacity
  behavior. Identify which results establish the P0 stream-bounds milestone and
  which remain blocked on later replay or recovery work.
