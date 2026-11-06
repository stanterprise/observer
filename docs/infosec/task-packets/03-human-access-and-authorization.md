# TP-03 - Protect Human Access and REST Operations

**Status:** Proposed; no owner assigned.
**Priority:** Deferred future work; package requirements remain applicable.
**Accountable roles:** Application and identity owners.
**Findings:** 3 and 4.
**Controls:** IAM-01/02/03/07, DATA-05; tests V-03/04/05.
**Prerequisites:** D-04 and the project/run contract from
[TP-02](02-project-ownership.md). Coordinate browser/session handling with
[TP-04](04-browser-and-websocket.md), artifacts with
[TP-05](05-artifact-security.md), and audit events with
[TP-11](11-audit-and-monitoring.md).

## Objective and current state

Require approved human identity and enforce project/resource authorization on
every application route, not only at a gateway. The
[API router](../../../cmd/api/main.go#L99) registers routes without an
application auth layer. [REST routes](../../../pkg/api/rest_postgres.go#L31)
include list/detail/statistics queries and delete/marker mutations. Current
project-name columns alone do not provide a role or grant model.

The intended interim deployment restricts web/API access at the organizational
network boundary. That does not identify a human, enforce project grants, or
prevent a reachable user from reading and mutating out-of-scope data. Gateway
SSO may be assessed as an interim human-authentication control, but it does not
by itself establish resource authorization. Record each gap and any required
owner-approved, time-bounded exception in [TP-16](16-assessment-and-approval.md);
do not treat this packet as complete or its controls as **Met** while deferred.
Machine authentication and project/run authorization for public ingestion remain
separate P0 work in TP-01/02.

## Work and acceptance criteria

1. Integrate the selected organizational SSO and policy-required MFA, covering
   static UI, REST, WebSocket, artifact routes, and alternate hostnames. Define
   session expiry/logout and a documented validated-identity trust boundary.
   If identity is forwarded, prevent header spoofing and backend bypass.
2. Define a deny-by-default role/project action matrix. Enforce it server-side
   in shared middleware and repository queries, including list/search, counts,
   trends, aggregates, detail, markers, and all mutation paths. No client filter
   may expand the caller's grants.
3. Require the explicit privilege for run deletion and marker changes. For a
   mixed authorized/unauthorized batch, use a documented fail-closed behavior
   that avoids partial cross-project mutation or existence leakage. Integrate
   deletion with TP-10 and emit auditable outcomes through TP-11.
4. Preserve project separation for anonymous users, viewers, maintainers,
   administrators, revoked users, and multi-project users. Make unauthorized and
   nonexistent resource behavior consistent with the approved disclosure policy.
5. Provide an owner-approved migration/access plan for existing data and
   operators. Add corresponding SQL migrations for relational changes; do not
   invent a local identity schema if the approved authoritative grant model does
   not require one. Preserve supported local demos without requiring an
   enterprise identity provider.

## Verification

- Add route and repository tests with at least two projects and each selected
  role. Cover individual IDs, list/aggregate leakage, mixed-ID batches, denied
  delete/marker operations, unauthenticated access, and valid in-scope behavior.
  Run `go test ./cmd/api ./pkg/api ./internal/repository/postgres`.
- Exercise the protected UI and alternate routes in the selected gateway
  deployment; run `cd web && npm run build` if UI code changes.
- Record V-03/04 results and link V-05/V-08 evidence from the WebSocket and
  artifact packets. SSO login success alone is not project authorization.
