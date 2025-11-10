# CI/CD Strategy

## Build and Release

- Multi-arch builds: `linux/amd64, linux/arm64`  
- Buildx and GitHub Actions workflows

## Versioning
- Docker tags: `<git-sha>`, `<semver>`, `latest`
- Helm chart published to OCI registry

## Quality & Security
- SBOM: Syft  
- Vulnerability scan: Grype / Trivy  
- Linter: golangci-lint

## Testing
- Unit and integration tests  
- E2E: spin up AIO container → run Playwright sample → verify `/runs` API

## Rollouts
- AIO: container replacement (persist `/data`)  
- Distributed: Helm upgrade + rolling restart
