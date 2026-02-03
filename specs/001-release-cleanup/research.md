# Phase 0 Research: Release Readiness Cleanup

## Decision 1: Store readiness audit artifacts in-repo

- **Decision**: Keep the readiness checklist, evidence index, and remediation tracking under specs/001-release-cleanup/artifacts/.
- **Rationale**: In-repo artifacts provide auditable, versioned evidence and are easy to gate in CI.
- **Alternatives considered**: External ticketing system or shared document repository; rejected due to weaker traceability and CI integration.

## Decision 2: Define a curated public documentation set

- **Decision**: Curate a public documentation set composed of README.md, QUICKSTART.md, DEPLOYMENT.md, docs/README.md, and docs/architecture/.
- **Rationale**: These files cover evaluation, deployment, and architecture without exposing internal-only notes.
- **Alternatives considered**: Publish all of docs/; rejected due to outdated/experimental content risk. Use only README.md; rejected due to insufficient setup detail.

## Decision 3: Secrets scanning approach

- **Decision**: Use a history-aware secrets scan (e.g., gitleaks) plus a working tree scan prior to release gating.
- **Rationale**: Full history scanning is required by the spec to prevent hidden secrets in past commits.
- **Alternatives considered**: Working-tree only scanning; rejected as insufficient. Manual review alone; rejected as incomplete and error-prone.

## Decision 4: Licensing evidence format

- **Decision**: Record a license inventory artifact (generated list of dependencies + licenses) referenced by the readiness checklist.
- **Rationale**: Provides objective evidence for licensing gate and supports legal review.
- **Alternatives considered**: Rely on go.mod/package.json only; rejected as not sufficient for attribution or obligations tracking.
