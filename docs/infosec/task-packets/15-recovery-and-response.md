# TP-15 - Prepare Recovery, Rotation, and Incident Response

**Status:** Proposed; no owner assigned.
**Priority:** P2 for full recovery and response; reporter revocation starts
with the P0 ingestion gate.
**Accountable roles:** Operations, data, identity, and InfoSec owners.
**Findings:** 28.
**Controls:** OPS-05 through OPS-09, DATA-07/08, ING-06; tests V-10/11/14.
**Prerequisites:** D-03 and [TP-01](01-machine-identity.md) for early reporter
revocation. Full exercises require D-05/D-06/D-08/D-09 and the relevant
implementations from [TP-07](07-replay-and-event-delivery.md),
[TP-08](08-dependency-tls-and-credentials.md),
[TP-10](10-retention-and-deletion.md), and
[TP-11](11-audit-and-monitoring.md).

## Objective and current state

Turn the [operations requirements](../09-operations-and-response.md) into
approved, executable runbooks and tested recovery procedures for the selected
deployment. The package currently states that contacts and runbook evidence are
pending; it is not proof that backups, alerts, or response exercises exist.

Establish and verify the minimum reporter credential issuance, revocation,
compromise containment, and escalation needed by TP-01 before public ingestion.
Schedule the broader backup, restore, and incident exercises under D-05/D-08;
elevate them if data policy, operational risk, or an approval gate requires it.
This sequencing does not waive the package's operating requirements.

## Work and acceptance criteria

1. Assign an on-call owner, escalation channels, incident commander, data and
   security contacts, and an evidence-custody process. Approve an incident
   playbook for reporter compromise, sensitive artifact exposure, and ingestion
   exhaustion, including containment, notification, recovery, and retesting.
2. Document and exercise issuance, inventory, rotation, revocation, reload, and
   recovery for reporter credentials, service identities, certificates, DB and
   broker credentials, storage credentials, and encryption keys. Record actual
   propagation latency against approved D-03/D-06 bounds.
3. Define encrypted backup scope, schedule, immutable copies where required,
   separate backup/restore privileges, RPO/RTO, keys, configuration dependencies,
   and restoration cadence. Test PostgreSQL and artifacts together and decide
   whether NATS backup/replay is required.
4. Document outage, bounded retry, replay/idempotency, DLQ recovery, and
   cross-store reconciliation. PostgreSQL remains authoritative; MongoDB live
   buffers are not a durable substitute. Reapply deletion/tombstone state before
   restoring user access.
5. Exercise upgrade, schema migration, rollback, patch, and emergency rebuild
   procedures against approved compatibility and recovery targets. Keep
   production credentials, payloads, and sensitive incident evidence out of
   public runbooks and ordinary review exports.

## Verification

- Run authorized non-production rotation, compromise, outage, restore, and
  rollback exercises with synthetic data; record start/end times, measured
  RPO/RTO and revocation latency, data-integrity checks, failures, owner, and
  remediation.
- Execute the applicable portions of V-10/11/14 and the three OPS-09 scenarios.
  Link restricted runbooks and sanitized evidence in
  [TP-16](16-assessment-and-approval.md); planned exercises are not passes.
  Record the early reporter-revocation result separately from full packet
  completion.
