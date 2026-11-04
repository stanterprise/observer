# Observer InfoSec Review Package

**Status:** Proposed target-state requirements; implementation assessment pending.
**Prepared:** 2026-10-28.
**Scope:** Distributed Observer deployed through Helm to a Kubernetes cluster.

This package specifies the security capabilities Observer and its hosting platform
are expected to provide. It is not a security assessment, implementation attestation,
penetration-test report, or approval to deploy. Diagrams describe the intended
architecture, including controls that may require implementation.

In this package, **must** identifies an approval requirement, **should** identifies
a recommendation whose omission needs justification, and **may** identifies an
option. A requirement is not satisfied until deployment-specific evidence is
reviewed. Every control begins as **Not assessed**.

## Reading order and deliverables

| Document | Purpose |
| --- | --- |
| [01 - Scope and ownership](01-scope-and-ownership.md) | Review boundary, assumptions, decisions, responsibilities, and approval conditions |
| [02 - C4 architecture](02-c4-architecture.md) | System Context and Container diagrams |
| [03 - Deployment and network](03-deployment-and-network.md) | Kubernetes topology, trust boundaries, TLS, and network flow matrix |
| [04 - Identity and ingestion](04-identity-and-ingestion.md) | Access matrix, machine identity, public gRPC, and abuse prevention |
| [05 - Data handling](05-data-handling.md) | Data inventory, security data flows, classification, retention, and deletion |
| [06 - Kubernetes baseline](06-kubernetes-baseline.md) | Helm and platform hardening requirements |
| [07 - Threat model and risk register](07-threat-model-and-risk-register.md) | Design threats, control mapping, unresolved decisions, and risk acceptance |
| [08 - Supply chain and verification](08-supply-chain-and-verification.md) | Build evidence, SBOM, scans, and security verification plan |
| [09 - Operations and response](09-operations-and-response.md) | Audit, monitoring, incident response, maintenance, and recovery |
| [10 - Assessment evidence](10-assessment-evidence.md) | Submission manifest, traceability, evidence records, and sign-off template |

The [remediation task packets](task-packets/README.md) turn the preliminary
repository gap review into proposed, dependency-aware work items. They do not
change the assessment status or replace deployment-specific evidence.

## Architecture anchors

These sources establish terminology and deployment contracts, not security assurance:

- [Helm chart contract](../../charts/observer/README.md): chart-managed workloads and
  Services; operator-managed exposure, TLS, secrets integration, and NetworkPolicies.
- [Distributed values](../../charts/observer/values.yaml): configuration starting point,
  not an approved production baseline.
- [Components](../architecture/01-components.md) and
  [data flow](../architecture/02-dataflow.md): service and event flow boundaries.
- [Data authority](../architecture/07-database-schema.md): PostgreSQL stores durable
  records; MongoDB is limited to transient `live_step_buffers`.

The existing architecture documentation describes authentication/OIDC as planned.
This package therefore makes no claim that application authentication, project
authorization, or other proposed controls already exist. A gateway is not a
substitute for resource-level authorization unless its enforcement is demonstrated.

## How to use this package

1. Select the deployment configuration and resolve the decisions in document 01.
2. Record named owners, approved policies, exact versions, and measurable limits.
3. Assess implementation against the requirement IDs throughout these documents.
4. Attach sanitized evidence using document 10; record gaps in document 07.
5. Obtain explicit InfoSec approval or time-bounded risk acceptance before exposure.

Do not attach credentials, raw production artifacts, personal data, unredacted
Helm Secret manifests, or sensitive kubeconfig files.
