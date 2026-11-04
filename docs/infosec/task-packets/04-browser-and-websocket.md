# TP-04 - Secure Browser Origins and Live Events

**Status:** Proposed; no owner assigned.
**Priority:** P0.
**Accountable roles:** Application and identity owners.
**Findings:** 3, 5, and 6.
**Controls:** IAM-01/02/04/06, DATA-06, K8S-14, NET-16; tests V-04/05.
**Prerequisites:** D-02/D-04 and, for full enforcement, the contracts from
[TP-02](02-project-ownership.md) and
[TP-03](03-human-access-and-authorization.md).

## Objective and current state

Prevent browser-origin bypasses and cross-project event delivery while preserving
authorized live updates. The [CORS middleware](../../../cmd/api/main.go#L205)
defaults to `*` and reflects any nonempty origin in its configured-list branch.
The [WebSocket upgrader](../../../pkg/websocket/websocket.go#L79) accepts every
origin; the [relay](../../../pkg/websocket/websocket.go#L341) consumes raw events
and has a raw-data fallback when normalization fails.

## Work and acceptance criteria

1. Parse and validate an explicit origin allowlist, including scheme and port;
   allow only exact approved origins. Configure consistent CORS and WebSocket
   Origin rules. Never reflect arbitrary origins or combine wildcard CORS with
   credentialed access. Define a safe policy for missing Origin on approved
   non-browser clients; do not equate Origin with authentication.
2. Enforce authentication before WebSocket upgrade and authorize each event
   against trusted project/run scope regardless of client-provided filters.
   Missing, unknown, or malformed scope must fail closed; an empty filter may
   never mean all projects.
3. Re-evaluate long-lived sessions on expiry and revocation within the approved
   D-04 bound. Close or reauthorize sessions and stop delivery when a grant is
   removed. Do not fall back to broadcasting unfiltered raw events on parse or
   normalization errors.
4. Implement session cookie/token protections and logout/expiry. If cookies
   authorize mutations, add CSRF defenses compatible with the selected login
   flow. Preserve legitimate same-origin and explicitly approved cross-origin
   browser behavior; document any development-only configuration.
5. Keep TP-09 responsible for general HTTP headers/timeouts and TP-03
   responsible for shared identity/REST policy; do not create divergent origin
   or project authorization implementations.

## Verification

- Add tests for exact allowed/disallowed origins, scheme/port mismatches, missing
  Origin, preflight and credential headers, CSRF where applicable, authenticated
  and unauthenticated upgrades, cross-project and unfiltered subscriptions,
  malformed events, expiry, revocation, and valid live delivery.
- Run `go test ./cmd/api ./pkg/websocket` and, for UI changes,
  `cd web && npm run build`. Exercise V-04/05 through the actual gateway using
  synthetic events and record the measured reauthorization bound.
