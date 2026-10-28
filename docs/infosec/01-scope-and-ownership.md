# 01 - Scope and Ownership

**Status:** Target-state specification; not assessed.

## System purpose and review boundary

Observer receives test lifecycle events and attachments from CI reporters,
processes events asynchronously, and provides persisted results and live updates
to authorized users.

In scope:

- Distributed ingestion, processor, API, and web workloads deployed with Helm.
- Public gRPC edge, private/corporate user access, and all routes to underlying APIs.
- NATS JetStream, PostgreSQL, MongoDB live buffers, and artifact storage.
- Kubernetes identities, network policy, secrets, migration job, persistent volumes,
  backups, logging, and release artifacts.
- Reporter credentials and the CI-to-Observer trust boundary.

Out of scope for this submission: AIO/Compose installations and security assurance
of the applications under test. Reporter dependencies, CI provider security, identity
provider, cluster, and managed services are external dependencies whose interfaces
and inherited controls remain in scope.

## Target assumptions

| Topic | Target assumption |
| --- | --- |
| Hosting | Dedicated Observer namespace in a supported Kubernetes environment |
| Ingestion | External CI runners can reach a dedicated public gRPC hostname over TLS on 443 |
| Human access | Private/corporate connectivity or an identity-aware gateway with organizational SSO |
| Data services | Private network endpoints; no direct internet exposure |
| Artifacts | Private S3-compatible storage, with narrowly scoped workload identities |
| Persistence | PostgreSQL authoritative; MongoDB transient live buffers only |
| Isolation | Explicit project-scoped authorization; Kubernetes namespace isolation alone is insufficient |

These are design assumptions, not selected infrastructure products or existing controls.

## Required deployment decisions

All entries are **Pending** until the accountable owner records a decision.

| ID | Decision required | Accountable role |
| --- | --- | --- |
| D-01 | Cluster/environment, region, namespace, exact chart and image digest, embedded versus managed dependencies | Platform owner |
| D-02 | Gateway/controller, hostnames, public ingress necessity, CI source restrictions | Platform owner |
| D-03 | Reporter identity method: short-lived tokens or mTLS; issuer, audience, scopes, rotation/revocation | Identity and application owners |
| D-04 | SSO integration, application/project roles, service identities, identity propagation | Identity and application owners |
| D-05 | Data classification, permitted test content, residency, retention/deletion periods, legal holds | Data owner with InfoSec |
| D-06 | TLS termination/re-encryption, internal mTLS where required, certificate ownership | Platform owner |
| D-07 | Payload/attachment limits, concurrency, quotas, backlog limits, alert thresholds | Application and operations owners |
| D-08 | Backup scope, RPO/RTO, recovery testing cadence, audit retention | Operations and data owners |
| D-09 | Patch SLAs, verification cadence, exception approval and expiry policy | InfoSec and operations owners |

## Responsibility boundaries

| Role | Responsibility |
| --- | --- |
| Application owner | Validation, project authorization, safe artifact handling, event integrity, security events |
| Platform owner | Gateway, DNS, TLS, NetworkPolicies, RBAC, pod admission, workload identity, cluster controls |
| Identity owner | Human and machine identity policy, issuance, federation, revocation |
| Data owner | Classification, retention, deletion, acceptable content, residency |
| Operations owner | Monitoring, response, backups, restore, upgrades, capacity |
| Release owner | Trusted builds, provenance, dependency inventory, image/chart distribution |
| InfoSec reviewer | Control assessment, residual-risk review, approval conditions |
| Business risk owner | Explicit acceptance of residual risks; cannot silently waive requirements |

Assign named individuals and escalation contacts in the assessment record.
Operator-owned resources must be versioned and reviewed with the deployment bundle;
the Observer chart does not currently render Ingress, Gateway API, Certificate,
or NetworkPolicy resources.

## Approval requirements

- **SCP-01:** Freeze the assessed deployment identity and sanitized configuration.
- **SCP-02:** Assign an accountable owner and evidence to every applicable requirement.
- **SCP-03:** Resolve D-01 through D-09 before final approval.
- **SCP-04:** Do not publish ingestion until authenticated submission, project
  authorization, TLS, gateway-bypass prevention, and abuse limits are evidenced or
  explicitly accepted as time-bounded exceptions.
- **SCP-05:** Reassess after material changes to exposure, identity, data classes,
  tenancy, dependencies, or storage topology.
