# Helm Chart Overview

The Observer Helm chart provides production-ready Kubernetes deployments with support for both All-in-One (AIO) and distributed architectures.

## Chart Location

- **Repository**: `charts/observer/`
- **Registry**: `oci://ghcr.io/stanterprise/observer/charts/observer`
- **GitHub Pages**: `https://stanterprise.github.io/observer/`

## Installation Methods

### 1. OCI Registry (Recommended)

```bash
helm install observer oci://ghcr.io/stanterprise/observer/charts/observer --version 0.1.0
```

### 2. GitHub Pages Repository

```bash
helm repo add observer https://stanterprise.github.io/observer/
helm install observer observer/observer
```

### 3. From Source

```bash
git clone https://github.com/stanterprise/observer.git
cd observer/charts/observer
helm dependency update
helm install observer .
```

## Structure

```text
charts/observer/
  Chart.yaml              # Metadata and dependencies
  values.yaml             # Default distributed mode configuration
  values-aio.yaml         # AIO mode preset
  values-production.yaml  # Production configuration preset
  README.md               # Chart documentation
  .helmignore            # Package exclusions
  templates/
    _helpers.tpl          # Reusable template functions
    serviceaccount.yaml   # Service account
    
    # AIO mode resources
    aio-deployment.yaml
    aio-service.yaml
    aio-pvc.yaml
    
    # Distributed mode resources
    ingestion-deployment.yaml
    ingestion-service.yaml
    processor-deployment.yaml
    api-deployment.yaml
    api-service.yaml
    web-deployment.yaml
    web-service.yaml
    
    # Scaling and networking
    hpa.yaml              # HorizontalPodAutoscaler
    ingress.yaml          # HTTP and gRPC ingress
    
    NOTES.txt             # Post-installation instructions
```

## Deployment Modes

### AIO (All-in-One)

Single pod with embedded SQLite and NATS. Best for development and testing.

```bash
helm install observer oci://ghcr.io/stanterprise/observer/charts/observer \
  --set mode=aio \
  --set aio.enabled=true \
  --set distributed.enabled=false \
  --set postgresql.enabled=false \
  --set nats.enabled=false
```

### Distributed

Separate services with PostgreSQL and NATS. Best for production.

```bash
helm install observer oci://ghcr.io/stanterprise/observer/charts/observer
```

## Important Values

### Deployment Mode

| Key | Description | Default |
|-----|-------------|---------|
| `mode` | Deployment mode: `aio` or `distributed` | `distributed` |
| `aio.enabled` | Enable AIO deployment | `false` |
| `distributed.enabled` | Enable distributed deployment | `true` |

### Service Scaling (Distributed Mode)

| Key | Description | Default |
|-----|-------------|---------|
| `distributed.ingestion.replicaCount` | gRPC ingestion service replicas | `2` |
| `distributed.processor.replicaCount` | Event processor replicas | `2` |
| `distributed.api.replicaCount` | API service replicas | `2` |
| `distributed.web.replicaCount` | Web UI replicas | `2` |

### Auto-Scaling

| Key | Description | Default |
|-----|-------------|---------|
| `distributed.*.autoscaling.enabled` | Enable HPA | `false` |
| `distributed.*.autoscaling.minReplicas` | Minimum replicas | `2` |
| `distributed.*.autoscaling.maxReplicas` | Maximum replicas | `10` |
| `distributed.*.autoscaling.targetCPUUtilizationPercentage` | CPU threshold | `80` |

### Database Configuration

| Key | Description | Default |
|-----|-------------|---------|
| `postgresql.enabled` | Deploy PostgreSQL | `true` |
| `postgresql.auth.username` | Database username | `postgres` |
| `postgresql.auth.password` | Database password | `postgres` |
| `postgresql.auth.database` | Database name | `observer` |
| `externalDatabase.host` | External DB host (when postgresql.enabled=false) | `""` |

### NATS Configuration

| Key | Description | Default |
|-----|-------------|---------|
| `nats.enabled` | Deploy NATS | `true` |
| `nats.config.jetstream.enabled` | Enable JetStream | `true` |
| `externalNats.url` | External NATS URL (when nats.enabled=false) | `""` |

### Ingress Configuration

| Key | Description | Default |
|-----|-------------|---------|
| `ingress.enabled` | Enable ingress for Web UI | `false` |
| `ingress.className` | Ingress class name | `nginx` |
| `ingress.hosts[0].host` | Hostname for Web UI | `observer.example.com` |
| `ingress.grpc.enabled` | Enable gRPC ingress | `false` |

### Resources

| Key | Description | Default |
|-----|-------------|---------|
| `distributed.*.resources.requests.cpu` | CPU request | varies by service |
| `distributed.*.resources.requests.memory` | Memory request | varies by service |
| `distributed.*.resources.limits.cpu` | CPU limit | varies by service |
| `distributed.*.resources.limits.memory` | Memory limit | varies by service |

## Typical Workflow

### Development/Testing

```bash
# 1. Install in AIO mode (using pre-packaged values from source)
# Note: values files are referenced without 'charts/' prefix when using OCI registry
helm install observer oci://ghcr.io/stanterprise/observer/charts/observer \
  --set mode=aio \
  --set aio.enabled=true \
  --set distributed.enabled=false

# Or from source repository:
# git clone https://github.com/stanterprise/observer.git
# helm install observer ./charts/observer -f charts/observer/values-aio.yaml

# 2. Port forward to access
kubectl port-forward svc/observer-aio 3000:80
kubectl port-forward svc/observer-aio 50051:50051

# 3. Configure test reporters to use localhost:50051
# 4. Access Web UI at http://localhost:3000
```

### Production

```bash
# 1. Install with production configuration
helm install observer oci://ghcr.io/stanterprise/observer/charts/observer \
  --set distributed.ingestion.replicaCount=3 \
  --set distributed.ingestion.autoscaling.enabled=true \
  --set ingress.enabled=true \
  --set ingress.hosts[0].host=observer.example.com

# Or from source with production values file:
# git clone https://github.com/stanterprise/observer.git
# helm install observer ./charts/observer -f charts/observer/values-production.yaml

# 2. Configure DNS to point to ingress
# 3. Test reporters send gRPC traffic to ingress endpoint
# 4. Access UI via https://observer.example.com
```

### With External Database

```bash
helm install observer oci://ghcr.io/stanterprise/observer/charts/observer \
  --set postgresql.enabled=false \
  --set externalDatabase.host=postgres.prod.example.com \
  --set externalDatabase.username=observer \
  --set externalDatabase.password=<secret-password>
```

## Dependencies

The chart depends on:

- **PostgreSQL** (Bitnami chart): Database for distributed mode
- **NATS** (Official chart): Message broker for event streaming

Dependencies are automatically managed when installing from OCI registry or GitHub Pages. When installing from source, run:

```bash
helm dependency update
```

## Upgrading

```bash
# Upgrade to latest version
helm upgrade observer oci://ghcr.io/stanterprise/observer/charts/observer

# Upgrade with new values
helm upgrade observer oci://ghcr.io/stanterprise/observer/charts/observer \
  -f custom-values.yaml
```

## Documentation

- [Chart README](../../charts/observer/README.md) - Detailed chart documentation
- [Deployment Guide](../../DEPLOYMENT.md) - Comprehensive deployment instructions
- [Quick Start](../../QUICKSTART.md) - Get started in minutes
