# Helm Chart Hardening Checklist

**Purpose:** Define the minimum hardening work required before the Observer chart is treated as a public distribution artifact.
**Audience:** Maintainers of `charts/observer/` and downstream infrastructure repositories that consume the chart.
**Last Reviewed:** 2026-09-22

Use this checklist before:

- publishing a new OCI chart version
- recommending the chart for third-party cluster installs
- promoting a new dependency, exposure, or security surface as supported

Checked items below are verified against the current repository state (`charts/observer/` as of 2026-09-20). Partially implemented work remains unchecked until the full acceptance condition is met.

## Current Baseline

The remaining release-gating defects are:

- CI still lacks an automated install/upgrade/rollback smoke test against a real or ephemeral cluster for the recommended public install path (source templates and packaged artifacts are otherwise fully lint/render/schema validated).
- `readOnlyRootFilesystem` is `false` chart-wide; no workload runs with a read-only root filesystem yet.
- No NetworkPolicy resources or documented examples are shipped by the chart.
- Documented upgrade/rollback guidance for stateful dependencies (PostgreSQL/MongoDB/NATS) and PodDisruptionBudget/replica-limit guidance for distributed mode do not exist yet.

Previously tracked defects around reusable passwords, credential-bearing rendered manifests, and missing security contexts have been resolved; see the checked items below.

One-off homelab k3s deployment notes and the separate `observer-mcp` chart task packets are archived under [archive/2026-09-copilot/](archive/2026-09-copilot/) and are not part of this checklist's scope.

## 1. Packaging And Release Hygiene

- [x] HC001 Every file under `charts/observer/templates/` has a single, intentional purpose with no duplicate resources or legacy aliases.
- [x] HC002 `helm lint charts/observer` passes without warnings that would confuse external consumers.
- [x] HC003 `helm template` succeeds for the default chart, `values-aio.yaml`, `values-production.yaml`, and every advertised external dependency mode.
- [x] HC004 `Chart.yaml`, `Chart.lock`, and dependency versions are aligned and intentionally updated together.
- [x] HC005 Chart metadata includes accurate version, appVersion, maintainers, sources, and an icon.
- [x] HC006 The published artifact path is stable and documented, including the OCI registry location, GitHub Pages path, and versioning policy.

## 2. Values Contract And Defaults

- [x] HC010 `values.yaml` defines the full contract for supported chart features; removed ingress, gateway, and cloud-controller resources are no longer implied elsewhere.
- [x] HC011 Example values files do not introduce keys that are missing from `values.yaml`.
- [x] HC012 Default install behavior in `values.yaml`, `charts/observer/README.md`, `DEPLOYMENT.md`, and `docs/architecture/06-helm.md` matches exactly.
- [x] HC013 Mutable defaults such as `latest` tags or `Always` pull policies are not used for public chart defaults unless the chart explicitly documents that policy.
- [x] HC014 Unsupported combinations and legacy distributed connection override paths fail early with template validation, `required`, `fail`, or schema validation instead of silently rendering broken manifests.
- [x] HC015 A `values.schema.json` file validates required fields, enums, and structural assumptions for supported modes.

## 3. External Dependency Contract

- [x] HC020 Every advertised external service mode works without hidden references to embedded services.
- [x] HC021 When `nats.enabled=false`, no workload or init container references the in-cluster NATS service name.
- [x] HC022 When `postgresql.enabled=false`, the chart requires a non-empty PostgreSQL host or documented secret-based equivalent.
- [x] HC023 When `mongodb.enabled=false`, the chart requires a non-empty MongoDB host or documented secret-based equivalent.
- [x] HC024 External dependency examples are rendered in CI and documented as first-class supported paths, not best-effort examples.
- [x] HC025 The chart clearly distinguishes between embedded dependencies, external dependencies, and unsupported legacy distributed connection override patterns.

## 4. Secrets And Credential Handling

- [x] HC030 Public defaults do not ship with reusable passwords such as `password` for PostgreSQL, MongoDB, or app users. All password/credential defaults in `values.yaml` are empty strings; embedded PostgreSQL/MongoDB subcharts auto-generate credentials.
- [x] HC031 Workload manifests use Kubernetes Secret references for credentials instead of raw `value:` entries where feasible.
- [x] HC032 Every `existingSecret` value in `values.yaml` is wired into templates and documented.
- [x] HC033 Connection strings do not expose credentials in rendered manifests when a secret-backed alternative exists. `NATS_URL`, `POSTGRES_DSN`, and `MONGODB_URI` are always injected via `secretKeyRef` in every workload/migration template.
- [x] HC034 README and deployment examples avoid committed live credentials and prefer `runtime.existingSecret` or out-of-band secret creation.
- [x] HC035 Secret-handling behavior is documented for both embedded and external dependency modes.

## 5. Security Posture

- [x] HC040 Non-root execution is the default for distributed workloads unless a component has a documented exception. `values.yaml` `securityContext` sets `runAsNonRoot: true`; the distributed `web` component's documented exception is scoped via `distributed.web.securityContext`.
- [x] HC041 Root execution in AIO mode is explicitly justified and scoped to the minimum required behavior. `aioSecurityContext` includes an inline comment explaining the s6-overlay init requirement and only adds `SETUID`/`SETGID`/`CHOWN`/`DAC_OVERRIDE`.
- [x] HC042 Container security contexts drop unnecessary capabilities and are consistent across services. All `securityContext` blocks drop `ALL` capabilities and set `allowPrivilegeEscalation: false` except the documented AIO exception.
- [ ] HC043 `readOnlyRootFilesystem`, `runAsNonRoot`, pod security context, and service account defaults are compatible with restricted clusters where possible. `readOnlyRootFilesystem` is `false` for every workload; no workload currently runs with a read-only root filesystem.
- [x] HC044 Resource requests and limits are defined for every workload and documented as examples rather than guesses. Every distributed workload, AIO, and embedded dependency defines `resources.limits`/`requests` in `values.yaml`.
- [x] HC045 The chart exposes hooks for pod annotations, service account annotations, and image pull secrets where operators expect them (`podAnnotations`, `serviceAccount.annotations`, `imagePullSecrets` in `values.schema.json`/`values.yaml`). NetworkPolicy hooks are not yet exposed; downstream infrastructure must supply NetworkPolicy separately.

## 6. Reliability And Day-2 Operations

- [x] HC050 Readiness, liveness, and startup behavior reflect the real health surface of each service. `livenessProbe`/`readinessProbe` (and `startupProbe` for ingestion) are defined per distributed workload and AIO.
- [x] HC051 Init containers only wait for dependencies that are actually enabled in the selected configuration.
- [x] HC052 Database migration behavior is single-path and deterministic; the chart does not run competing migration strategies in multiple workloads.
- [ ] HC053 Install, upgrade, and rollback behavior is documented for stateful dependencies and schema changes. The chart README documents the migration hook and Secret update step for `helm upgrade`, but there is no documented rollback story or compatibility matrix for PostgreSQL/MongoDB/NATS schema or data changes.
- [ ] HC054 HPA behavior, replica defaults, and disruption tolerance are documented for distributed mode. HPA (`autoscaling/v2`) is implemented per distributed workload with configurable min/max replicas, but there is no PodDisruptionBudget and no dedicated documentation of replica/disruption behavior beyond the values files.
- [x] HC055 `NOTES.txt` reflects the real access paths and does not assume services or ports that the selected mode does not expose.

## 7. Networking, Exposure, And Downstream Infra

- [x] HC060 The chart explicitly documents that ingress and Gateway API are out of scope for chart-managed resources and must be managed downstream.
- [x] HC061 Cloud-specific resources such as GKE `ManagedCertificate`, `FrontendConfig`, and `BackendConfig` are not rendered by the chart.
- [x] HC062 gRPC exposure expectations are documented at the Service layer for downstream controllers and providers.
- [x] HC063 TLS, redirect, and certificate guidance lives in downstream infrastructure docs rather than chart values or templates.
- [x] HC064 Service names, ports, and protocol expectations are stable and documented for downstream infrastructure automation.

## 8. Documentation And Supportability

- [x] HC070 `charts/observer/README.md` explains the current supported defaults, known limitations, and recommended install paths.
- [x] HC071 `DEPLOYMENT.md` and `docs/architecture/06-helm.md` are updated whenever install defaults or support surfaces change.
- [x] HC072 Every values file shipped in the repo is still supported, still tested, and still documented.
- [x] HC073 Breaking changes and deprecations are called out in `CHANGELOG.md` or release notes.
- [x] HC074 Public examples prefer safe patterns such as secret references, immutable image tags, and downstream-managed exposure configuration.

## 9. CI Gates For Public Publication

- [x] HC080 CI runs `helm lint` for the chart on every change that touches `charts/observer/**`.
- [x] HC081 CI renders a matrix that includes default, AIO, production, and advertised external dependency modes (`.github/workflows/helm-publish.yml`).
- [x] HC082 Rendered manifests are validated with a Kubernetes schema tool such as `kubeconform` or `kubeval`.
- [ ] HC083 At least one install smoke test runs against a real or ephemeral cluster for the recommended public install path. CI currently validates rendering, schema, and packaged-artifact contents (`scripts/test-helm-chart.sh`) but does not provision a cluster (e.g. `kind`/`k3d`) and run `helm install`/`helm upgrade`/`helm test`.
- [x] HC084 The chart is packaged and published only after the validation matrix passes.

## Release Gate

Treat the chart as public-ready only when:

- no blocker from the current baseline remains open
- every checklist item in sections 1 through 4 is complete
- section 9 is automated in CI rather than checked manually
- documentation and defaults describe the same operator experience

Until then, downstream infrastructure repositories should assume they are integrating an internal chart that may still require wrappers, patches, or pinned local knowledge.
