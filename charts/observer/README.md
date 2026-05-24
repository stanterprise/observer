# Observer Helm Chart

This Helm chart deploys the Observer test observability system on Kubernetes. The chart defaults to distributed mode, and the shipped presets cover the current supported install surfaces:

- `values-production.yaml`: distributed mode with embedded dependencies and ingress examples
- `values-aio.yaml`: all-in-one evaluation preset that disables the PostgreSQL, MongoDB, and NATS subcharts
- `values-aio-gateway.yaml`: all-in-one GKE Gateway API preset

Current chart contract:

- Distributed mode is the primary install path.
- AIO is intended for development and evaluation.
- Distributed installs run PostgreSQL migrations through the dedicated Helm hook job controlled by `postgres.migration.enabled`.
- External PostgreSQL, MongoDB, and NATS are configured through `postgres.*`, `externalDatabase.*`, and `externalNats.*`.
- Gateway API is only wired for AIO mode today.
- Distributed workloads consume connection settings through Secret references. Set `runtime.existingSecret` to reuse a pre-created Secret containing `NATS_URL`, `POSTGRES_DSN`, and `MONGODB_URI`.

## Prerequisites

- Kubernetes 1.23+
- Helm 3.8+
- PV provisioner support in the underlying infrastructure for persisted dependencies or AIO state

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
helm install observer oci://ghcr.io/stanterprise/observer/charts/observer --version 0.1.0
```

### From GitHub Pages Repository

If the release you want has been published to the Helm repository:

```bash
helm repo add observer https://stanterprise.github.io/observer/
helm repo update
helm install observer observer/observer --version 0.1.0
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

### AIO Gateway API Preset

The Gateway API path is currently an AIO-only preset for GKE-style environments:

```bash
helm install observer ./charts/observer -f ./charts/observer/values-aio-gateway.yaml
```

## Key Values

| Key                          | Purpose                                                                      | Default       |
| ---------------------------- | ---------------------------------------------------------------------------- | ------------- |
| `mode`                       | Deployment mode                                                              | `distributed` |
| `image.tag`                  | Mutable image tag default                                                    | `latest`      |
| `image.pullPolicy`           | Pull policy default                                                          | `Always`      |
| `postgresql.enabled`         | Deploy embedded PostgreSQL                                                   | `true`        |
| `postgres.host`              | External PostgreSQL host when `postgresql.enabled=false` in distributed mode | `""`          |
| `mongodb.enabled`            | Deploy embedded MongoDB                                                      | `true`        |
| `externalDatabase.host`      | External MongoDB host when `mongodb.enabled=false` in distributed mode       | `""`          |
| `nats.enabled`               | Deploy embedded NATS                                                         | `true`        |
| `externalNats.url`           | External NATS URL when `nats.enabled=false` in distributed mode              | `""`          |
| `runtime.existingSecret`     | Existing Secret with `NATS_URL`, `POSTGRES_DSN`, and `MONGODB_URI`           | `""`          |
| `postgres.migration.enabled` | Run the dedicated migration hook job in distributed mode                     | `true`        |
| `ingress.web.enabled`        | Enable web ingress                                                           | `false`       |
| `ingress.api.enabled`        | Enable API ingress                                                           | `false`       |
| `ingress.grpc.enabled`       | Enable gRPC ingress                                                          | `false`       |
| `gateway.enabled`            | Enable Gateway API resources                                                 | `false`       |
| `podAnnotations`             | Additional annotations for workload pods and the migration job               | `{}`          |
| `imagePullSecrets`           | Image pull secrets applied to workload pods and the migration job            | `[]`          |

If you want immutable image behavior, override `image.tag` and `image.pullPolicy` in your values.

## Networking

The chart treats web, API, and gRPC exposure as separate ingress surfaces:

- `ingress.web.*` configures the web UI ingress
- `ingress.api.*` configures the API ingress
- `ingress.grpc.*` configures the gRPC ingress

GKE-specific managed certificate resources are only rendered when `ingress.managedCertificate.enabled=true`. Gateway API resources are rendered only when `gateway.enabled=true` and `mode=aio`.

## External Dependency Example

```yaml
mode: distributed
distributed:
  enabled: true

postgresql:
  enabled: false
postgres:
  host: postgres.example.com
  port: 5432
  username: observer
  password: secure-password
  database: observer
  sslmode: require

mongodb:
  enabled: false
externalDatabase:
  host: mongo.example.com
  port: 27017
  username: observer
  password: secure-password
  database: observer
  authSource: admin

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

## Uninstalling

```bash
helm uninstall observer
```

## Testing the Chart

```bash
helm lint ./charts/observer
helm template observer ./charts/observer
helm template observer ./charts/observer -f ./charts/observer/values-aio.yaml
helm template observer ./charts/observer -f ./charts/observer/values-production.yaml
helm template observer ./charts/observer -f ./charts/observer/values-aio-gateway.yaml
```

## Accessing the Application

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
