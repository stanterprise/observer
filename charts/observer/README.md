# Observer Helm Chart

This Helm chart deploys the Observer test observability system on Kubernetes. The chart defaults to distributed mode.

Shipped presets:

- `values-production.yaml`: distributed example with larger resource and HPA settings, embedded dependencies, and no chart-managed exposure resources
- `values-aio.yaml`: all-in-one evaluation preset that disables the PostgreSQL, MongoDB, and NATS subcharts
- `values-aio-gateway.yaml`: legacy AIO compatibility preset retained for existing automation names; the chart itself no longer renders Gateway API resources

## Current Contract

- Distributed mode is the primary install path.
- AIO is intended for development and evaluation.
- The chart does not render Ingress or Gateway API manifests. Exposure, TLS, and certificates are downstream infrastructure concerns.
- Distributed installs and upgrades run PostgreSQL migrations through the dedicated Helm hook job controlled by `postgres.migration.enabled`.
- External PostgreSQL, MongoDB, and NATS are configured through `postgres.*`, `externalDatabase.*`, and `externalNats.*`.
- Distributed workloads consume connection settings through Secret references. Set `runtime.existingSecret` to reuse a pre-created Secret containing `NATS_URL`, `POSTGRES_DSN`, and `MONGODB_URI`.
- If `runtime.existingSecret` is empty, the chart renders a generated distributed runtime Secret.
- `distributed.ingestion.env`, `distributed.api.env`, and `distributed.processor.env` must not set `NATS_URL`, `POSTGRES_DSN`, or `MONGODB_URI`; the chart rejects those keys during lint and template validation.
- `image.tag` defaults to the chart `appVersion` when empty, and `image.pullPolicy` auto-detects `Always` for mutable tags such as `latest`, `main`, and `develop`, and `IfNotPresent` otherwise.

## Prerequisites

- Kubernetes 1.23+
- Helm 3.8+
- PV provisioner support in the underlying infrastructure for persisted dependencies or AIO state

## Artifact Locations

- OCI Registry: `oci://ghcr.io/stanterprise/observer/charts/observer`
- GitHub Pages Helm repository: `https://stanterprise.github.io/observer/`
- Source checkout: `charts/observer/`

## Versioning Policy

- Tagged releases publish clean semantic chart versions such as `0.7.0-beta` or `0.7.0`.
- Manual testing publishes internal prerelease artifacts in the form `<chart-version>-<sha7>`.

## Installation

### From Source

```bash
cd charts/observer
helm dependency update
cd ../..

helm install observer ./charts/observer
```

### From OCI Registry

```bash
helm install observer oci://ghcr.io/stanterprise/observer/charts/observer --version <chart-version>
```

### From GitHub Pages Repository

If the release you want has been published to the Helm repository:

```bash
helm repo add observer https://stanterprise.github.io/observer/
helm repo update
helm install observer observer/observer --version <chart-version>
```

## Deployment Modes

### Distributed Mode

Distributed mode is the default. It deploys separate ingestion, processor, API, and web workloads, and it can run with embedded or external PostgreSQL, MongoDB, and NATS.

```bash
helm install observer ./charts/observer
```

When `postgres.migration.enabled=true`, install and upgrade runs include a dedicated migration hook job. The API and processor workloads do not run migrations themselves.

### AIO Mode

AIO mode runs Observer in a single container for development and evaluation.

From source, use the shipped preset:

```bash
helm install observer ./charts/observer -f ./charts/observer/values-aio.yaml
```

Or set the equivalent values explicitly:

```bash
helm install observer ./charts/observer \
  --set mode=aio \
  --set aio.enabled=true \
  --set distributed.enabled=false \
  --set postgresql.enabled=false \
  --set mongodb.enabled=false \
  --set nats.enabled=false
```

### Legacy AIO Compatibility Preset

The `values-aio-gateway.yaml` file is retained for compatibility with existing automation names, but it no longer renders Gateway API resources:

```bash
helm install observer ./charts/observer -f ./charts/observer/values-aio-gateway.yaml
```

## Key Values

| Key                                  | Purpose                                                                      | Default       |
| ------------------------------------ | ---------------------------------------------------------------------------- | ------------- |
| `mode`                               | Deployment mode                                                              | `distributed` |
| `image.tag`                          | Image tag override. Empty means the chart `appVersion`.                      | `""`          |
| `image.pullPolicy`                   | Pull policy override. Empty means auto-detect based on the effective tag.    | auto          |
| `postgresql.enabled`                 | Deploy embedded PostgreSQL                                                   | `true`        |
| `postgres.host`                      | External PostgreSQL host when `postgresql.enabled=false` in distributed mode | `""`          |
| `mongodb.enabled`                    | Deploy embedded MongoDB                                                      | `true`        |
| `externalDatabase.host`              | External MongoDB host when `mongodb.enabled=false` in distributed mode       | `""`          |
| `nats.enabled`                       | Deploy embedded NATS                                                         | `true`        |
| `externalNats.url`                   | External NATS URL when `nats.enabled=false` in distributed mode              | `""`          |
| `runtime.existingSecret`             | Existing Secret with `NATS_URL`, `POSTGRES_DSN`, and `MONGODB_URI`           | `""`          |
| `postgres.migration.enabled`         | Run the dedicated migration hook job in distributed mode                     | `true`        |
| `distributed.ingestion.service.type` | Service type for the distributed gRPC ingestion workload                     | `ClusterIP`   |
| `distributed.api.service.type`       | Service type for the distributed API workload                                | `ClusterIP`   |
| `distributed.web.service.type`       | Service type for the distributed web workload                                | `ClusterIP`   |
| `aio.service.type`                   | Service type for the AIO service                                             | `ClusterIP`   |
| `podAnnotations`                     | Additional annotations for workload pods and the migration job               | `{}`          |
| `imagePullSecrets`                   | Image pull secrets applied to workload pods and the migration job            | `[]`          |

For shared or production environments, pin `image.tag` to an immutable image tag.

## Service And Exposure Contract

The chart does not render Ingress or Gateway API resources. It exposes stable Services that downstream infrastructure can front with a load balancer, ingress controller, or gateway implementation.

Distributed mode service contract:

- `<release>-web`: HTTP on port `80`
- `<release>-api`: HTTP on port `8080`
- `<release>-ingestion`: gRPC on port `50051`

AIO service contract:

- `<release>-aio`: web `80`, API `8080`, gRPC `50051`, NATS `4222`, NATS monitor `8222`

Chart-level exposure knobs are limited to Service types and ports, for example `distributed.web.service.type`, `distributed.api.service.type`, `distributed.ingestion.service.type`, and `aio.service.type`.

## Runtime Secret Contract

Distributed workloads read `NATS_URL`, `POSTGRES_DSN`, and `MONGODB_URI` from a Secret.

- Recommended production path: create `observer-runtime-env` out of band and set `runtime.existingSecret=observer-runtime-env`.
- Generated-secret path: leave `runtime.existingSecret` empty and let the chart render the Secret for internal or testing installs.
- When using external dependencies, keep `postgres.*`, `externalDatabase.*`, and `externalNats.*` aligned with the services selected for the deployment.
- When `runtime.existingSecret` is set, create or update that Secret before `helm install` or `helm upgrade`.

Example production values:

```yaml
mode: distributed
image:
  tag: "<immutable-image-tag>"

distributed:
  enabled: true

runtime:
  existingSecret: observer-runtime-env

postgresql:
  enabled: false
postgres:
  host: postgres.example.com

mongodb:
  enabled: false
externalDatabase:
  host: mongo.example.com

nats:
  enabled: false
externalNats:
  url: nats://nats.example.com:4222
```

## Upgrading

```bash
helm upgrade observer ./charts/observer
```

Distributed upgrades run the migration hook job when `postgres.migration.enabled=true`.

If you use `runtime.existingSecret`, update the Secret before running `helm upgrade`.

The chart rejects legacy distributed connection overrides in `distributed.*.env`, so keep runtime connection settings in `runtime.existingSecret` or canonical dependency values.

## Uninstalling

```bash
helm uninstall observer
```

## Testing The Chart

```bash
helm lint ./charts/observer
helm template observer ./charts/observer
helm template observer ./charts/observer -f ./charts/observer/values-aio.yaml
helm template observer ./charts/observer -f ./charts/observer/values-production.yaml
helm template observer ./charts/observer -f ./charts/observer/values-aio-gateway.yaml
```

## Accessing The Application

Use the post-install notes for the selected mode:

```bash
helm status observer
```

For ClusterIP access in distributed mode:

```bash
kubectl port-forward svc/observer-web 8080:80
kubectl port-forward svc/observer-ingestion 50051:50051
kubectl port-forward svc/observer-api 8080:8080
```

## Support

For issues and questions, visit https://github.com/stanterprise/observer.

## License

See the main repository for license information.
