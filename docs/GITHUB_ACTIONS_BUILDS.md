# GitHub Actions Workflows

This document summarizes the CI/CD workflows in [../.github/workflows/](../.github/workflows/).

## Docker image publishing

Workflow: [../.github/workflows/docker-publish.yml](../.github/workflows/docker-publish.yml)

What it does:

- Builds and publishes Docker images (AIO + per-service images)
- Uses BuildKit caching to speed up builds
- Produces multi-architecture images when configured

## Build performance validation

Workflow: [../.github/workflows/build-performance.yml](../.github/workflows/build-performance.yml)

What it does:

- Runs periodic builds to confirm caching and build times stay healthy
- Writes links and summaries to the GitHub Actions job summary

## How to reproduce locally

- Use the Make targets that CI uses:

```bash
make build-all
make test
make docker-build-all
```

- For multi-arch builds and caching, see: [BUILD_OPTIMIZATION.md](BUILD_OPTIMIZATION.md)

## Customizing

If you change image names, tags, or build contexts, update both:

- Dockerfiles / `docker-compose.yml`
- The workflow files under [../.github/workflows/](../.github/workflows/)
