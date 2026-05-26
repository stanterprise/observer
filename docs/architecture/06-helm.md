# Helm Chart Overview

The Observer Helm chart lives in `charts/observer/` and defaults to distributed mode. It also ships AIO presets for development and evaluation workflows.

## Current Contract

- Distributed mode is the primary multi-workload install path.
- AIO presets disable the PostgreSQL, MongoDB, and NATS subcharts and run the stack in one Observer container.
- The chart does not render Ingress or Gateway API resources. Exposure, TLS, and certificates are downstream infrastructure concerns.
- Distributed installs and upgrades run PostgreSQL migrations through the dedicated Helm hook job in `templates/distributed/migration/migration-job.yaml` when `postgres.migration.enabled=true`.
- External PostgreSQL, MongoDB, and NATS are configured through `postgres.*`, `externalDatabase.*`, and `externalNats.*`.
- Distributed workloads read runtime connection settings from Secret references and can reuse a pre-created Secret through `runtime.existingSecret`.
- `distributed.*.env` may not set `NATS_URL`, `POSTGRES_DSN`, or `MONGODB_URI`; those keys are rejected during validation.
- `image.tag` defaults to the chart `appVersion` when empty, and `image.pullPolicy` auto-detects mutable vs immutable tags.

## Artifact Locations

- **Repository**: `charts/observer/`
- **OCI Registry**: `oci://ghcr.io/stanterprise/observer/charts/observer`
- **GitHub Pages Repository**: `https://stanterprise.github.io/observer/`

## Versioning Policy

- Tagged releases publish clean semantic chart versions.
- Manual testing publishes prerelease artifacts in the form `<chart-version>-<sha7>`.

## Install Methods

### OCI Registry

```bash
helm install observer oci://ghcr.io/stanterprise/observer/charts/observer --version <chart-version>
```

### GitHub Pages Repository

If the release has been published to the Helm repository:

```bash
helm repo add observer https://stanterprise.github.io/observer/
helm repo update
helm install observer observer/observer --version <chart-version>
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
  Chart.yaml
  Chart.lock
  values.yaml
  values-production.yaml
  values-aio.yaml
  values-aio-gateway.yaml
  values.schema.json
  templates/
    _helpers.tpl
    validate.yaml
    serviceaccount.yaml
    NOTES.txt
    aio/
      deployment/aio-deployment.yaml
      pvc/aio-pvc.yaml
      service/aio-service.yaml
    distributed/
      api/
        api-deployment.yaml
        api-service.yaml
      ingestion/
        ingestion-deployment.yaml
        ingestion-service.yaml
      processor/
        processor-deployment.yaml
      web/
        web-deployment.yaml
        web-service.yaml
      migration/
        migration-job.yaml
      runtime-secret.yaml
      hpa.yaml
```

## Modes And Presets

### Distributed Mode

Distributed mode is the default and deploys separate ingestion, processor, API, and web workloads.

```bash
helm install observer oci://ghcr.io/stanterprise/observer/charts/observer --version <chart-version>
```

### AIO Preset

The AIO preset is intended for development and evaluation and disables the PostgreSQL, MongoDB, and NATS subcharts.

```bash
helm install observer ./charts/observer -f charts/observer/values-aio.yaml
```

### Legacy AIO Compatibility Preset

The `values-aio-gateway.yaml` preset is retained for compatibility with existing automation names, but it no longer renders Gateway API resources.

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

## Service And Exposure Contract

The chart only manages Services and workload ports. Downstream infrastructure is expected to manage ingress, gateway, load balancer, and certificate resources.

Distributed mode service contract:

- `<release>-web`: HTTP on port `80`
- `<release>-api`: HTTP on port `8080`
- `<release>-ingestion`: gRPC on port `50051`

AIO service contract:

- `<release>-aio`: web `80`, API `8080`, gRPC `50051`, NATS `4222`, NATS monitor `8222`

## External Dependency Contract

Distributed installs can disable embedded dependencies, but the chart requires the matching external selectors:

- `postgresql.enabled=false` requires `postgres.host`
- `mongodb.enabled=false` requires `externalDatabase.host`
- `nats.enabled=false` requires `externalNats.url`

Example:

```bash
helm install observer ./charts/observer \
  --set runtime.existingSecret=observer-runtime-env \
  --set postgresql.enabled=false \
  --set postgres.host=postgres.example.com \
  --set mongodb.enabled=false \
  --set externalDatabase.host=mongo.example.com \
  --set nats.enabled=false \
  --set externalNats.url=nats://nats.example.com:4222
```

## Runtime Secret Contract

- Distributed workloads read `NATS_URL`, `POSTGRES_DSN`, and `MONGODB_URI` from Secret references.
- The chart renders a runtime Secret by default for distributed mode when `runtime.existingSecret` is empty.
- Operators can set `runtime.existingSecret` to reuse a pre-created Secret with those keys.
- If `runtime.existingSecret` is set, create or update that Secret before install or upgrade.
- Embedded dependency credentials continue to follow the dependency-chart auth configuration.

## Upgrade And Migration Behavior

Distributed installs and upgrades run PostgreSQL migrations through `templates/distributed/migration/migration-job.yaml`. The API and processor deployments no longer run migrations themselves.

- When the chart manages the runtime Secret, the migration hook computes `POSTGRES_DSN` directly from canonical PostgreSQL values.
- When `runtime.existingSecret` is set, that Secret must already contain `POSTGRES_DSN`.
- Rollbacks do not reverse schema migrations automatically.

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
