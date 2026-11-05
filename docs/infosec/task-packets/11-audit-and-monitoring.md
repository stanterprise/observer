# TP-11 - Security Audit, Safe Logs, and Alerts

**Status:** Proposed; no owner assigned.
**Priority:** P2 for full audit; basic ingestion and mutation signals start early.
**Accountable roles:** Application and operations owners.
**Findings:** 4, 14, and 27.
**Controls:** IAM-07, DATA-09, OPS-01 through OPS-04; tests V-10/13.
**Prerequisites:** D-07/D-08 for thresholds and retention; D-04 for future human
identity. Define a shared event contract early; integrate
the actual event sources from [TP-01](01-machine-identity.md),
[TP-03](03-human-access-and-authorization.md),
[TP-05](05-artifact-security.md), and
[TP-06](06-ingestion-limits.md) before completion.

## Objective and current state

Provide useful, protected security records and actionable operational signals
without copying credentials or test payloads into logs. The
[NATS publisher initialization](../../../pkg/publisher/nats.go#L83) logs the
configured URL, which may contain credentials. Existing gRPC request logging
and health endpoints are not, by themselves, a centralized audit trail or
proof that security alerts exist.

For the interim private-web/public-ingestion deployment, prioritize useful,
sanitized reporter-authentication, project-rejection, quota, and mutation
signals without waiting for TP-03. Record human actor as unknown until an
identity is actually verified. A source address may identify a proxy or a shared
network, not a person; source IP is not a substitute for human authentication
or a complete audit trail. The full package audit requirements remain applicable.

## Work and acceptance criteria

1. **Early milestone:** Record available reporter-authentication successes and
   failures, project rejections, throttling/quota outcomes, and run deletion or
   marker mutations. Include time, action, outcome, correlation ID, validated
   machine principal where available, and a correctly labeled source address
   only when obtained through the trusted proxy policy; otherwise record
   unknown or the proxy peer. Never invent a human principal. Identify whether
   the gateway or application emits each signal.
2. **Full audit:** Define structured events for authentication/authorization
   denials, privileged mutations, role/grant changes, deletion, artifact access
   as policy requires, credential changes, and migration execution. Include time,
   validated principal or explicit unknown, scope, action, outcome, and
   correlation ID. Inherit gateway/policy-change audit from the responsible
   platform when proven.
3. Redact connection URL credentials and sensitive query parameters, bearer
   tokens, signed URLs, raw test content, and sensitive error details across
   startup and request logs. Treat client-controlled titles/identifiers as
   untrusted; minimize and safely encode them.
4. Export logs to an approved restricted, time-synchronized, tamper-resistant
   destination with approved retention/residency. Define correlation across
   gateway, ingestion, broker, processor, and API without exposing payloads.
5. Instrument or collect denied access, reporter anomalies, throttling, queue
   age/depth, consumer/DLQ failures, dropped/rejected events, capacity, pod
   restarts, backup failures, certificate expiry, and unexpected exposure.
   A metrics endpoint is one option, not an assumed existing feature; protect
   any new endpoint and identify its collector.
6. Configure approved numeric thresholds, staffed alert destinations, severity,
   escalation, acknowledgement targets, and periodic response exercises.
   Events must remain observable if a downstream collector is unavailable under
   the approved loss/backpressure policy.

## Verification

- For the early milestone, test reporter outcomes, quota rejections, and mutations
  with verified or explicitly unknown actors and correctly attributed proxy
  sources. Verify that spoofed forwarded addresses are not recorded as trusted
  source addresses or treated as human identities.
- For full audit, add tests that generate allowed/denied human calls and
  mutations, assert structured outcomes and correlation, and use synthetic
  canaries to verify that tokens, signed URLs, connection credentials, and raw
  payloads are absent. Run the same redaction checks for the early milestone.
- Run `go test ./cmd/api ./pkg/server ./pkg/api ./pkg/publisher ./pkg/consumer
  ./pkg/websocket`. In a non-production deployment, execute V-13 using the real
  collector and alert route; verify access/tampering controls and retain only
  sanitized evidence. Mark human-attribution and full-audit tests pending until
  their identity and collection dependencies are actually available.
