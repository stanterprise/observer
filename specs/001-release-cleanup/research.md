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

---

# Phase 0 Findings

## Documentation audit

- **Public docs set** (per Decision 2) is present and generally coherent: README.md, QUICKSTART.md, DEPLOYMENT.md, docs/README.md, docs/architecture/.
- **Inconsistency found**: README.md listed Go 1.23+ while go.mod requires Go 1.24 with toolchain 1.24.9. (Remediation: update README to match toolchain.)
- **Credentials in examples**: Multiple docs included default `root:password` MongoDB examples. These are non-secret placeholders but should be replaced with explicit `<db-password>` / `change-me` placeholders for public release.
- **Internal-only docs**: Root-level guides like STEP*COMPONENT*\* and some docs under docs/ (e.g., implementation plans and summaries) are likely internal and should be flagged as non-public or archived. This will be tracked as cleanup tasks.

## Secrets and sensitive data scan (working tree)

- **Result**: No hardcoded private keys or tokens detected in the working tree search. Findings were limited to placeholder credentials and example passwords in documentation and editor configs.
- **Examples**: `.vscode/settings.json`, `.vscode/launch.json`, CODESPACES.md, BUILD_QUICK_REF.md, and multiple docs referenced `root:password` defaults.
- **Action**: Replace defaults with explicit placeholders and remove inline credentials from editor configs.
- **Git history**: Ran gitleaks over full history; initial false positives in docs/ATTACHMENT_STORAGE.md were allowlisted with a scoped rule. Clean scan report saved to specs/001-release-cleanup/artifacts/gitleaks-report.json.

## Build & deployment configuration review

- **Build tooling**: Makefile targets cover build, test, lint, and infra; Dockerfiles are present for each service and AIO; docker-compose profiles support AIO and distributed modes.
- **Documentation**: Build steps are documented in README.md and BUILD_QUICK_REF.md, but Go version mismatch required correction (see docs audit).
- **CI**: GitHub workflows exist for build/helm publish, but there is no explicit release readiness gate or checklist validation step yet.
- **Validation**: Build and test steps succeeded; `npm audit fix` remediated vulnerabilities (current audit clean).

## Licensing audit

- **LICENSE**: MIT license added at repo root.
- **Attribution**: ATTRIBUTION.md added with inventory references; Go module list and web license inventory generated.
- **Action**: Go license report generated; `Unknown` entries remain for github.com/stanterprise/proto-go. Confirm license in that repo and update attribution before signoff.
