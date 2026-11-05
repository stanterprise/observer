# TP-16 - Complete the Control Assessment and Approval

**Status:** Proposed; no owner assigned.
**Priority:** Ongoing; exposure and final-approval gates remain unchanged.
**Accountable roles:** InfoSec reviewer, application/platform/data/operations
owners, and authorized business risk owner.
**Controls:** Every applicable SCP, NET, IAM, ING, DATA, K8S, SUP, and OPS
requirement; tests V-01 through V-15.
**Prerequisites:** [TP-00](00-decisions-and-scope.md) and the deliverables and
deployment evidence from the other applicable packets.

## Objective and current state

Turn the proposed requirements and remediation outputs into a deployment-specific
assessment. The [assessment document](../10-assessment-evidence.md) is an
unfilled template; neither this packet set nor repository code alone constitutes
approval or proof that platform-owned controls are present. Deferred human
AuthN/AuthZ and any delayed internal TLS remain gaps against applicable package
requirements unless a deployment-specific assessment demonstrates otherwise.

## Work and acceptance criteria

1. Freeze the assessment identity: source revision, chart/package and dependency
   versions, image digests, gateway/CNI/cluster versions, environment,
   configuration, identities, storage, and owners. Link sanitized deployment
   diagrams and the actual network/data-flow matrix.
2. Create and review one row for **every** requirement in the package:
   SCP-01 through SCP-05, NET-01 through NET-19, IAM-01 through IAM-07,
   ING-01 through ING-09, DATA-01 through DATA-09, K8S-01 through K8S-16,
   SUP-01 through SUP-05, and OPS-01 through OPS-09. Use the allowed statuses
   from the assessment template and assign an owner and evidence/test reference.
   Mark not applicable only with reviewer-approved rationale.
3. Assemble E-01 through E-12, including identity/grants, policy and ACLs,
   retention/encryption, rendered manifests, SBOMs and scans, test results,
   response/restore records, and the decision on independent testing. Evidence
   must identify its collector, time, target revision/configuration, tool version,
   limitations, and restricted location; do not attach secrets or real payloads.
4. Execute the applicable [V-01 through V-15](../08-supply-chain-and-verification.md#verification-plan)
   using authorized synthetic-data tests. Include negative and quantitative
   boundary results, record skipped or blocked tests honestly, and obtain
   InfoSec's penetration-test decision and retest where required.
5. Create a [risk record](../07-threat-model-and-risk-register.md#assessment-and-risk-acceptance-record)
   for each remaining gap with severity rationale, compensating controls,
   remediation owner/date, reviewer, and any authorized time-bounded exception.
   In particular, assess the deferred human SSO, REST/mutation authorization,
   artifact access, and WebSocket project isolation against applicable controls
   such as NET-03/04, IAM-01/02/04, and DATA-05/06. Assess each delayed
   dependency-TLS path separately. Private network access, source IP, a planned
   future feature, or D-06 alone is not an exception or a reason to mark an
   applicable control **Not applicable**. Risk acceptance does not turn a
   failed control into **Met**.
6. Obtain the required design, assessment, exposure, operations, and risk
   approvals from named owners. In particular, apply SCP-04 before public
   ingestion, including reporter authentication, machine project/run
   authorization, TLS, gateway-bypass prevention, and abuse limits. Do not use
   packet completion, priority, or a gateway as a substitute for evidence or an
   authorized exception. Passing the exposure gate alone is not final approval.
7. Keep local Docker/AIO and Compose outside this assessment while recording
   compatibility testing for changes shared with those supported workflows; do
   not claim local mode meets the distributed deployment requirements.

## Verification

- Reconcile the complete requirement and E-01 through E-12 inventories against
  [document 10](../10-assessment-evidence.md); show no missing row, owner, status,
  evidence reference, or unresolved disposition.
- Verify evidence applies to the frozen deployment and that V-01 through V-15
  have an explicit pass, fail, blocked, not-run, or reviewer-approved
  not-applicable disposition. Record approvals and reassessment triggers without
  asserting they already exist. Check each exception's exact control, scope,
  approver, compensating controls, expiry, and remediation owner; a proposed
  exception is not an approved exception.
