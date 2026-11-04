# TP-11 - Security Audit, Safe Logs, and Alerts

**Status:** Proposed; no owner assigned.
**Priority:** P1.
**Accountable roles:** Application and operations owners.
**Findings:** 4, 14, and 27.
**Controls:** IAM-07, DATA-09, OPS-01 through OPS-04; tests V-10/13.
**Prerequisites:** D-04/D-07/D-08. Define a shared event contract early; integrate
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

## Work and acceptance criteria

1. Define structured events for authentication/authorization denials, privileged
   mutations, role/grant changes, deletion, artifact access as policy requires,
   credential changes, and migration execution. Include time, validated
   principal or explicit unknown, scope, action, outcome, and correlation ID.
   Inherit gateway/policy-change audit from the responsible platform when proven.
2. Redact connection URL credentials and sensitive query parameters, bearer
   tokens, signed URLs, raw test content, and sensitive error details across
   startup and request logs. Treat client-controlled titles/identifiers as
   untrusted; minimize and safely encode them.
3. Export logs to an approved restricted, time-synchronized, tamper-resistant
   destination with approved retention/residency. Define correlation across
   gateway, ingestion, broker, processor, and API without exposing payloads.
4. Instrument or collect denied access, reporter anomalies, throttling, queue
   age/depth, consumer/DLQ failures, dropped/rejected events, capacity, pod
   restarts, backup failures, certificate expiry, and unexpected exposure.
   A metrics endpoint is one option, not an assumed existing feature; protect
   any new endpoint and identify its collector.
5. Configure approved numeric thresholds, staffed alert destinations, severity,
   escalation, acknowledgement targets, and periodic response exercises.
   Events must remain observable if a downstream collector is unavailable under
   the approved loss/backpressure policy.

## Verification

- Add tests that generate allowed/denied calls and mutations, assert structured
  outcomes and correlation, and use synthetic canaries to verify that tokens,
  signed URLs, connection credentials, and raw payloads are absent.
- Run `go test ./cmd/api ./pkg/server ./pkg/api ./pkg/publisher ./pkg/consumer
  ./pkg/websocket`. In a non-production deployment, execute V-13 using the real
  collector and alert route; verify access/tampering controls and retain only
  sanitized evidence.
