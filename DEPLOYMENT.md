# Observer Deployment Guide

This guide covers deploying the Observer test observability system using Docker images and Helm charts.

## Table of Contents

1. [Docker Images](#docker-images)
2. [Helm Chart Installation](#helm-chart-installation)
3. [Deployment Modes](#deployment-modes)
4. [Configuration](#configuration)
5. [Production Deployment](#production-deployment)
6. [Troubleshooting](#troubleshooting)

## Docker Images

Observer provides pre-built Docker images for all components, published to GitHub Container Registry (ghcr.io).

### Available Images

All images are available at `ghcr.io/stanterprise/observer/`:

- **aio**: All-in-One image with embedded MongoDB, PostgreSQL, NATS, and application services
- **ingestion**: gRPC ingestion service
- **processor**: Event processor service
- **api**: REST/GraphQL API service with WebSocket support
- **web**: Web UI (React + Nginx)

### Image Tags

Images are tagged with:

- `latest` - Latest build from main branch
- `main` - Main branch build
- `develop` - Develop branch build
- `v*.*.*` - Semantic version releases
- `main-<sha>` - Specific commit from main branch

### Pulling Images

```bash
# Pull latest AIO image
docker pull ghcr.io/stanterprise/observer/aio:latest

# Pull specific version
docker pull ghcr.io/stanterprise/observer/ingestion:<image-tag>

# Pull all distributed mode images
docker pull ghcr.io/stanterprise/observer/ingestion:latest
docker pull ghcr.io/stanterprise/observer/processor:latest
docker pull ghcr.io/stanterprise/observer/api:latest
docker pull ghcr.io/stanterprise/observer/web:latest
```

### Running with Docker

#### AIO Mode

## Helm Chart Installation

### Contract Summary

The current Helm chart contract is:

- Distributed mode is the default and primary install path.
- AIO is a development and evaluation path.
- The shipped AIO presets disable the PostgreSQL, MongoDB, and NATS subcharts.
- Distributed installs run PostgreSQL migrations through the dedicated Helm hook job controlled by `postgres.migration.enabled`.
- External PostgreSQL, MongoDB, and NATS require `postgres.host`, `externalDatabase.host`, and `externalNats.url` when their embedded dependencies are disabled.
- The chart does not render Ingress or Gateway API manifests; downstream infrastructure owns exposure and TLS.
- Distributed workloads consume connection settings through Secret references. Use `runtime.existingSecret` to reuse a Secret containing `NATS_URL`, `POSTGRES_DSN`, and `MONGODB_URI`, or let the chart render its generated runtime Secret.

### Prerequisites

- Kubernetes 1.23+
- Helm 3.8+
- kubectl configured to access your cluster
- Persistent storage for AIO state or embedded distributed dependencies
- Optional downstream exposure controller, load balancer, or gateway implementation depending on how you publish Services

### Quick Start

#### Method 1: Install from OCI Registry

Use this for published releases.

```bash
# Default distributed install
helm install observer oci://ghcr.io/stanterprise/observer/charts/observer --version <chart-version>

# AIO install
helm install observer oci://ghcr.io/stanterprise/observer/charts/observer \
  --version <chart-version> \
  --set mode=aio \
  --set aio.enabled=true \
  --set distributed.enabled=false \
  --set postgresql.enabled=false \
  --set mongodb.enabled=false \
  --set nats.enabled=false
```

#### Method 2: Install from GitHub Pages

If the release you are consuming has been published to the Helm repository:

```bash
helm repo add observer https://stanterprise.github.io/observer/
helm repo update

helm install observer observer/observer --version <chart-version>
```

#### Method 3: Install from Source

Use this when working from the repository or the shipped preset files.

```bash
git clone https://github.com/stanterprise/observer.git
cd observer

cd charts/observer
helm dependency update
cd ../..

helm install observer ./charts/observer
```

### Verify Installation

```bash
helm status observer
kubectl get pods -w
kubectl get all -l app.kubernetes.io/instance=observer
```

## Deployment Modes

### AIO (All-in-One) Mode

Best for development, evaluation, and small-scale demos.

Characteristics:

- Single Observer workload
- No PostgreSQL, MongoDB, or NATS subcharts when using the shipped AIO presets
- Simplified networking and persistence options

Install from source with the preset:

```bash
helm install observer ./charts/observer -f ./charts/observer/values-aio.yaml
```

For the GKE Gateway API preset:

```bash
helm install observer ./charts/observer -f ./charts/observer/values-aio-gateway.yaml
```

Access example:

```bash
kubectl port-forward svc/observer-aio 3000:80
kubectl port-forward svc/observer-aio 50051:50051
```

### Distributed Mode

Best for the primary multi-workload deployment path.

Characteristics:

- Separate ingestion, processor, API, and web workloads
- Embedded or external PostgreSQL, MongoDB, and NATS
- Dedicated migration hook job for schema changes
- Separate ingress surfaces for web, API, and gRPC

Default install:

```bash
helm install observer ./charts/observer
```

All-external dependency example:

```bash
helm install observer ./charts/observer \
  --set postgresql.enabled=false \
  --set postgres.host=postgres.example.com \
  --set postgres.username=observer \
  --set postgres.password=securepassword \
  --set postgres.database=observer \
  --set mongodb.enabled=false \
  --set externalDatabase.host=mongodb.example.com \
  --set externalDatabase.username=observer \
  --set externalDatabase.password=securepassword \
  --set externalDatabase.database=observer \
  --set nats.enabled=false \
  --set externalNats.url=nats://nats.example.com:4222
```

Services:

- `observer-ingestion`: gRPC endpoint on port 50051
- `observer-processor`: background processor
- `observer-api`: REST API and WebSocket endpoint on port 8080
- `observer-web`: web UI on port 80

## Configuration

### External Dependencies

Use these keys when disabling embedded distributed dependencies:

| Embedded Dependency | Disable Key                | Required External Key   |
| ------------------- | -------------------------- | ----------------------- |
| PostgreSQL          | `postgresql.enabled=false` | `postgres.host`         |
| MongoDB             | `mongodb.enabled=false`    | `externalDatabase.host` |
| NATS                | `nats.enabled=false`       | `externalNats.url`      |

### Secret-Backed Runtime Config

Distributed mode reads `NATS_URL`, `POSTGRES_DSN`, and `MONGODB_URI` from a Secret.

- Leave `runtime.existingSecret` empty to let the chart render a generated runtime Secret for distributed workloads.
- Set `runtime.existingSecret` to reuse a pre-created Secret with those three keys. This is the recommended production path.
- When `runtime.existingSecret` is set, create or update that Secret before `helm install` or `helm upgrade`.
- `distributed.ingestion.env`, `distributed.api.env`, and `distributed.processor.env` must not set `NATS_URL`, `POSTGRES_DSN`, or `MONGODB_URI`; the chart rejects those keys during validation.
- External dependency selectors still use `postgres.*`, `externalDatabase.*`, and `externalNats.*`.
- Embedded dependency credentials still use the dependency-chart auth configuration.

### Images

Default image behavior:

- `image.tag=""` inherits the chart `appVersion`.
- `image.pullPolicy` auto-detects `Always` for mutable tags such as `latest`, `main`, and `develop`, and `IfNotPresent` otherwise.

For shared environments, pin `image.tag` to an immutable image tag.

### Persistence

For AIO:

```yaml
aio:
  persistence:
    enabled: true
    size: 10Gi
```

For embedded distributed dependencies, see [charts/observer/values-production.yaml](charts/observer/values-production.yaml) for larger example sizes.

### Scaling

Distributed workloads support both manual replica counts and HPA settings through `distributed.*.replicaCount` and `distributed.*.autoscaling.*`.

The chart does not render PodDisruptionBudgets. Operators that need disruption budgets should add them downstream.

### Networking

The chart does not render Ingress or Gateway API manifests. Exposure and TLS are downstream infrastructure concerns.

Chart-level exposure knobs are limited to Services:

- `distributed.ingestion.service.type`
- `distributed.api.service.type`
- `distributed.web.service.type`
- `aio.service.type`
- `aio.grpcLoadBalancer.enabled` for the direct AIO gRPC LoadBalancer Service

Stable service contract:

- distributed mode: `observer-web` HTTP `80`, `observer-api` HTTP `8080`, `observer-ingestion` gRPC `50051`
- AIO mode: `observer-aio` exposes web `80`, API `8080`, gRPC `50051`, NATS `4222`, and NATS monitor `8222`

### Migration Behavior

Distributed installs and upgrades use the dedicated migration Helm hook job in `charts/observer/templates/distributed/migration/migration-job.yaml` when `postgres.migration.enabled=true`.

- When the chart manages the runtime Secret, the hook computes `POSTGRES_DSN` directly from canonical PostgreSQL values.
- When `runtime.existingSecret` is set, that Secret must already contain `POSTGRES_DSN`.
- The API and processor workloads do not run migrations themselves.
- Rollbacks do not reverse schema migrations automatically.

## Helm Validation

```bash
helm lint ./charts/observer
helm template observer ./charts/observer
helm template observer ./charts/observer -f ./charts/observer/values-aio.yaml
helm template observer ./charts/observer -f ./charts/observer/values-production.yaml
helm template observer ./charts/observer -f ./charts/observer/values-aio-gateway.yaml
```

## Production Deployment

### Recommended Production Pattern

```bash
# Create namespace
kubectl create namespace observer

# Create or sync observer-runtime-env out of band through your secret manager.
# Required keys: NATS_URL, POSTGRES_DSN, MONGODB_URI.

# Install with pinned chart and image versions
helm install observer oci://ghcr.io/stanterprise/observer/charts/observer \
  --version <chart-version> \
  --namespace observer \
  -f - <<'EOF'
mode: distributed

image:
  tag: "<immutable-image-tag>"

distributed:
  enabled: true

  ingestion:
    replicaCount: 3
    autoscaling:
      enabled: true
      minReplicas: 3
      maxReplicas: 10
    resources:
      requests: { cpu: 200m, memory: 256Mi }
      limits: { cpu: 1000m, memory: 512Mi }

  processor:
    replicaCount: 3
    autoscaling:
      enabled: true
      minReplicas: 3
      maxReplicas: 10
    resources:
      requests: { cpu: 500m, memory: 512Mi }
      limits: { cpu: 2000m, memory: 2Gi }

  api:
    replicaCount: 3
    autoscaling:
      enabled: true
      minReplicas: 3
      maxReplicas: 8
    resources:
      requests: { cpu: 200m, memory: 256Mi }
      limits: { cpu: 1000m, memory: 1Gi }

  web:
    replicaCount: 2
    resources:
      requests: { cpu: 100m, memory: 128Mi }
      limits: { cpu: 500m, memory: 256Mi }

runtime:
  existingSecret: observer-runtime-env

postgresql:
  enabled: false
postgres:
  host: postgresql.example.com

mongodb:
  enabled: false
externalDatabase:
  host: mongodb.example.com

nats:
  enabled: false
externalNats:
  url: nats://nats.example.com:4222
EOF
```

### Using External Services

```bash
helm install observer oci://ghcr.io/stanterprise/observer/charts/observer \
  --version <chart-version> \
  --namespace observer \
  --set runtime.existingSecret=observer-runtime-env \
  --set postgresql.enabled=false \
  --set postgres.host=postgres.example.com \
  --set mongodb.enabled=false \
  --set externalDatabase.host=mongodb.example.com \
  --set nats.enabled=false \
  --set externalNats.url=nats://nats.example.com:4222
```

If you use `runtime.existingSecret`, update the Secret before running `helm upgrade`.

### Monitoring and Observability

```bash
# Check pod status
kubectl get pods -n observer

# View logs
kubectl logs -n observer -l app.kubernetes.io/component=ingestion -f
kubectl logs -n observer -l app.kubernetes.io/component=processor -f
kubectl logs -n observer -l app.kubernetes.io/component=api -f

# Check resource usage
kubectl top pods -n observer
kubectl top nodes

# Describe deployments
kubectl describe deployment -n observer observer-ingestion
```

## Troubleshooting

### Pods Not Starting

```bash
# Check pod status and events
kubectl describe pod <pod-name> -n observer

# Check logs
kubectl logs <pod-name> -n observer

# Common issues:
# 1. Image pull errors - verify image tags and registry access
# 2. Resource limits - check node resources
# 3. PVC binding - check storage class and availability
```

### Database Connection Issues

```bash
# Test MongoDB connection from a pod
kubectl exec -it <processor-pod> -n observer -- sh
# Inside pod: check MONGODB_URI environment variable

# Check MongoDB pod
kubectl logs -n observer deploy/observer-mongodb

# Verify service endpoints
kubectl get endpoints -n observer
```

### NATS Connection Issues

```bash
# Check NATS pod
kubectl logs -n observer observer-nats-0

# Check JetStream status
kubectl exec -it observer-nats-0 -n observer -- nats stream ls

# Verify NATS service
kubectl get svc observer-nats -n observer
```

### Exposure Not Working

```bash
# Check service status and endpoints
kubectl get svc -n observer
kubectl get endpoints -n observer

# If you manage ingress or gateway resources outside the chart, inspect them separately
kubectl get ingress,gateway,httproute,grpcroute -A

# Verify your controller is running
kubectl get pods -n ingress-nginx
```

The chart does not render Ingress or Gateway API manifests.

### Performance Issues

```bash
# Check HPA status
kubectl get hpa -n observer

# Scale manually if needed
kubectl scale deployment observer-ingestion --replicas=5 -n observer

# Check resource limits
kubectl describe deployment observer-processor -n observer
```

## Upgrading

```bash
# Upgrade from the OCI registry with a pinned chart version
helm upgrade observer oci://ghcr.io/stanterprise/observer/charts/observer \
  --namespace observer \
  --version <chart-version> \
  -f custom-values.yaml

# Rollback if needed
helm rollback observer <revision> -n observer
```

If you use `runtime.existingSecret`, update the Secret before running `helm upgrade`.

Distributed upgrades run the migration hook job when `postgres.migration.enabled=true`.

Rollbacks do not reverse schema migrations automatically; review migration compatibility before rolling back application code.

## Uninstalling

```bash
# Uninstall the release
helm uninstall observer -n observer

# Delete PVCs if needed (data will be lost!)
kubectl delete pvc -n observer -l app.kubernetes.io/instance=observer

# Delete namespace
kubectl delete namespace observer
```

## Additional Resources

- [Helm Chart README](charts/observer/README.md)
- [Repository README](README.md)
- [Architecture Documentation](docs/architecture/)
- [GitHub Repository](https://github.com/stanterprise/observer)

## Support

For issues and questions:

- GitHub Issues: https://github.com/stanterprise/observer/issues
- Documentation: https://github.com/stanterprise/observer/tree/main/docs
