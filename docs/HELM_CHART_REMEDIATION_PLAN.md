# Helm Chart Remediation Plan

## Phase 1 — Foundation and Validation

Goal: establish a reliable baseline for chart quality and remove immediate publish-readiness blockers.

### Scope

- Restore executable Helm validation workflow from the repository (`make helm-test`)
- Ensure chart documentation reflects real defaults and install path

### Deliverables

- [x] Add `scripts/test-helm-chart.sh` used by `make helm-test`
- [x] Validate chart with lint plus template rendering across:
  - default values
  - `values-aio.yaml`
  - `values-production.yaml`
- [x] Correct chart README defaults to match `charts/observer/values.yaml`
- [x] Correct OCI install reference in chart README

### Notes

- In constrained/offline environments, dependency fetches for subcharts may fail.
- The test script supports `HELM_REQUIRE_DEPENDENCIES=true` to fail hard when dependency retrieval is required.

## Phase 2 — Security and Runtime Hardening

Planned scope:

- tighten production values for security contexts and network exposure
- verify secret handling paths (`existingSecret`) across all sensitive values
- enforce safer defaults for internet-facing deployment profiles

## Phase 3 — Publish and Release Readiness

Planned scope:

- finalize end-to-end install/upgrade docs from OCI registry and source
- automate release gating for chart quality checks in CI
- align architecture/deployment docs with final chart behavior
