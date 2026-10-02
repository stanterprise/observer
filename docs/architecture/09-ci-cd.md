# CI/CD Strategy

## Build and Release

- Multi-architecture Docker builds target `linux/amd64` and `linux/arm64` for non-PR builds.
- Buildx and GitHub Actions workflows publish the service images.
- Helm charts are published to the OCI registry.

## Versioning

- Docker tags include branch, commit SHA, semantic-version, and `latest` tags as configured by the active publishing workflow.
- Helm chart releases use semantic chart versions; manual artifacts may include a commit suffix.

## Quality & Security

- `golangci-lint` is available as a repository development target.
- Syft SBOM generation and Grype/Trivy vulnerability scans appeared in earlier CI plans; they are not guaranteed current workflow jobs unless present in `.github/workflows/`.

## Testing

- Unit and integration tests are maintained in the Go packages and `tests/`.
- NATS integration tests exercise event flow.
- AIO smoke testing can run a Playwright sample and verify the REST API at `/api/runs`.
- CI job details are defined in [`.github/workflows/`](../../.github/workflows/).

## Rollouts

- AIO: replace the container while preserving the mounted `/data` volume.
- Distributed: apply a Helm upgrade and allow Kubernetes rolling updates to complete.
- PostgreSQL migrations are run through the chart migration hook; application rollback does not automatically reverse schema migrations.
