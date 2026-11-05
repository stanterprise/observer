# TP-02 - Bind Project and Run Ownership End to End

**Status:** Proposed; no owner assigned.
**Priority:** P0 for minimum machine project/run binding; P2 for broader scope.
**Accountable roles:** Application and data owners.
**Findings:** 2, plus the ownership prerequisite for 3, 6, and 7.
**Controls:** IAM-02/03, ING-02/04, DATA-05/06; tests V-03/05/08.
**Prerequisites:** D-03 machine grants and the principal contract from
[TP-01](01-machine-identity.md); D-04 is needed for future human grants.
Coordinate the P0 trusted broker path with
[TP-08](08-dependency-tls-and-credentials.md) and consumers with
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

Before public ingestion, a reporter must be limited to its approved projects and
permitted runs, with trusted scope carried through to the authoritative write.
This minimum is part of SCP-04 even if product AuthN/AuthZ is otherwise planned
for later. The enforcement location and grant source remain open; a gateway that
only authenticates a caller does not by itself satisfy project/run authorization.
Broader human-facing enforcement is later work and does not become complete
merely because machine submission is protected.

## Work and acceptance criteria

1. **P0:** Define stable project and machine-principal identifiers, authorized
   grants, run creation/ownership rules, and the authoritative grant source under
   D-03, coordinating the future human contract under D-04.
   Do not assume that a client project name, run ID, subscription filter, or
   mutable metadata grants access.
2. **P0:** Reject unauthorized creation, writes to another project's run, and
   attempts to reassign ownership. Make create-or-check ownership atomic through
   the authoritative write under concurrent submissions. Preserve legitimate
   same-project sharding, retries, and logical run aggregation.
3. **P0:** Carry versioned trusted project/run scope and event identity through
   the publisher envelope, processor, and PostgreSQL. Restrict broker producers
   through TP-08. If another trusted path is proposed for P0, demonstrate its
   isolation and separately assess the package's broker-credential and subject
   requirements. Any control relied on for P0 ownership must be ready before
   public exposure. Scope in a client-editable payload alone is not a trust
   boundary. In the protected deployment, reject or quarantine missing,
   inconsistent, and unauthorized scope rather than widening access.
4. **P0 for affected intake:** For existing events and stored runs, define an
   owner-approved migration, compatibility window, backfill, and handling for
   ambiguous legacy records.
   Never automatically grant access based solely on historical untrusted names.
   Add additive, reversible SQL migrations for any relational model change.
5. **P2/future access:** Carry trusted scope into WebSocket delivery and expose
   one reusable authorization/scope contract for REST, mutations, artifact
   lookup, and human access. Complete per-user enforcement with TP-03/04/05.
   A chosen external identity provider need not imply a particular local
   user-table design. Preserve supported unauthenticated local demo workflows;
   any development-only compatibility configuration must not silently grant
   access in the protected distributed deployment.

## Verification

- Add unit and repository tests for cross-project run collisions, concurrent
  creation, ownership modification, sparse lifecycle events, valid same-project
  shards, and legacy/unscoped records. Run `go test ./pkg/server ./pkg/publisher
  ./pkg/consumer ./internal/repository/postgres ./tests`.
- With an isolated PostgreSQL and JetStream deployment, test an authorized
  event end to end. Show that an ungranted reporter cannot publish and a
  conflicting project cannot persist or change another project's run, including
  concurrent submissions and gateway-bypass attempts. If an ownership conflict
  is detected asynchronously, verify its explicit, observable failure handling
  and document what the earlier RPC acknowledgement guarantees. Exercise
  old/new envelope, rollback, and supported local demo behavior.
- Record P0 machine-binding evidence separately from later V-05/human-scope
  evidence; no blanket project-authorization claim is complete until TP-03/04/05
  enforce this scope on their surfaces. A time-bounded exception, if approved,
  is not evidence that the missing binding is implemented.
