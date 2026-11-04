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
contains no completed submission identity or sign-off.

## Work and acceptance criteria

1. Assign named, accountable owners and escalation contacts for the application,
   platform, identity, data, operations, release, InfoSec, and risk decisions.
2. Record D-01 and D-02: environment, cluster, region, namespace, deployment
   revision, chart and image digests, dependency topology, gateway, hostnames,
   public-ingestion necessity, and source restrictions.
3. Record D-03 and D-04: reporter credential mechanism and lifecycle, human SSO
   and MFA, trust/identity propagation, project/run grants, roles, revocation
   bounds, and workload identities. Do not select token versus mTLS implicitly.
4. Record D-05 through D-09: data classes, permitted content, retention and legal
   holds; TLS trust and certificate ownership; numeric limits and alerts; backup
   scope, RPO/RTO and audit retention; patch SLAs and exception policy.
5. Link each decision to its approving role, date, scope, follow-up owner, and
   affected packet. Explicitly record unresolved choices as **Pending**; do not
   place secrets, personal data, or sensitive configuration in this repository.
6. Document the exposure gate and rollback/exception process. Do not approve
   public ingestion merely because a gateway or private namespace exists.

## Verification and handoff

- Review all nine decisions against
  [D-01 through D-09](../01-scope-and-ownership.md#required-deployment-decisions)
  with their accountable owners. Record decisions and sanitized evidence
  references in the deployment assessment, not as invented completed values.
- Confirm each affected packet has its required inputs or a named blocker.
- Hand the frozen deployment identity and ownership record to
  [TP-16](16-assessment-and-approval.md). Reassess after material exposure,
  identity, tenancy, storage, or dependency changes.
