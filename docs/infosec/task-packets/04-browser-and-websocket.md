# TP-04 - Secure Browser Origins and Live Events

**Status:** Proposed; no owner assigned.
**Priority:** P1 for browser safeguards; human access enforcement is deferred.
**Accountable roles:** Application and identity owners.
**Findings:** 3, 5, and 6.
**Controls:** IAM-01/02/04/06, DATA-06, K8S-14, NET-16; tests V-04/05.
**Prerequisites:** D-02 for origin/host policy; D-04 and, for full enforcement,
the contracts from
[TP-02](02-project-ownership.md) and
[TP-03](03-human-access-and-authorization.md).

## Objective and current state

Prevent browser-origin bypasses and cross-project event delivery while preserving
authorized live updates. The [CORS middleware](../../../cmd/api/main.go#L205)
defaults to `*` and reflects any nonempty origin in its configured-list branch.
The [WebSocket upgrader](../../../pkg/websocket/websocket.go#L79) accepts every
origin; the [relay](../../../pkg/websocket/websocket.go#L341) consumes raw events
and has a raw-data fallback when normalization fails.

The P1 browser safeguards below can and should proceed before human sign-in
exists. An internal user visiting an untrusted site can still send browser
requests to a private or local endpoint; CORS alone does not prevent all
state-changing requests. These safeguards do not authenticate users or provide
project isolation. Record the remaining private-access risk in TP-16.

## Work and acceptance criteria

1. **P1:** Parse and validate an explicit origin allowlist, including scheme and
   port; allow only exact approved origins. Configure consistent CORS and
   WebSocket Origin rules. Never reflect arbitrary origins or combine wildcard
   CORS with credentialed access. Define a safe policy for missing Origin on
   approved non-browser clients; do not equate Origin with authentication.
2. **P1:** Protect state-changing HTTP routes, including currently unauthenticated
   delete and marker operations, against cross-origin browser requests. Validate
   origins and/or use another approved CSRF defense even when no session cookie
   exists; do not rely on CORS preflight alone. In the protected distributed
   deployment, reject missing or `null` Origin on unsafe methods by default
   unless an explicitly documented, effective alternative protects the approved
   non-browser flow. Do not infer that a client is non-browser merely because
   Origin is missing, or claim that an Origin header identifies it.
3. **P1:** Reject unapproved WebSocket origins and avoid broadcasting unfiltered
   raw events when normalization fails in the protected deployment. Reject or
   quarantine and log malformed events while preserving legitimate event formats
   and live updates. Coordinate Host/authority and trusted-proxy validation with
   [TP-09](09-http-and-proxy-hardening.md), including DNS-rebinding scenarios.
4. **Future access:** Enforce authentication before WebSocket upgrade and
   authorize each event against trusted project/run scope regardless of
   client-provided filters.
   Missing, unknown, or malformed scope must fail closed; an empty filter may
   never mean all projects.
5. **Future access:** Re-evaluate long-lived sessions on expiry and revocation
   within the approved D-04 bound. Close or reauthorize sessions and stop
   delivery when a grant is removed.
6. **Future access:** Implement session cookie/token protections and
   logout/expiry. Ensure mutation defenses remain compatible with the selected
   login flow.
7. Preserve legitimate same-origin and explicitly approved cross-origin browser
   behavior, including documented local Docker/AIO, Compose, and web-development
   flows. Enable and test the applicable origin/Host safeguards in local modes
   with those supported origins and hosts, and preserve supported unauthenticated
   local non-browser clients. Document any local compatibility exception and
   its security trade-off. Use profile-appropriate configuration so the
   distributed deployment does not silently inherit permissive development
   settings; any additional permissive development origin/host allowances must
   be explicit and local-only. Do not require local demos to run enterprise
   identity infrastructure.
8. Keep TP-09 responsible for general HTTP headers/timeouts and TP-03
   responsible for shared identity/REST policy; do not create divergent origin
   or project authorization implementations.

## Verification

- For the P1 milestone, test exact allowed/disallowed origins, scheme/port
  mismatches, missing and `null` Origin, preflight and credential headers,
  malicious cross-origin simple/form mutation requests without authentication,
  WebSocket upgrades, malformed events, Host/DNS-rebinding cases, and valid local
  browser and non-browser as well as distributed flows.
- For full access enforcement, additionally test authenticated and unauthenticated
  upgrades, cross-project and unfiltered subscriptions, expiry, revocation, and
  the measured reauthorization bound. Track this separately from the P1 milestone.
- Run `go test ./cmd/api ./pkg/websocket` and, for UI changes,
  `cd web && npm run build`. Exercise applicable V-04/05 cases through the actual
  gateway using synthetic events; do not claim V-05 passes before project
  filtering exists.
