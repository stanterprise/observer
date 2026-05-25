# Helm Chart Overview

The Observer Helm chart lives in `charts/observer/` and defaults to distributed mode. It also ships AIO presets for development and evaluation workflows.

## Current Contract

- Distributed mode is the primary multi-workload install path.
- AIO presets disable the PostgreSQL, MongoDB, and NATS subcharts and run the stack in one Observer container.
- Distributed installs and upgrades run PostgreSQL migrations through the dedicated Helm hook job in `templates/migration-job.yaml` when `postgres.migration.enabled=true`.
- External PostgreSQL, MongoDB, and NATS are configured through `postgres.*`, `externalDatabase.*`, and `externalNats.*`.
- Distributed workloads read runtime connection settings from Secret references and can reuse a pre-created Secret through `runtime.existingSecret`.
- Gateway API resources are only wired for AIO mode today.
- The chart currently uses mutable image defaults: `image.tag=latest` and `image.pullPolicy=Always`.

## Artifact Locations

- **Repository**: `charts/observer/`
- **OCI Registry**: `oci://ghcr.io/stanterprise/observer/charts/observer`
- **GitHub Pages Repository**: `https://stanterprise.github.io/observer/`

## Install Methods

### OCI Registry

```bash
helm install observer oci://ghcr.io/stanterprise/observer/charts/observer --version 0.1.0
```

### GitHub Pages Repository

If the release has been published to the Helm repository:

```bash
helm repo add observer https://stanterprise.github.io/observer/
helm repo update
helm install observer observer/observer --version 0.1.0
```

### Source Checkout

```bash
git clone https://github.com/stanterprise/observer.git
cd observer/charts/observer
helm dependency update
helm install observer .
```

## Chart Structure

```text
charts/observer/
  Chart.yaml                # Chart metadata and dependency declarations
  Chart.lock                # Locked dependency versions
  values.yaml               # Default distributed-mode configuration
  values-production.yaml    # Distributed example preset
  values-aio.yaml           # AIO preset
  values-aio-gateway.yaml   # AIO Gateway API preset
  values.schema.json        # Schema validation for chart values
  templates/
    aio-deployment.yaml
    aio-service.yaml
    aio-pvc.yaml
    ingestion-deployment.yaml
    ingestion-service.yaml
    processor-deployment.yaml
    api-deployment.yaml
    api-service.yaml
    web-deployment.yaml
    web-service.yaml
    migration-job.yaml      # Dedicated PostgreSQL migration hook job
    hpa.yaml
    ingress.yaml            # Per-surface ingress resources
    gateway.yaml            # AIO-only Gateway API resources
    gateway-certificate.yaml
    backendconfig.yaml
    grpc-loadbalancer.yaml
    NOTES.txt
```

## Modes And Presets

### Distributed Mode

Distributed mode is the default and deploys separate ingestion, processor, API, and web workloads.

```bash
helm install observer oci://ghcr.io/stanterprise/observer/charts/observer --version 0.1.0
```

### AIO Preset

The AIO preset is intended for development and evaluation and disables the PostgreSQL, MongoDB, and NATS subcharts.

```bash
helm install observer ./charts/observer -f charts/observer/values-aio.yaml
```

### AIO Gateway API Preset

The Gateway API preset is the current GKE-style AIO path.

```bash
helm install observer ./charts/observer -f charts/observer/values-aio-gateway.yaml
```

## Core Values

| Key                                  | Description                                                                  | Default       |
| ------------------------------------ | ---------------------------------------------------------------------------- | ------------- |
| `mode`                               | Deployment mode                                                              | `distributed` |
| `aio.enabled`                        | Enable AIO resources                                                         | `false`       |
| `distributed.enabled`                | Enable distributed resources                                                 | `true`        |
| `distributed.ingestion.replicaCount` | Ingestion replicas                                                           | `1`           |
| `distributed.processor.replicaCount` | Processor replicas                                                           | `1`           |
| `distributed.api.replicaCount`       | API replicas                                                                 | `1`           |
| `distributed.web.replicaCount`       | Web replicas                                                                 | `1`           |
| `postgresql.enabled`                 | Deploy embedded PostgreSQL                                                   | `true`        |
| `postgres.host`                      | External PostgreSQL host when `postgresql.enabled=false` in distributed mode | `""`          |
| `mongodb.enabled`                    | Deploy embedded MongoDB                                                      | `true`        |
| `externalDatabase.host`              | External MongoDB host when `mongodb.enabled=false` in distributed mode       | `""`          |
| `nats.enabled`                       | Deploy embedded NATS                                                         | `true`        |
| `externalNats.url`                   | External NATS URL when `nats.enabled=false` in distributed mode              | `""`          |
| `runtime.existingSecret`             | Existing Secret with `NATS_URL`, `POSTGRES_DSN`, and `MONGODB_URI`           | `""`          |
| `postgres.migration.enabled`         | Enable the dedicated migration hook job in distributed mode                  | `true`        |

## Networking Contract

Ingress configuration is split by surface:

- `ingress.web.*`
- `ingress.api.*`
- `ingress.grpc.*`

There is no single `ingress.enabled` switch in the current contract. Gateway API resources are only rendered when `gateway.enabled=true` and `mode=aio`.

## External Dependency Contract

Distributed installs can disable embedded dependencies, but the chart requires the matching external settings:

- `postgresql.enabled=false` requires `postgres.host`
- `mongodb.enabled=false` requires `externalDatabase.host`
- `nats.enabled=false` requires `externalNats.url`

Example:

```bash
helm install observer ./charts/observer \
  --set postgresql.enabled=false \
  --set postgres.host=postgres.example.com \
  --set mongodb.enabled=false \
  --set externalDatabase.host=mongo.example.com \
  --set nats.enabled=false \
  --set externalNats.url=nats://nats.example.com:4222
```

## Secret Handling Contract

- Distributed workloads read `NATS_URL`, `POSTGRES_DSN`, and `MONGODB_URI` from Secret references.
- The chart renders a runtime Secret by default for distributed mode.
- Operators can set `runtime.existingSecret` to reuse a pre-created Secret with those keys.
- Embedded dependency credentials continue to follow the dependency-chart auth configuration.

## Upgrade And Migration Behavior

Distributed installs and upgrades run PostgreSQL migrations through `templates/migration-job.yaml`. The API and processor deployments no longer run migrations themselves.

## Dependencies

The chart currently depends on:

- Bitnami PostgreSQL
- Bitnami MongoDB
- NATS Helm chart

When installing from source, refresh dependency charts with:

```bash
helm dependency update
```

## Validation

```bash
helm lint ./charts/observer
helm template observer ./charts/observer
helm template observer ./charts/observer -f ./charts/observer/values-aio.yaml
helm template observer ./charts/observer -f ./charts/observer/values-production.yaml
helm template observer ./charts/observer -f ./charts/observer/values-aio-gateway.yaml
```

## Documentation

- [Chart README](../../charts/observer/README.md) - Day-to-day chart usage
- [Deployment Guide](../../DEPLOYMENT.md) - Installation and operational examples
- [Quick Start](../../QUICKSTART.md) - Repository quick start
