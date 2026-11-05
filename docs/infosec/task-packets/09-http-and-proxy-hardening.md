# TP-09 - Harden HTTP Servers and Reverse Proxies

**Status:** Proposed; no owner assigned.
**Priority:** P1.
**Accountable roles:** Application and platform owners.
**Findings:** 12.
**Controls:** NET-04/16/17, IAM-06, K8S-14, DATA-09; tests V-04/09/10.
**Prerequisites:** D-02/D-06/D-07. Coordinate origin/session policy with
[TP-04](04-browser-and-websocket.md), artifact-specific headers with
[TP-05](05-artifact-security.md), and gateway behavior with
[TP-14](14-network-and-platform.md).

## Objective and current state

Bound HTTP resource usage and make proxy and browser security behavior explicit.
The [API HTTP server](../../../cmd/api/main.go#L151) sets its address and handler
without explicit read/header/idle timeouts. The distributed web server is
configured in [nginx.web.conf.template](../../../docker/nginx/nginx.web.conf.template).
Security headers and trusted-proxy behavior must be evaluated across both the
application and the selected gateway.

## Work and acceptance criteria

1. Configure reviewed request-header, read, write, and idle timeouts and
   endpoint-appropriate request-body limits. Handle slow clients and oversized
   requests explicitly without breaking valid gRPC proxying, large approved
   requests, WebSocket upgrades, or long-lived authorized sessions.
2. Set an appropriate CSP, content-type and framing protections, and other
   organization-required headers on the correct UI/API responses. Apply HSTS
   only at the actual HTTPS boundary under the approved hostname policy.
   Avoid contradictory headers or a CSP that silently breaks the UI.
3. Define trusted proxy hops, scheme/host handling, and sanitized forwarded
   headers. Reject spoofed identity and forwarded metadata from untrusted
   sources. Validate allowed HTTP Host/authority values at the appropriate web,
   gateway, and direct API boundaries to mitigate DNS rebinding; do not trust a
   forwarded host supplied by an untrusted client. Support configured
   deployment hostnames and ports rather than hardcoding one production name.
   Keep public health/debug/version responses minimal and prevent them from
   creating an alternate application-data route.
4. Document which layer owns each header, timeout, size limit, and origin check.
   Keep general origin/CSRF policy in TP-04 and content-specific artifact
   handling in TP-05. Preserve documented local Docker/AIO, Compose, and
   web-development hostnames, ports, API, and WebSocket workflows when shared
   proxy or server code changes.

## Verification

- Add tests for slow headers, body limits at and above approved thresholds,
  forwarded-header spoofing, allowed and unapproved Host/authority values,
  DNS-rebinding to local and private addresses, response headers, preflight
  behavior, health responses, and legitimate local and distributed WebSocket
  upgrades/timeouts. Run
  `go test ./cmd/api ./pkg/api ./pkg/websocket`.
- If UI or nginx configuration changes, run `cd web && npm run build` and test
  the rendered UI plus API/WS paths through the selected gateway. Record V-04,
  V-09, and V-10 results with the actual proxy configuration.
