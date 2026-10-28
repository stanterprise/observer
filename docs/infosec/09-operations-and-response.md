# 09 - Operational Security and Response

**Status:** Required operating capabilities; named contacts and runbook evidence pending.

## Audit and monitoring

- **OPS-01:** Emit security events for authentication/authorization failures,
  privileged actions, project/role changes, deletion, artifact access as policy
  requires, credential changes, gateway/policy changes, and migration execution.
  Record time, validated principal, scope, action, outcome, and correlation ID.
  Minimize personal data; never record tokens, signed URLs, or raw test payloads.
- **OPS-02:** Centralize logs with restricted access, synchronized clocks,
  tamper-resistant retention, defined export/residency policy, and investigation
  correlation across gateway, ingestion, broker, processor, and API.
- **OPS-03:** Monitor denied access, unusual reporter activity, rate-limit hits,
  queue age/depth, consumer failures, dropped/rejected events, disk/object growth,
  pod restarts, backup failures, certificate expiry, and unexpected exposure.
  Define measurable alert thresholds and accountable responders.
- **OPS-04:** Deliver alerts to staffed channels with documented severity,
  escalation, acknowledgement targets, and periodic response exercises.
  Do not depend on planned application metrics endpoints as existing capabilities;
  identify actual collectors and any instrumentation work required.

## Credentials and maintenance

- **OPS-05:** Maintain issuance, inventory, rotation, revocation, and recovery
  runbooks for reporter credentials, service identities, certificates, database/
  broker/storage accounts, and encryption keys. Test propagation and client reload.
- **OPS-06:** Apply approved patch SLAs to all components, including dependency
  subcharts and gateways. Review changes, stage upgrades, assess schema/data
  compatibility, retain rollback artifacts, and reassess material security changes.

## Backup and recovery

- **OPS-07:** Establish approved RPO/RTO for durable records and artifacts. Encrypt
  backups, restrict restore privileges, separate backup identities, protect required
  immutable copies, and test restoration at an approved cadence. Include keys and
  configuration dependencies without placing secrets in ordinary backups/evidence.
- **OPS-08:** Define event replay/idempotency and cross-store reconciliation after
  failure. PostgreSQL remains authoritative; MongoDB buffers are not a durable
  substitute. Decide whether NATS backup/replay is required and define the effect of
  losing in-flight events. Restore must preserve artifact-reference consistency and
  reapply deletion policy before users regain access.

## Incident response playbook

| Phase | Required actions |
| --- | --- |
| Prepare | Assign on-call/incident commander, security contact, data owner and communication authority; maintain restricted contact directory |
| Detect/triage | Correlate identity, project/run, gateway and workload events; determine exposed data and time window; protect evidence |
| Contain reporter abuse | Revoke affected identity, block scoped submission, constrain gateway/quotas; preserve unrelated reporters where safe |
| Contain workload/data compromise | Isolate affected workload/namespace paths, revoke service credentials, restrict downloads, preserve forensic material |
| Eradicate/recover | Patch/redeploy trusted digests, rotate affected secrets/keys, restore or reconcile data, verify controls before reopening |
| Notify | Follow organizational/legal reporting policy; assess personal-data or credential exposure with data owner and InfoSec |
| Learn | Record root cause, scope, timeline, residual risk, remediation owners and deadlines; retest changed controls |

**OPS-09:** Approve this playbook and exercise credential compromise, sensitive
artifact exposure, and ingestion exhaustion scenarios. Define evidence custody and
safe retention; incident investigation must not spread confidential payloads.

## Operating record to complete

Record named service/platform/data/security owners; escalation channels; support
hours; patch SLAs; revocation latency; alert thresholds; audit retention; backup
schedule; RPO/RTO; last restore/response exercise; and links to access-restricted
runbooks. Missing entries remain **Pending**, not implicitly accepted.
