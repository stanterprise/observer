# TP-00 - Resolve Deployment Decisions and Assign Ownership

**Status:** Proposed; no owner assigned.
**Priority:** P0.
**Accountable roles:** Application, platform, identity, data, operations, release,
InfoSec, and business risk owners.
**Controls:** SCP-01 through SCP-05; inputs D-01 through D-09.
**Prerequisites:** None. Related findings and scope: [packet index](README.md).

## Objective and current state

Establish the concrete deployment and policy inputs needed to implement and
assess the other packets. The [scope document](../01-scope-and-ownership.md)
marks every decision pending; the [evidence template](../10-assessment-evidence.md)
contains no completed submission identity or sign-off. The user's intended
private organizational deployment, public CI ingestion, deferred product
AuthN/AuthZ, and local-mode compatibility are planning inputs in the
[packet index](README.md#planning-assumptions-and-compatibility), not approved
deployment decisions.

## Work and acceptance criteria

1. Assign named, accountable owners and escalation contacts for the application,
   platform, identity, data, operations, release, InfoSec, and risk decisions.
2. Record D-01 and D-02: environment, cluster, region, namespace, deployment
   revision, chart and image digests, dependency topology, gateway, hostnames,
   public-ingestion necessity, private web/API access, and source restrictions.
   Verify all alternate exposure and gateway-bypass paths rather than assuming
   the surrounding infrastructure blocks them.
3. Record D-03 and D-04 separately for machine and human access: reporter
   credential mechanism, project/run grants and lifecycle; future human SSO,
   MFA and roles; trust/identity propagation, revocation bounds, and workload
   identities. Select where machine checks execute without assuming a particular
   token, mTLS, or application identity provider.
4. Record D-05 through D-09: data classes, permitted content, retention and legal
   holds; TLS trust and certificate ownership; numeric limits and alerts; backup
   scope, RPO/RTO and audit retention; patch SLAs and exception policy.
   Deferred dependency TLS still requires the applicable exception and approval;
   D-06 alone does not waive the package.
5. Document the lack of human identity and per-user authorization as a proposed
   risk disposition, including reachable operations, compensating network and
   browser controls, accountable owner, and required exception process. Count
   compensating controls only when their actual enforcement is evidenced; do
   not describe private access or source IP as proof of individual identity.
6. Preserve supported local Docker/AIO and Compose workflows without making them
   part of this assessment. Note any configuration or migration needed for
   shared changes; do not silently change local port exposure or require the
   distributed identity infrastructure.
7. Link each decision to its approving role, date, scope, follow-up owner, and
   affected packet. Explicitly record unresolved choices as **Pending**; do not
   place secrets, personal data, or sensitive configuration in this repository.
8. Document the SCP-04 exposure gate and rollback/exception process. Do not
   approve public ingestion merely because a gateway or private namespace exists,
   or treat deferred human access work as an exception to machine authorization.

## Verification and handoff

- Review all nine decisions against
  [D-01 through D-09](../01-scope-and-ownership.md#required-deployment-decisions)
  with their accountable owners. Record decisions and sanitized evidence
  references in the deployment assessment, not as invented completed values.
- Confirm each affected packet has its required inputs or a named blocker.
- Review assumptions and any proposed exception with the responsible approvers;
  record a pending decision or exception as pending, not accepted.
- Hand the frozen deployment identity and ownership record to
  [TP-16](16-assessment-and-approval.md). Reassess after material exposure,
  identity, tenancy, storage, or dependency changes.
