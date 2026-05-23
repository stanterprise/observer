# Helm Chart Remediation Plan

**Purpose:** Convert the current Helm chart readiness findings into an implementation plan that can move `charts/observer/` from internal-use quality to public-consumption quality.
**Audience:** Maintainers of the Observer chart and downstream infrastructure repositories that depend on it.
**Last Reviewed:** 2026-05-22

This plan is the execution companion to:

- [HELM_CHART_HARDENING_CHECKLIST.md](HELM_CHART_HARDENING_CHECKLIST.md)
- [HELM_CHART_PUBLISH_READINESS_RUBRIC.md](HELM_CHART_PUBLISH_READINESS_RUBRIC.md)

## Problem Statement

The chart currently renders for some primary modes, but it is not yet a stable public distribution artifact. The current release blockers fall into five groups:

1. Duplicate template files produce colliding GKE resources.
2. External dependency modes are advertised more broadly than they are actually implemented.
3. Secrets and credentials are handled as plain values instead of first-class secret references.
4. Gateway and cloud-specific networking support do not share a stable values contract.
5. Documentation, defaults, and published support boundaries do not describe the same operator experience.

Until these are fixed, downstream infrastructure repositories should assume they are consuming an internal chart rather than a public product.

## Objectives

### Primary Objectives

- make the chart safe to publish and consume without template patching
- narrow the supported public contract to behavior the chart can actually guarantee
- replace implicit team knowledge with validated defaults, schema, and documentation
- add CI gates so public readiness stays true after future changes

### Non-Goals For The First Hardening Pass

- redesigning the application architecture or service topology
- making every current chart surface public-ready in one release
- supporting every ingress controller or cloud provider immediately
- solving unrelated runtime defects outside the chart contract

## Target End State

At the end of this plan, the chart should meet these conditions:

- the recommended install path renders, installs, upgrades, and rolls back predictably
- every documented external dependency path works without embedded-service assumptions
- secrets are sourced from Kubernetes Secrets or generated subchart secrets, not raw committed values
- unsupported or experimental paths are either removed, explicitly gated, or clearly marked as non-public
- README, deployment docs, architecture docs, values files, and OCI metadata all describe the same supported behavior
- CI enforces lint, schema validation, render coverage, Kubernetes manifest validation, and at least one cluster smoke test

## Recommended Public Support Boundary

Define the public support boundary before deeper implementation work. Without this decision, the chart will continue to over-advertise features.

### Recommended Public Baseline For The Next Release

- **Supported:** distributed mode with embedded or external dependencies, one primary ingress path, OCI publication
- **Supported with limitations:** AIO mode for development or evaluation, clearly marked as not production-grade
- **Experimental or internal-only until hardened:** Gateway API path, GKE managed certificate path, any cloud-specific route not covered by CI

### Required Decision

Before phase 1 begins, maintainers should explicitly choose:

- the recommended public install mode
- the one networking pattern to treat as the primary public path
- which optional surfaces are supported, experimental, or removed from docs

## Workstreams And Sequencing

The work should be executed in the order below. Earlier workstreams reduce ambiguity for later ones.

## Phase 0. Support Boundary And Inventory

**Goal:** Stop further drift by defining what the chart is and is not promising publicly.

### Tasks

1. Inventory all chart surfaces:
   - default mode
   - AIO preset
   - production preset
   - external PostgreSQL
   - external MongoDB
   - external NATS
   - ingress
   - GKE managed certificates
   - Gateway API
2. Mark each surface as one of:
   - supported
   - experimental
   - internal-only
   - remove
3. Record the chosen public support boundary in chart docs.
4. Freeze introduction of new chart features until phases 1 through 4 are complete.

### Deliverables

- documented support matrix in chart docs
- single recommended installation path
- explicit list of deferred or experimental surfaces

### Exit Criteria

- maintainers agree which surfaces are public
- docs stop implying support for paths that are not validated

## Phase 1. Template Hygiene And Values Contract

**Goal:** Remove structural defects and make the chart contract internally consistent.

### Tasks

1. Remove duplicate template files that emit the same resource names.
   - consolidate `frontend-config.yaml` and `frontendconfig.yaml`
   - consolidate `managed-certificate.yaml` and `managedcertificate.yaml`
2. Review all template filenames for legacy duplicates or renamed leftovers.
3. Normalize values keys so `values.yaml` contains the canonical schema for all supported features.
   - ensure Gateway keys in `values.yaml` match what templates dereference
   - ensure example values files only use keys that exist in the canonical contract
4. Add `values.schema.json` with at least:
   - allowed `mode` values
   - required subtrees for supported Gateway or ingress modes
   - type checks for hosts, ports, booleans, lists, and maps
   - validation for mutually exclusive modes where practical
5. Add template-level `required` or `fail` checks for combinations that schema alone cannot enforce.

### Acceptance Criteria

- no duplicate resources are emitted by any supported values file
- `helm template` succeeds or fails intentionally with actionable errors
- optional trees do not nil-pointer when enabled
- example values files remain aligned with canonical defaults

### Validation

- `helm lint ./charts/observer`
- `helm template observer ./charts/observer`
- `helm template observer ./charts/observer -f ./charts/observer/values-aio.yaml`
- `helm template observer ./charts/observer -f ./charts/observer/values-production.yaml`
- render every supported optional mode explicitly

## Phase 2. External Dependency Contract

**Goal:** Make every advertised external dependency path behave correctly.

### Tasks

1. Fix external NATS mode.
   - make the ingestion init container conditional on embedded NATS being enabled
   - ensure no workload references the in-cluster NATS service when `nats.enabled=false`
   - add validation that `externalNats.url` is required when embedded NATS is disabled
2. Harden external PostgreSQL mode.
   - require host or secret-backed equivalent when `postgresql.enabled=false`
   - ensure migration and runtime paths share the same source of truth for DSN construction
3. Harden external MongoDB mode.
   - require host or secret-backed equivalent when `mongodb.enabled=false`
   - ensure templates do not assume the embedded service DNS name
4. Decide whether hybrid modes are supported.
   - if supported, test them
   - if not supported, block them early with validation
5. Add explicit examples for supported external dependency combinations.

### Acceptance Criteria

- external dependency modes no longer rely on embedded service names, init steps, or assumptions
- disabled embedded dependencies cause clear validation failures if required external config is missing
- public examples cover every supported external mode

### Validation

- render matrix for external NATS, PostgreSQL, and MongoDB paths
- install smoke test for at least one external dependency scenario
- negative tests that confirm missing required external config fails early

## Phase 3. Secrets And Credential Handling

**Goal:** Replace internal-value credential patterns with public-safe secret handling.

### Tasks

1. Remove reusable password defaults from public-facing values.
   - do not ship `password` as a public default for PostgreSQL, MongoDB, or app credentials
2. Decide the canonical secret model for the chart.
   - existing secret references supplied by the operator
   - generated secrets for embedded dependencies via subcharts
   - clear precedence rules between explicit values and secret refs
3. Wire every advertised `existingSecret` field into templates.
4. Refactor workload env wiring to use `valueFrom.secretKeyRef` where feasible.
5. Avoid rendering DSNs with inline secrets when a secret-backed path exists.
6. Split safe example values from local-dev-only examples if necessary.
7. Document secret setup for:
   - embedded dependencies
   - external dependencies
   - migration jobs or hooks

### Acceptance Criteria

- public values files no longer encourage committed live secrets
- workload manifests no longer expose passwords as plain `value:` entries in the primary public paths
- secret-related keys in values are fully implemented and documented

### Validation

- inspect rendered manifests for raw password values
- run manifest search checks in CI for obvious leaked defaults
- update docs examples to use secret references or placeholders only

## Phase 4. Migration, Probes, And Runtime Behavior

**Goal:** Make day-2 behavior predictable and operationally sane.

### Tasks

1. Choose one migration strategy as the public default.
   - dedicated migration job
   - init container on one workload
   - another deterministic single-path strategy
2. Remove duplicated migration execution paths if both job and workload init containers can run the same work.
3. Review probes for each service to ensure they reflect real health surfaces.
4. Ensure startup logic only waits for enabled dependencies.
5. Review `NOTES.txt` so it matches the supported access patterns and exposed services.
6. Review security defaults for distributed mode and document why AIO requires elevated behavior.

### Acceptance Criteria

- installs and upgrades do not rely on competing migration mechanisms
- probes and init logic match the selected deployment topology
- post-install notes are accurate for supported modes

### Validation

- upgrade simulation across at least one prior chart version
- install smoke tests for the recommended public mode
- template inspection for dependency-conditional init logic

## Phase 5. Networking And Exposure Hardening

**Goal:** Turn networking from a loose set of options into a clearly supported product surface.

### Tasks

1. Pick one primary public exposure pattern.
   - for example: ingress with a documented controller
2. Separate cloud-specific resources from generic chart behavior.
3. Gate GKE resources so they render only when their controller assumptions are valid.
4. Decide whether Gateway API is:
   - in scope for the public release
   - experimental and hidden from primary docs
   - removed until the contract is stable
5. Validate gRPC exposure explicitly for the chosen public path.
6. Ensure certificate, redirect, and backend configuration examples do not collide or emit duplicate resources.

### Acceptance Criteria

- one public networking path is clearly documented and tested
- cloud-specific paths are either validated or clearly marked experimental
- GKE managed certificate and Gateway paths cannot silently render broken manifests

### Validation

- render tests for ingress and any retained cloud-specific modes
- smoke test for the recommended public network entrypoint
- negative tests for unsupported controller combinations

## Phase 6. Documentation And Example Alignment

**Goal:** Make the docs tell the truth about the chart.

### Tasks

1. Align `charts/observer/README.md` with actual defaults and supported modes.
2. Update `DEPLOYMENT.md` and `docs/architecture/06-helm.md` to match the chosen public support boundary.
3. Review every shipped values file.
   - keep only supported presets
   - rename local or experimental presets if needed
   - document intended use for each file
4. Add a support matrix that distinguishes:
   - production-supported
   - development-only
   - experimental
5. Document secret setup, upgrade behavior, and networking prerequisites.
6. Add release note expectations for future chart changes.

### Acceptance Criteria

- defaults and examples match the rendered chart behavior
- consumers can identify which values files are safe to adopt
- docs explain limitations without relying on maintainers to add context out of band

### Validation

- doc review across chart README, deployment docs, architecture docs, and values files
- every documented command is render-tested or install-tested

## Phase 7. CI And Release Automation

**Goal:** Make the hardening durable by automating the release gate.

### Tasks

1. Add chart-specific CI for changes under `charts/observer/**`.
2. Build a render matrix that covers:
   - default mode
   - AIO preset
   - production preset
   - external NATS
   - external PostgreSQL
   - external MongoDB
   - retained cloud-specific paths
3. Add Kubernetes schema validation for rendered manifests.
4. Add at least one cluster smoke test for the recommended public path.
5. Package the chart in CI and verify the packaged artifact is publishable.
6. Gate OCI publication on successful validation.

### Acceptance Criteria

- regressions in chart rendering are caught before merge
- public release artifacts are only produced after the validation matrix passes
- maintainers do not rely on manual local Helm checks as the primary gate

### Validation

- CI green on lint, render matrix, schema validation, and smoke tests
- package step produces a chart artifact with matching metadata and lockfile

## Milestones

## Milestone A. Wrapper-Safe Internal Consumption

**Target:** Downstream infra repo can consume the chart behind an internal wrapper without template patching.

Required phases:

- phase 0 complete
- phase 1 complete
- phase 2 complete for the chosen dependency modes
- phase 3 complete for core credentials

Expected rubric outcome:

- domains 1 through 4 score at least `2`

## Milestone B. Public Release Candidate

**Target:** The chart has a credible public contract, but optional surfaces may still be marked experimental.

Required phases:

- phases 0 through 6 complete
- at least partial CI automation in phase 7

Expected rubric outcome:

- domains 1 through 4 score `3`
- domains 5 through 8 score at least `2`

## Milestone C. Public-Ready Distribution

**Target:** The chart can be published and recommended without downstream wrappers for the supported paths.

Required phases:

- all phases complete
- hardening checklist release gate satisfied

Expected rubric outcome:

- domains 1 through 8 score at least `2`
- domains 1 through 4 score `3`
- no open blocker on installation, secrets, or external dependency correctness

## Ownership Model

The plan will move faster if work is split by workstream instead of by file.

### Recommended Owners

- **Chart contract owner:** values contract, schema, template validation, release metadata
- **Platform owner:** networking, ingress, Gateway, cloud-specific resources, cluster smoke tests
- **Security owner:** secret handling, public defaults, credential flow review
- **Application owner:** migration behavior, probes, service startup assumptions
- **Docs owner:** README, deployment docs, architecture docs, release notes

One person can hold multiple roles, but each workstream should still have a single decision-maker.

## Proposed Implementation Order

Use this order to avoid rework:

1. Define support boundary.
2. Remove duplicate templates and nil-prone values drift.
3. Fix external dependency correctness.
4. Refactor secret handling.
5. Choose and simplify migration behavior.
6. Narrow networking support to validated paths.
7. Update docs only after the contract is stable.
8. Lock the whole path in CI.

This sequence keeps documentation and CI aligned to the actual contract instead of hardening surfaces that may later be dropped or demoted.

## Immediate Next Actions

If work starts now, the first implementation batch should be:

1. remove duplicate GKE templates
2. fix external NATS startup behavior and validation
3. normalize Gateway values keys or explicitly demote Gateway support
4. align README defaults with actual chart defaults or change the defaults to match the intended public contract

Those four changes remove the most misleading public signals and create a cleaner base for the deeper secrets and CI work.

## Exit Criteria

Treat this remediation plan as complete only when:

- the hardening checklist no longer has open blocker items for the supported public paths
- the publish-readiness rubric reaches direct-consumption quality for the supported public modes
- the chart can be published without downstream template patching
- maintainers have release evidence for the published chart version

Until then, the safe operating assumption is that downstream infrastructure repositories should continue to wrap or block direct consumption of the chart.