# Helm Chart Issue Resolution Task Packet

**Status:** Proposed
**Source:** [Helm chart deployment retrospective](HELM_CHART_DEPLOYMENT_RETROSPECTIVE.md)
**Scope:** Observer chart, observer-mcp chart, and the declarative deployment bundle that combines them
**Target outcome:** The application charts install and operate predictably without manual Deployment patches or release-specific tribal knowledge. DNS, TLS, and ingress are tracked separately in [HELM_PLATFORM_EXPOSURE_TASK_PACKET.md](HELM_PLATFORM_EXPOSURE_TASK_PACKET.md).

## 1. Problem Statement

The current application installation is operational, but it is not reproducible from the published Helm charts alone. It depends on manual Secrets, Deployment mutations, an external ARM64 MongoDB workload, and image-tag overrides. Those dependencies are easy to lose during upgrade and are not represented by one tested application release contract.

The existing [Helm chart remediation plan](HELM_CHART_REMEDIATION_PLAN.md) covers Observer chart public-readiness issues. This packet extends that work to the observed two-chart deployment and its platform dependencies. It does not replace the existing hardening checklist or publish-readiness rubric.

## 2. Definition Of Done

The target is a supported application release, not a promise that every Kubernetes platform or dependency combination is automatic. The application charts own workloads, Services, ports, probes, and application runtime configuration. Environment-specific exposure, DNS, certificates, ingress policy, and private-PKI integration are out of scope for this packet.

An operator should be able to:

1. Install the documented prerequisites and provide credentials through an external Secret mechanism.
2. Run one pinned deployment command or one documented bundle command.
3. Receive all application workloads, migrations, Services, and required application configuration from the charts.
4. Upgrade or roll back without reapplying manual patches or remembering hidden resource ownership.
5. Run the published smoke test and verify web/API, gRPC ingestion, S3 attachments, and MCP SSE behavior.

The application may require platform prerequisites such as Kubernetes, storage, and a Secret provider. It must fail early and explain those prerequisites; it must not silently depend on them.

## 3. Supported Golden Path

The first release target should be the environment represented by the retrospective:

| Concern        | Supported choice                                                                                                        |
| -------------- | ----------------------------------------------------------------------------------------------------------------------- |
| Kubernetes     | A currently supported Kubernetes minor                                                                                  |
| Architecture   | `linux/amd64` and `linux/arm64` for every required image                                                                |
| Observer       | Distributed mode, OCI chart, immutable image reference                                                                  |
| observer-mcp   | One replica, OCI chart, immutable image reference                                                                       |
| Databases      | PostgreSQL and MongoDB supplied as explicit external or bundled modes; the ARM64 profile uses a validated MongoDB image |
| Messaging      | NATS JetStream with persistent storage                                                                                  |
| Object storage | S3-compatible storage through a Secret and optional custom CA                                                           |
| Secrets        | ExternalSecret/SOPS-compatible contract; no credential literals in values, `--set`, or chart release metadata           |

The chart repositories must state which parts of this table are chart-owned and which are operator-owned. A chart by itself is supported only for the documented chart-owned surface.

## 4. Prioritized Work Items

### P0. Remove installation and upgrade blockers

#### P0-1. Define the release contract and ownership boundary

**Owner:** Chart maintainers and platform owner  
**Repositories:** Observer, observer-mcp, deployment bundle

Document one support matrix covering embedded versus external PostgreSQL, MongoDB, NATS, S3, architecture, and MCP transport. Mark every application resource as chart-owned or operator-owned. Pin chart, app, and image versions independently until release metadata is made consistent.

**Acceptance criteria:**

- A clean-cluster install guide contains one recommended path and no contradictory provider-specific path.
- Every resource in the retrospective inventory has an owner and lifecycle.
- Chart README, values schema, release notes, and bundle values agree on supported defaults.
- Unsupported combinations fail during render with an actionable message.

#### P0-2. Make image selection release-safe

**Owner:** Chart and release maintainers  
**Repositories:** Observer, observer-mcp

Derive the default image tag from the packaged application version using the repository's actual tag convention, or require an explicit immutable tag/digest. Add release tests for `v`-prefixed tags and verify the packaged chart, image, and runtime-reported version are aligned.

**Acceptance criteria:**

- A chart-only install of a published version selects an existing image without `--set image.tag`.
- Mutable `latest` is not the default for a released chart.
- CI verifies `linux/amd64` and `linux/arm64` manifests for every required image.
- A digest-pinned install is supported and documented.

#### P0-3. Make runtime and storage Secrets first-class

**Owner:** Chart maintainers and security owner  
**Repositories:** Observer, observer-mcp

Support existing Secret references for PostgreSQL, MongoDB, NATS, S3, custom CA material, and MCP authentication. Add `extraEnv`, `extraEnvFrom`, `extraVolumes`, and `extraVolumeMounts` only where they are needed as stable chart extension points. Remove reusable password defaults and prevent credentials from being rendered into Helm values or release metadata.

**Acceptance criteria:**

- Migration hooks and all workloads consume the same validated runtime Secret contract.
- Missing Secret names or keys fail before a hook Job or workload starts, without printing values.
- S3 configuration and the CA mount are rendered declaratively for both API and processor.
- Secret rotation has a documented rollout mechanism, such as checksum annotations or a Secret reloader.
- Rendered manifests contain no reusable passwords or literal access keys.

#### P0-4. Fix external dependency behavior

**Owner:** Application and chart maintainers  
**Repositories:** Observer, observer-mcp

Complete the external PostgreSQL, MongoDB, and NATS contracts. Remove hidden references to embedded service names. Make MongoDB genuinely optional in observer-mcp, or explicitly require and validate its Secret key. Validate the supported ARM64 MongoDB path and document backup/PVC ownership.

**Acceptance criteria:**

- Every advertised external mode renders without embedded DNS references.
- Missing external endpoints or keys fail at render time.
- observer-mcp starts with PostgreSQL only when MongoDB is disabled, if that mode is advertised.
- The selected dependency images have tested multi-architecture manifests.

#### P0-5. Add a real install and upgrade smoke test

**Owner:** CI/platform owner

Test the packaged OCI artifacts, not only source templates. The test should install the golden path into an ephemeral or dedicated cluster, wait for migrations and readiness, exercise the endpoints, upgrade one version, and verify rollback evidence.

**Acceptance criteria:**

- CI covers lint, schema validation, render matrix, package inspection, install, upgrade, and rollback.
- The smoke test verifies exact image references, Secret references, Services, probes, and migration completion.
- The test performs an S3 operation and an MCP `initialize`, `tools/list`, PostgreSQL-backed query, and Mongo-backed query when enabled.
- A failed smoke test blocks OCI publication.

### P1. Make day-two operations predictable

#### P1-1. Make migrations and compatibility explicit

Document and test the migration hook strategy, PostgreSQL backup gate, rollback limits, and Observer/observer-mcp schema compatibility matrix. Ensure an application rollback cannot be mistaken for a database rollback.

#### P1-2. Harden security and tenancy defaults

Default distributed workloads to restricted-compatible security contexts, provide MCP authentication through a Secret, narrow wildcard CORS where possible, and add NetworkPolicy support or maintained examples.

#### P1-3. Improve runtime operations

Validate probes, resource defaults, replica limitations, NATS persistence, local-path recovery, Secret-triggered rollouts, and MCP's one-replica/session limitation. Make `NOTES.txt` and upgrade documentation reflect the actual endpoints.

#### P1-4. Add backup and restore gates

Provide tested procedures for PostgreSQL, MongoDB, NATS JetStream, and S3 metadata/object data. Run a restore test before promoting a release that changes stateful dependencies.

### P2. Reduce platform maintenance risk

- Publish dependency compatibility and architecture matrices for every release.
- Move the working application deployment into a small release bundle with encrypted Secret inputs and environment overlays.
- Add release evidence to the changelog: packaged chart, image digests, smoke-test result, migration notes, and known limitations.

## 5. Required Deliverables

1. **Chart changes:** values schema, Secret references, S3/CA extension points, dependency validation, image defaults, security defaults, and accurate notes.
2. **observer-mcp chart changes:** independent datastore Secret references, optional MongoDB behavior, image/version alignment, auth Secret, and stable Service/protocol documentation. Ingress/TLS examples belong in the deployment bundle unless explicitly supported by the chart.
3. **Deployment bundle:** pinned values, external dependency manifests, Secret-provider contracts, and application overlays for supported platform profiles.
4. **CI:** render and schema matrix plus packaged-artifact install/upgrade/smoke tests on both architectures where practical.
5. **Documentation:** support matrix, prerequisites, install, upgrade, rollback, backup/restore, Secret rotation, and troubleshooting.

## 6. Verification Matrix

| Gate         | Evidence required                                                           |
| ------------ | --------------------------------------------------------------------------- |
| Render       | All supported profiles render; unsupported combinations fail clearly        |
| Security     | No literal reusable credentials; Secret keys and RBAC are validated         |
| Architecture | Required images expose tested `amd64` and `arm64` manifests                 |
| Install      | Clean-cluster install from packaged OCI artifacts reaches Ready             |
| Runtime      | Web/API, gRPC, S3, and MCP workflows pass                                   |
| Upgrade      | Version N to N+1 preserves PVCs, Services, Secrets, and data access         |
| Rollback     | Application rollback procedure is tested and database limits are documented |
| Operations   | Probes, logs, backups, Secret rotation, and recovery procedures are tested  |

## 7. Open Decisions For Review

These decisions should be made before implementation begins:

1. Is bundled PostgreSQL/MongoDB production-supported, or only an evaluation profile?
2. Should the application release bundle live in this repository or a separate infrastructure repository?
3. Is MCP MongoDB access required, optional, or split into a separate deployment profile?
4. Which Kubernetes versions define the first supported application platform matrix?

## 8. Relationship To Existing Readiness Work

Use this packet together with:

- [HELM_CHART_REMEDIATION_PLAN.md](HELM_CHART_REMEDIATION_PLAN.md) for the existing Observer chart workstreams.
- [HELM_CHART_HARDENING_CHECKLIST.md](HELM_CHART_HARDENING_CHECKLIST.md) as the release gate.
- [HELM_CHART_PUBLISH_READINESS_RUBRIC.md](HELM_CHART_PUBLISH_READINESS_RUBRIC.md) to score direct-consumption readiness.

The retrospective's current decision is **block direct consumption until P0-1 through P0-5 are complete**. Until then, a downstream deployment may use a wrapper, but that wrapper must be treated as temporary compatibility infrastructure rather than the application installation contract. See [HELM_PLATFORM_EXPOSURE_TASK_PACKET.md](HELM_PLATFORM_EXPOSURE_TASK_PACKET.md) for the separate platform configuration work.
