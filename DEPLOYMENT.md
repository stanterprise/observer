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
docker pull ghcr.io/stanterprise/observer/ingestion:v0.1.0

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
- Gateway API is only wired for AIO mode today.
- Credentials are still passed through chart values and workload env vars today, so use non-committed overrides for sensitive values.

### Prerequisites

- Kubernetes 1.23+
- Helm 3.8+
- kubectl configured to access your cluster
- Persistent storage for AIO state or embedded distributed dependencies
- Optional ingress controller or GKE Gateway API support depending on the networking path you choose

### Quick Start

#### Method 1: Install from OCI Registry

Use this for published releases.

```bash
# Default distributed install
helm install observer oci://ghcr.io/stanterprise/observer/charts/observer --version 0.1.0

# AIO install
helm install observer oci://ghcr.io/stanterprise/observer/charts/observer \
  --version 0.1.0 \
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

helm install observer observer/observer --version 0.1.0
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

### Images

The current default image behavior is mutable:

- `image.tag=latest`
- `image.pullPolicy=Always`

Override both values if you want immutable release pinning.

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

### Networking

The chart does not use a single `ingress.enabled` flag. Configure ingress per surface:

- `ingress.web.*`
- `ingress.api.*`
- `ingress.grpc.*`

Gateway API resources are rendered only when `gateway.enabled=true` and `mode=aio`.

### Migration Behavior

Distributed installs and upgrades use the dedicated migration Helm hook job in `charts/observer/templates/migration-job.yaml` when `postgres.migration.enabled=true`.

## Helm Validation

```bash
helm lint ./charts/observer
helm template observer ./charts/observer
helm template observer ./charts/observer -f ./charts/observer/values-aio.yaml
helm template observer ./charts/observer -f ./charts/observer/values-production.yaml
helm template observer ./charts/observer -f ./charts/observer/values-aio-gateway.yaml
```

#### Ingress with TLS

```yaml
ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
  hosts:
    - host: observer.example.com
      paths:
        - path: /
          pathType: Prefix
          service: web
  tls:
    - secretName: observer-tls
      hosts:
        - observer.example.com
```

#### gRPC Ingress

```yaml
ingress:
  grpc:
    enabled: true
    className: nginx
    annotations:
      nginx.ingress.kubernetes.io/backend-protocol: "GRPC"
    hosts:
      - host: grpc.observer.example.com
        paths:
          - path: /
            pathType: Prefix
            service: ingestion
```

## Production Deployment

### Complete Production Example

```bash
# Create namespace
kubectl create namespace observer

# Create secrets for sensitive data (optional but recommended)
kubectl create secret generic observer-db-secret \
  --from-literal=password='your-secure-password' \
  -n observer

# Install with production values
helm install observer oci://ghcr.io/stanterprise/observer/charts/observer \
  --version 0.1.0 \
  --namespace observer \
  -f - <<EOF
mode: distributed

image:
  tag: "v0.1.0"  # Use specific version tag

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

mongodb:
  enabled: true
  auth:
    enabled: true
    rootPassword: "change-in-production"
    usernames: ["observer"]
    passwords: ["change-in-production"]
    databases: ["observer"]
    existingSecret: "observer-db-secret"  # Use secret instead
  persistence:
    enabled: true
    size: 100Gi
    storageClass: "fast-ssd"
  resources:
    requests: { cpu: 500m, memory: 512Mi }
    limits: { cpu: 2000m, memory: 2Gi }

nats:
  enabled: true
  config:
    jetstream:
      enabled: true
      fileStore:
        pvc:
          size: 100Gi
          storageClass: "fast-ssd"
  resources:
    requests: { cpu: 200m, memory: 256Mi }
    limits: { cpu: 1000m, memory: 1Gi }

ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
  hosts:
    - host: observer.example.com
      paths:
        - path: /
          pathType: Prefix
          service: web
  tls:
    - secretName: observer-tls
      hosts:
        - observer.example.com

  grpc:
    enabled: true
    annotations:
      nginx.ingress.kubernetes.io/backend-protocol: "GRPC"
      cert-manager.io/cluster-issuer: "letsencrypt-prod"
    hosts:
      - host: grpc.observer.example.com
        paths:
          - path: /
            pathType: Prefix
            service: ingestion
EOF
```

### Using External Services

```bash
helm install observer oci://ghcr.io/stanterprise/observer/charts/observer \
  --version 0.1.0 \
  --namespace observer \
  --set postgresql.enabled=false \
  --set postgres.host=postgres.example.com \
  --set postgres.port=5432 \
  --set postgres.username=observer \
  --set postgres.password=securepassword \
  --set postgres.database=observer \
  --set mongodb.enabled=false \
  --set externalDatabase.host=mongodb.example.com \
  --set externalDatabase.port=27017 \
  --set externalDatabase.username=observer \
  --set externalDatabase.password=securepassword \
  --set externalDatabase.database=observer \
  --set externalDatabase.authSource=admin \
  --set nats.enabled=false \
  --set externalNats.url=nats://nats.example.com:4222
```

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
kubectl logs -n observer observer-mongodb-0

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

### Ingress Not Working

```bash
# Check ingress status
kubectl describe ingress observer -n observer

# Verify ingress controller is running
kubectl get pods -n ingress-nginx

# Check TLS certificate (if using cert-manager)
kubectl describe certificate observer-tls -n observer
```

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
# Update Helm repository
helm repo update observer

# Upgrade to latest version
helm upgrade observer observer/observer --namespace observer

# Upgrade with new values
helm upgrade observer observer/observer \
  --namespace observer \
  -f custom-values.yaml

# Rollback if needed
helm rollback observer -n observer
```

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
