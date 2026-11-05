# InfoSec Remediation Task Packets

**Status:** Proposed; no packet is approved or closed. TP-01 shared-key
authentication and TP-06 reflection changes are partially implemented; their
remaining controls and deployment evidence are open. Owners are unassigned.
**Baseline:** Repository revision `f03b3511135400efe61bc9077c9ab805c69d9aa7`,
chart `0.8.3` / app version `v0.8.3`, reviewed 2026-11-04.
**Scope:** Distributed Observer deployed with Helm, plus the operator-owned
platform controls in the [InfoSec package](../README.md). Local Docker, AIO, and
Compose are not assessed deployments, but changes must preserve their supported
demo and development workflows.

These are implementation inputs, not a penetration test, an exhaustive control
assessment, deployment approval, or evidence that a control is met. File paths and
observations describe the baseline above; recheck them before implementation.
Every owner, deployment decision, numeric limit, policy, environment-specific
control, and final status remains pending until explicitly recorded and reviewed.

## Planning assumptions and compatibility

- Observer is a utility intended primarily for Kubernetes inside an organization's
  protected infrastructure. The web UI, API, WebSocket, and artifacts are intended
  to be restricted to approved private/corporate access; verify the actual network
  controls rather than assuming a namespace or `ClusterIP` provides that restriction.
- Cloud-hosted CI runners may need a public gRPC endpoint. Protect that path with
  authenticated reporters, project/run authorization, TLS, gateway-bypass prevention,
  and abuse limits as required by SCP-04. The selected initial authentication
  mechanism is a deployment-wide shared bearer key, intended only as an
  ingestion gate; any holder may submit under any client-supplied project name.
  It is not reporter identity or project authorization, so it does not alone
  satisfy the public-ingestion gate.
- Product AuthN/AuthZ is planned for future work. Defer human sign-in and
  per-user/project access in TP-03, while treating the machine protections above
  as a separate exposure requirement. Private network access is not human identity
  or resource authorization. Until those controls exist, reachable users may be
  able to read or mutate data beyond their intended scope.
- These are planning inputs, not approved D-01 through D-09 decisions, evidence,
  or waivers. The unchanged package still requires human SSO and server-side
  authorization. Record affected controls and obtain any required, explicitly
  approved, time-bounded exceptions before the applicable deployment or exposure
  gate; an exception does not make an unmet control **Met**.
- Do not impose an identity provider, enterprise gateway, or ingestion key on
  supported local AIO/Docker/Compose demos. AIO/local mode remains keyless;
  distributed mode requires an operator-managed key. Test local ingestion and
  UI/API/WebSocket flows when shared components change; keep development
  configuration separate from the protected distributed profile. Local mode is
  outside assessment, not automatically isolated: the
  existing [Compose configuration](../../../docker-compose.yml) publishes
  several ports, including ingestion, on `0.0.0.0`. Loopback-only publishing may
  be documented as an opt-in safety measure, but do not silently change
  supported local bindings.

## How to use

1. Pick a packet, read it together with its linked requirements and dependencies,
   assign a named owner, and record its actual state in your tracking system.
2. Resolve the specific D-01 through D-09 inputs before choosing an implementation
   or claiming acceptance. Work not dependent on those decisions can proceed in
   parallel; do not treat an unselected default as an approved policy.
3. Implement and test the complete behavior, including negative cases and
   existing integrations. Record the revision, configuration, commands, results,
   skipped tests, and sanitized evidence. An unchecked acceptance criterion is
   unfinished work, not a waiver. Record early milestones separately from full
   packet completion when access-control work is deferred.
4. Close a packet only after its owner reviews the acceptance criteria. The
   [assessment packet](16-assessment-and-approval.md) separately covers every
   applicable requirement, deployment evidence, risk disposition, and sign-off.

**Priorities describe sequencing, not vulnerability severity or approval.** P0
items are public-ingestion exposure blockers; P1 items are early mitigations and
P2 items are scheduled follow-up. Deferred items remain applicable requirements.
Owners must elevate work when data policy, deployment risk, or an approval gate
requires it; priority alone never permits exposure or operation. In particular,
[SCP-04](../01-scope-and-ownership.md#approval-requirements) prohibits publishing
ingestion until authenticated submission, project authorization, TLS,
gateway-bypass prevention, and abuse limits are evidenced or explicitly accepted
as time-bounded exceptions. Stream bounds in TP-07, minimum machine project/run
binding in TP-02, and the trusted broker controls it relies on in TP-08 are
part of that early work even though their remaining scope is scheduled later.

**Shared guardrails:** Keep PostgreSQL authoritative for durable data and MongoDB
limited to transient live buffers. Preserve supported run/shard/retry behavior.
Add reversible SQL migrations for relational schema changes. Protect credentials
and sensitive payloads in logs, examples, tests, and evidence. Use synthetic data
and an authorized non-production environment for destructive, abuse, and load
tests. A gateway or CORS is not a substitute for server-side authorization.

## Packet index

| Packet | Work | Priority | Accountable roles | Primary prerequisites |
| --- | --- | --- | --- | --- |
| [TP-00](00-decisions-and-scope.md) | Deployment decisions and ownership | P0 | Application, platform, identity, data, InfoSec | None |
| [TP-01](01-machine-identity.md) | Authenticate ingestion callers | P0; shared-key limitations remain | Application, identity | D-03 key provision, D-06 TLS |
| [TP-02](02-project-ownership.md) | Bind trusted project/run ownership across events | P0 machine binding; P2 broader scope | Application, data | D-03; TP-01 principal, TP-08 trusted broker path; D-04 for human grants |
| [TP-03](03-human-access-and-authorization.md) | Human identity, REST authorization and mutations | Deferred | Application, identity | D-04; TP-02 ownership; risk disposition |
| [TP-04](04-browser-and-websocket.md) | Browser, CORS, CSRF and live-event access | P1 baseline; access enforcement deferred | Application, identity | D-02; D-04 and TP-02/03 for full enforcement |
| [TP-05](05-artifact-security.md) | Authorized, safe artifact delivery | P1 safe delivery; grants deferred | Application, data | D-05, D-07; TP-02/03 for full authorization |
| [TP-06](06-ingestion-limits.md) | Validation, quotas and backpressure | P0 | Application, operations | D-07; TP-01/02 for identity/project enforcement |
| [TP-07](07-replay-and-event-delivery.md) | Replay, idempotency, stream bounds and DLQ | P1; P0 stream bounds | Application, operations | D-07 for capacity; D-05 for retention; D-03/TP-02 for replay |
| [TP-08](08-dependency-tls-and-credentials.md) | Dependency TLS and scoped workload credentials | P2 internal TLS; P1 external TLS/credentials; P0 trusted broker path | Platform, application, data | D-01, workload D-04; D-06 for TLS; any required exception |
| [TP-09](09-http-and-proxy-hardening.md) | HTTP and reverse-proxy hardening | P1 | Application, platform | D-02, D-06, D-07; coordinate TP-04/05/14 |
| [TP-10](10-retention-and-deletion.md) | Retention and cross-store deletion | P2; policy-dependent | Data, application, operations | D-05, D-08; TP-02/03/07 for completion |
| [TP-11](11-audit-and-monitoring.md) | Security audit, logging and alerts | P2; baseline signals early | Application, operations | D-07/D-08; event sources from TP-01/03/05/06 |
| [TP-12](12-kubernetes-workloads.md) | Helm workload and migration hardening | P1 | Platform, application, release | D-01, D-07; coordinate TP-08/13 |
| [TP-13](13-supply-chain.md) | Immutable releases, scans and provenance | P2 | Release, platform, InfoSec | D-01, D-09 |
| [TP-14](14-network-and-platform.md) | Separate public gRPC and private web; secure platform paths | P0 | Platform, identity | D-01, D-02, D-06; coordinate TP-01/02/08/12 |
| [TP-15](15-recovery-and-response.md) | Rotation, incident response and recovery | P2; P0 reporter revocation | Operations, data, InfoSec | D-03/TP-01 for early revocation; D-05/D-06/D-08/D-09 and TP-07/08/10/11 for full exercises |
| [TP-16](16-assessment-and-approval.md) | Verification, evidence and risk disposition | Ongoing; gates unchanged | InfoSec, owners, risk authority | TP-00 and applicable packet deliverables |

Decision dependencies block completion of the affected behavior, not all
investigation. TP-04/05/09 browser and artifact safeguards can proceed without
waiting for human authentication. TP-06/07 can establish global limits before
machine-specific quotas, and TP-11/15 can establish basic ingestion monitoring
and reporter-revocation procedures before their full scope. Coordinated work
must agree on interfaces and avoid duplicate or contradictory enforcement.

## Traceability to the prior action list

Numbers below are the 28 numbered findings from the preceding conversation, not
InfoSec requirement IDs. Each maps to a primary packet; linked packets share work
where necessary.

| Finding | Packet(s) | Finding | Packet(s) |
| --- | --- | --- | --- |
| 1 | [TP-01](01-machine-identity.md) | 15 | [TP-12](12-kubernetes-workloads.md) |
| 2 | [TP-02](02-project-ownership.md) | 16 | [TP-12](12-kubernetes-workloads.md) |
| 3 | [TP-03](03-human-access-and-authorization.md), [TP-04](04-browser-and-websocket.md), [TP-05](05-artifact-security.md) | 17 | [TP-08](08-dependency-tls-and-credentials.md), [TP-12](12-kubernetes-workloads.md) |
| 4 | [TP-03](03-human-access-and-authorization.md), [TP-11](11-audit-and-monitoring.md) | 18 | [TP-13](13-supply-chain.md) |
| 5 | [TP-04](04-browser-and-websocket.md) | 19 | [TP-12](12-kubernetes-workloads.md) |
| 6 | [TP-04](04-browser-and-websocket.md) | 20 | [TP-12](12-kubernetes-workloads.md) |
| 7 | [TP-05](05-artifact-security.md) | 21 | [TP-08](08-dependency-tls-and-credentials.md), [TP-14](14-network-and-platform.md) |
| 8 | [TP-06](06-ingestion-limits.md) | 22 | [TP-14](14-network-and-platform.md) |
| 9 | [TP-07](07-replay-and-event-delivery.md) | 23 | [TP-12](12-kubernetes-workloads.md) |
| 10 | [TP-06](06-ingestion-limits.md), [TP-07](07-replay-and-event-delivery.md) | 24 | [TP-13](13-supply-chain.md) |
| 11 | [TP-08](08-dependency-tls-and-credentials.md) | 25 | [TP-13](13-supply-chain.md) |
| 12 | [TP-09](09-http-and-proxy-hardening.md), [TP-04](04-browser-and-websocket.md) | 26 | [TP-13](13-supply-chain.md) |
| 13 | [TP-10](10-retention-and-deletion.md) | 27 | [TP-11](11-audit-and-monitoring.md) |
| 14 | [TP-11](11-audit-and-monitoring.md) | 28 | [TP-15](15-recovery-and-response.md) |

Deployment decisions, complete control coverage, and final sign-off are tracked
by [TP-00](00-decisions-and-scope.md) and
[TP-16](16-assessment-and-approval.md), respectively.

## Corrections and qualifications to the preliminary review

- Ingestion already performs some required-field and ID checks in
  [server.go](../../../pkg/server/server.go); it does not yet provide the
  full validation and authorization required by the package.
- The repository has an existing [Helm contract test script](../../../scripts/test-helm-chart.sh),
  and [Helm CI](../../../.github/workflows/helm-publish.yml) runs chart testing
  and kubeconform schema validation. Extend those tests; do not assume there are
  no chart tests or that schema validation is a security scan.
- NATS client connections use a URL and can depend on client-library URL
  behavior; lack of explicit TLS options is not proof that TLS is impossible.
  Verify certificate validation, credentials, and the selected configuration
  rather than asserting all NATS connections are necessarily plaintext.
- A prior recommendation to add `Nats-Msg-Id` alone is not a complete
  idempotency strategy: its broker deduplication window does not replace durable
  replay handling. The existing consumer also acknowledges a message after
  logging a failed DLQ publish; [TP-07](07-replay-and-event-delivery.md)
  explicitly addresses that failure path.
- The later priority discussion placed reporter-to-project binding under P2,
  but SCP-04 makes machine project authorization an exposure gate. TP-02 now
  separates that P0 minimum from broader, later ownership and human access work.
  Similarly, a D-06 decision to defer internal TLS does not itself waive the
  package's TLS requirements; any unmet control needs the applicable approval.
- Tools, SSO products, roles, project storage models, exact limits, PDBs, metrics
  endpoints, and particular nginx ports are implementation choices, not
  automatically mandated solutions. Evidence must demonstrate the required
  outcomes in the selected deployment.
