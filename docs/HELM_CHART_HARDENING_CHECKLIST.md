# Helm Chart Hardening Checklist

**Purpose:** Define the minimum hardening work required before the Observer chart is treated as a public distribution artifact.
**Audience:** Maintainers of `charts/observer/` and downstream infrastructure repositories that consume the chart.
**Last Reviewed:** 2026-05-23

Use this checklist before:

- publishing a new OCI chart version
- recommending the chart for third-party cluster installs
- promoting a new networking, dependency, or security surface as supported

Checked items below are verified against the current repository state. Partially implemented work remains unchecked until the full acceptance condition is met.

## Current Baseline

The remaining release-gating defects are:

- Secret-backed runtime wiring exists, but public defaults, generated secrets, and secret setup examples are not fully hardened yet.

## 1. Packaging And Release Hygiene

- [x] HC001 Every file under `charts/observer/templates/` has a single, intentional purpose with no duplicate resources or legacy aliases.
- [x] HC002 `helm lint charts/observer` passes without warnings that would confuse external consumers.
- [x] HC003 `helm template` succeeds for the default chart, `values-aio.yaml`, `values-production.yaml`, and every advertised external dependency mode.
- [x] HC004 `Chart.yaml`, `Chart.lock`, and dependency versions are aligned and intentionally updated together.
- [ ] HC005 Chart metadata includes accurate version, appVersion, maintainers, sources, and an icon.
- [ ] HC006 The published artifact path is stable and documented, including OCI registry location and versioning policy.

## 2. Values Contract And Defaults

- [ ] HC010 `values.yaml` defines a complete contract for every supported feature tree, including optional Gateway and cloud-specific settings.
- [x] HC011 Example values files do not introduce keys that are missing from `values.yaml`.
- [x] HC012 Default install behavior in `values.yaml`, `charts/observer/README.md`, `DEPLOYMENT.md`, and `docs/architecture/06-helm.md` matches exactly.
- [ ] HC013 Mutable defaults such as `latest` tags or `Always` pull policies are not used for public chart defaults unless the chart explicitly documents that policy.
- [ ] HC014 Unsupported combinations fail early with template validation, `required`, `fail`, or schema validation instead of silently rendering broken manifests.
- [x] HC015 A `values.schema.json` file validates required fields, enums, and structural assumptions for supported modes.

## 3. External Dependency Contract

- [x] HC020 Every advertised external service mode works without hidden references to embedded services.
- [x] HC021 When `nats.enabled=false`, no workload or init container references the in-cluster NATS service name.
- [x] HC022 When `postgresql.enabled=false`, the chart requires a non-empty PostgreSQL host or documented secret-based equivalent.
- [x] HC023 When `mongodb.enabled=false`, the chart requires a non-empty MongoDB host or documented secret-based equivalent.
- [x] HC024 External dependency examples are rendered in CI and documented as first-class supported paths, not best-effort examples.
- [ ] HC025 The chart clearly distinguishes between embedded dependencies, external dependencies, and unsupported hybrid combinations.

## 4. Secrets And Credential Handling

- [ ] HC030 Public defaults do not ship with reusable passwords such as `password` for PostgreSQL, MongoDB, or app users.
- [x] HC031 Workload manifests use Kubernetes Secret references for credentials instead of raw `value:` entries where feasible.
- [x] HC032 Every `existingSecret` or `existingSecret*Key` value in `values.yaml` is wired into templates and documented.
- [ ] HC033 Connection strings do not expose credentials in rendered manifests when a secret-backed alternative exists.
- [ ] HC034 README examples do not encourage operators to place live credentials directly into committed values files.
- [x] HC035 Secret-handling behavior is documented for both embedded and external dependency modes.

## 5. Security Posture

- [ ] HC040 Non-root execution is the default for distributed workloads unless a component has a documented exception.
- [ ] HC041 Root execution in AIO mode is explicitly justified and scoped to the minimum required behavior.
- [ ] HC042 Container security contexts drop unnecessary capabilities and are consistent across services.
- [ ] HC043 `readOnlyRootFilesystem`, `runAsNonRoot`, pod security context, and service account defaults are compatible with restricted clusters where possible.
- [ ] HC044 Resource requests and limits are defined for every workload and documented as examples rather than guesses.
- [ ] HC045 The chart exposes hooks for network policies, pod annotations, service account annotations, and image pull secrets where operators expect them.

## 6. Reliability And Day-2 Operations

- [ ] HC050 Readiness, liveness, and startup behavior reflect the real health surface of each service.
- [x] HC051 Init containers only wait for dependencies that are actually enabled in the selected configuration.
- [x] HC052 Database migration behavior is single-path and deterministic; the chart does not run competing migration strategies in multiple workloads.
- [ ] HC053 Install, upgrade, and rollback behavior is documented for stateful dependencies and schema changes.
- [ ] HC054 HPA behavior, replica defaults, and disruption tolerance are documented for distributed mode.
- [ ] HC055 `NOTES.txt` reflects the real access paths and does not assume services or ports that the selected mode does not expose.

## 7. Networking, Ingress, And Gateway Support

- [ ] HC060 Ingress support and Gateway API support are treated as separate supported surfaces with distinct documentation and CI coverage.
- [ ] HC061 Cloud-specific resources such as GKE `ManagedCertificate`, `FrontendConfig`, and `BackendConfig` are rendered only when their controller assumptions are valid.
- [ ] HC062 gRPC exposure is validated for every supported controller or provider combination.
- [ ] HC063 TLS, redirect, and certificate examples render without duplicate resources or conflicting annotations.
- [ ] HC064 Service names, ports, and protocol expectations are stable and documented for downstream infrastructure automation.

## 8. Documentation And Supportability

- [x] HC070 `charts/observer/README.md` explains the current supported defaults, known limitations, and recommended install paths.
- [x] HC071 `DEPLOYMENT.md` and `docs/architecture/06-helm.md` are updated whenever install defaults or support surfaces change.
- [x] HC072 Every values file shipped in the repo is still supported, still tested, and still documented.
- [ ] HC073 Breaking changes and deprecations are called out in `CHANGELOG.md` or release notes.
- [ ] HC074 Public examples prefer safe patterns such as secret references, immutable image tags, and explicit ingress configuration.

## 9. CI Gates For Public Publication

- [x] HC080 CI runs `helm lint` for the chart on every change that touches `charts/observer/**`.
- [x] HC081 CI renders a matrix that includes default, AIO, production, external NATS, external PostgreSQL, and any supported Gateway or managed-certificate paths.
- [x] HC082 Rendered manifests are validated with a Kubernetes schema tool such as `kubeconform` or `kubeval`.
- [ ] HC083 At least one install smoke test runs against a real or ephemeral cluster for the recommended public install path.
- [x] HC084 The chart is packaged and published only after the validation matrix passes.

## Release Gate

Treat the chart as public-ready only when:

- no blocker from the current baseline remains open
- every checklist item in sections 1 through 4 is complete
- section 9 is automated in CI rather than checked manually
- documentation and defaults describe the same operator experience

Until then, downstream infrastructure repositories should assume they are integrating an internal chart that may still require wrappers, patches, or pinned local knowledge.
