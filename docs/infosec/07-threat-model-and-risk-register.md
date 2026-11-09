# 07 - Threat Model and Risk Register

**Status:** Design-stage threat scenarios, not findings of exploitable vulnerabilities.
Likelihood, residual risk, implementation status, and acceptance remain unassessed.

## Assets and adversaries

Protect confidentiality of test output/artifacts and credentials; integrity of run
ownership, results and audit records; availability of ingestion, processing and
queries; and recoverability of durable data.

Consider unauthenticated internet clients, compromised reporters/CI jobs, malicious
or over-privileged users, compromised workloads, administrators, and compromised
dependencies/build artifacts. Authentication does not make event content trustworthy.

Trust boundaries are external CI to public gateway; gateway to ingestion; web
access to API; workload to broker/data service; project to project; application to
artifact origin; and runtime to administrative/backup systems.

## Threat register

"Priority" denotes proposed design attention, not assessed vulnerability severity.
All rows have implementation status **Not assessed** and residual risk **Unknown**.

| ID | Scenario and impact | Priority | Required mitigations | Accountable role |
| --- | --- | --- | --- | --- |
| T-01 | Unauthorized or compromised reporter injects/overwrites results | High | IAM-02/03; ING-01/02/04/09; scoped identity and trusted ownership | Application + identity |
| T-02 | User/reporter accesses another project's runs, artifacts, or live events | High | IAM-02/04; DATA-05/06; cross-project negative tests | Application |
| T-03 | Credentials/personal data leak through logs, traces, screenshots, stream payloads, or downloads | High | DATA-01/02/05/09; retention; scoped access | Data + application |
| T-04 | Public ingestion exhausts memory, broker, storage, processing capacity, or budget | High | ING-03/05/06; K8S-10; quotas and saturation alerts | Application + operations |
| T-05 | Caller bypasses gateway or spoofs trusted identity headers | High | NET-02/04/14/15; IAM-03; direct-route denial tests | Platform + application |
| T-06 | Malicious artifact triggers active-content execution, path traversal, SSRF, or parser abuse | High | DATA-03/04/05; ING-03/05; isolated delivery and constrained fetching | Application |
| T-07 | Compromised pod moves laterally to DB, broker, secrets, or cluster API | High | IAM-05; NET-05 through NET-10; K8S-02 through K8S-08 | Platform |
| T-08 | Tampered or vulnerable image/chart/dependency compromises deployment | High | K8S-09; SUP-01 through SUP-05 | Release + platform |
| T-09 | Replay, reordering, redelivery, or poisoned events corrupt state or stall consumers | Medium | ING-03/04/06; bounded failure handling and recovery | Application |
| T-10 | Deleted/expired sensitive data survives in buffers, object versions, or restored backups | High | DATA-07/08; OPS-07/08; lifecycle/restore tests | Data + operations |
| T-11 | Credential/certificate expiry or dependency failure disrupts ingestion and recovery | Medium | ING-06; OPS-03/05/07; rotation and outage exercises | Operations |
| T-12 | Missing or tampered security logs prevent investigation | High | IAM-07; OPS-01/02/04; protected centralized audit | Operations |
| T-13 | Stolen session, CSRF, or unauthorized WS session permits actions/data access | High | IAM-01/04/06; expiry, origin and mutation tests | Application + identity |
| T-14 | Schema migration or backup misuse causes data loss or excessive privilege | High | K8S-13; OPS-07/08; separated migration/recovery identity | Release + operations |

## Assessment and risk acceptance record

Create one record per identified gap after assessment:

| Field | Required content |
| --- | --- |
| Record ID / threat / control | Unique gap ID; associated T-* and requirement IDs |
| Observed condition | Specific version/configuration, affected surface, evidence |
| Severity / likelihood / impact | Organizational rubric and rationale; do not inherit design priority automatically |
| Existing / compensating controls | Verified controls only, with evidence references |
| Residual risk | Assessed risk after verified controls |
| Remediation | Concrete action, named owner, target date, verification criteria |
| Acceptance | Authorized risk owner, rationale, approval date, expiry and conditions |
| Closure | Evidence, reviewer, closure date; required reassessment triggers |

## Unresolved design risks

D-01 through D-09 in [scope and ownership](01-scope-and-ownership.md) remain
open decisions, not accepted risks. In particular, selecting a public endpoint
without an evidenced machine-identity and project-authorization design is not an
approved deployment baseline.

Before approval, review each scenario with application, platform, data, and InfoSec
owners; add deployment-specific threats and abuse cases, then record residual risk.
