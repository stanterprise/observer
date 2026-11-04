# TP-02 - Bind Project and Run Ownership End to End

**Status:** Proposed; no owner assigned.
**Priority:** P0; part of the public-ingestion gate.
**Accountable roles:** Application and data owners.
**Findings:** 2, plus the ownership prerequisite for 3, 6, and 7.
**Controls:** IAM-02/03, ING-02/04, DATA-05/06; tests V-03/05/08.
**Prerequisites:** D-03/D-04 and the principal contract from
[TP-01](01-machine-identity.md). Coordinate consumers with
[TP-04](04-browser-and-websocket.md), [TP-07](07-replay-and-event-delivery.md),
and [TP-10](10-retention-and-deletion.md).

## Objective and current state

Bind a run to one authorized project and carry that trusted scope from accepted
RPC to persistence and live delivery. The
[ingestion handlers](../../../pkg/server/server.go#L57) publish event-supplied
fields; the [event envelope](../../../pkg/publisher/nats.go#L134) contains type,
timestamp, and data but no trusted ownership. PostgreSQL stores a
`project_name` field in [runs](../../../migrations/000001_init.up.sql#L1), which
by itself is not a grant or an immutable ownership guarantee.

## Work and acceptance criteria

1. Define stable project and principal identifiers, authorized grants, run
   creation/ownership rules, and the authoritative grant source under D-04.
   Do not assume that a client project name, run ID, subscription filter, or
   mutable metadata grants access.
2. Reject unauthorized creation, writes to another project's run, and attempts
   to reassign ownership. Make create-or-check ownership atomic under concurrent
   submissions. Preserve legitimate same-project sharding, retries, and logical
   run aggregation.
3. Carry versioned trusted project/run scope and event identity through the
   publisher envelope, processor, PostgreSQL, and WebSocket relay. Restrict
   broker producers through TP-08; scope in a client-editable payload alone is
   not a trust boundary. Reject or quarantine missing, inconsistent, and
   unauthorized scope rather than widening access.
4. For existing events and stored runs, define an owner-approved migration,
   compatibility window, backfill, and handling for ambiguous legacy records.
   Never automatically grant access based solely on historical untrusted names.
   Add additive, reversible SQL migrations for any relational model change.
5. Expose one reusable authorization/scope contract for REST, mutations,
   artifact lookup, and WebSocket delivery. A chosen external identity provider
   need not imply a particular local user-table design.

## Verification

- Add unit and repository tests for cross-project run collisions, concurrent
  creation, ownership modification, sparse lifecycle events, valid same-project
  shards, and legacy/unscoped records. Run `go test ./pkg/server ./pkg/publisher
  ./pkg/consumer ./internal/repository/postgres ./tests`.
- With an isolated PostgreSQL and JetStream deployment, test an authorized
  event end to end and show a conflicting project cannot publish or persist it.
  Exercise old/new envelope and rollback behavior without exposing legacy data.
- Record sanitized V-03 and V-05 evidence; no blanket project authorization
  claim is complete until TP-03/04/05 enforce this scope on their surfaces.
