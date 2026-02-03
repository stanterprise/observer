# Quickstart: Release Readiness Cleanup

## Prerequisites

- Go toolchain (see go.mod toolchain)
- Node.js (for web build)
- Docker (for infrastructure checks)
- Access to CI configuration

## Steps

1. Review the curated public documentation set:
   - README.md
   - QUICKSTART.md
   - DEPLOYMENT.md
   - docs/README.md
   - docs/architecture/
2. Run baseline checks on a clean checkout:
   - Build all components
   - Run go test ./tests
   - Run web build (npm run build in web/)
3. Perform secrets scanning:
   - Working tree scan
   - Full git history scan
4. Generate/update readiness artifacts:
   - readiness checklist
   - evidence index
   - cleanup tasks
   - risk register (if needed)
5. Record the release decision with rationale in the checklist.

## Expected Outputs

- Updated artifacts under specs/001-release-cleanup/artifacts/ with pass/fail status, evidence paths, and remediation ownership.
