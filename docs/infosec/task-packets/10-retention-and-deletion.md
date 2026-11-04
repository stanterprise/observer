# TP-10 - Implement Retention and Cross-Store Deletion

**Status:** Proposed; no owner assigned.
**Priority:** P1.
**Accountable roles:** Data, application, and operations owners.
**Findings:** 13.
**Controls:** DATA-01/07/08, OPS-07/08; test V-11.
**Prerequisites:** D-05/D-08, trusted ownership and mutation authorization from
[TP-02](02-project-ownership.md) and
[TP-03](03-human-access-and-authorization.md), and replay policy from
[TP-07](07-replay-and-event-delivery.md). Coordinate restore drills with
[TP-15](15-recovery-and-response.md).

## Objective and current state

Implement an approved, auditable lifecycle for all copies of run data and
artifacts, including after restore. The [run delete handler](../../../pkg/api/rest_postgres.go#L308)
calls the [PostgreSQL deletion transaction](../../../internal/repository/postgres/postgres_query_mutations.go#L12),
which removes the relational run graph but does not coordinate object deletion.
MongoDB has a [TTL index](../../../internal/database/mongodb.go#L232) for live
buffers; that index alone does not establish an approved orphan lifetime or
cross-store deletion guarantee.

## Work and acceptance criteria

1. Translate D-05/D-08 into retention, deletion-lag, legal-hold, backup-horizon,
   and audit-record rules for PostgreSQL, artifacts and versions, MongoDB live
   buffers, JetStream, caches, logs, and backups. Record external artifact copies
   and storage-provider constraints rather than claiming control over them.
2. Design a durable, idempotent deletion workflow that first enforces TP-03
   authorization and prevents further access. Preserve enough object references
   and progress information to retry failures after database changes; never
   report full deletion while required object deletion has failed.
3. Prevent subsequent broker replay, stale buffers, cache entries, signed URLs,
   or retries from restoring access or resurrecting deleted runs. Account for
   already-issued signed URLs and the selected broker/object-version deletion
   semantics in the approved deletion SLO and exceptions.
4. Implement approved retention and orphan cleanup, legal holds, and durable
   tombstones/deletion records as needed. Ensure tombstones cover the backup
   horizon and are reapplied before restored data becomes accessible. Add
   reversible SQL migrations for relational changes.
5. Document status, retries, alerts, reconciliation, and failure recovery
   without logging sensitive artifacts. Preserve MongoDB's transient role and
   PostgreSQL's authoritative role.

## Verification

- Add tests for authorized deletion, cross-project denial, shared objects,
  failure between database and object deletion, retries, duplicate requests,
  legal holds, orphan buffers, replay after deletion, object versions, and a
  backup restore that reapplies deletion records.
- Run `go test ./pkg/api ./pkg/storage ./pkg/consumer
  ./internal/repository/postgres ./internal/database` and the selected
  PostgreSQL/MongoDB/object-store integration tests.
- Execute V-11 with synthetic data in an isolated environment; measure deletion
  lag and verify each selected store and restored backup against D-05/D-08.
