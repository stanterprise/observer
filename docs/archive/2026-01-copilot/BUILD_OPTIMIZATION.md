# Build Optimization Guide

This document explains how Observer’s Docker builds are optimized using BuildKit cache mounts and `docker buildx`.

## When you need this

- You build multi-arch images (amd64 + arm64)
- You publish images from CI
- You want fast rebuilds during development

## Prerequisites

- Docker Desktop (macOS) or Docker Engine with BuildKit
- `docker buildx` available (`docker buildx version`)

## One-time setup

```bash
make docker-buildx-setup
```

This sets up a buildx builder suitable for multi-arch builds (see [../scripts/setup-buildx.sh](../scripts/setup-buildx.sh)).

## Common commands

```bash
# Standard builds (single-arch, local)
make docker-build-all

# Optimized multi-arch builds
make docker-buildx-aio
make docker-buildx-ingestion
make docker-buildx-processor
make docker-buildx-api
```

## How the caching works (high level)

- BuildKit cache mounts avoid re-downloading dependencies (Go modules, npm deps) on every build.
- Multi-stage Dockerfiles keep runtime images small and keep build tooling out of the final image.

## Troubleshooting

### `docker buildx` not found

- Ensure Docker Desktop is up-to-date.
- Verify: `docker buildx version`

### Slow builds even with buildx

- First build is expected to be slower; subsequent builds should reuse caches.
- If you changed dependency manifests (`go.mod`, `go.sum`, `web/package-lock.json`, etc.), caches will invalidate.

### Reset the build cache

```bash
make clean-cache
```

Use this if you suspect corrupted caches or want to force a clean rebuild.
