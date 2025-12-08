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

- **aio**: All-in-One image with all services embedded
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

```bash
docker run -d \
  --name observer \
  -p 3000:80 \
  -p 50051:50051 \
  -p 8080:8080 \
  -v observer-data:/data \
  ghcr.io/stanterprise/observer/aio:latest
```

Access:
- Web UI: http://localhost:3000
- gRPC: localhost:50051
- API: http://localhost:8080

#### Distributed Mode with Docker Compose

See [docker-compose.yml](docker-compose.yml) for the full configuration.

```bash
# Clone the repository for docker-compose.yml
git clone https://github.com/stanterprise/observer.git
cd observer

# Update docker-compose.yml to use published images
# Change image: observer:ingestion to ghcr.io/stanterprise/observer/ingestion:latest

# Start distributed mode
docker compose --profile dist up -d
```

## Helm Chart Installation

### Prerequisites

- Kubernetes 1.23+
- Helm 3.8+
- kubectl configured to access your cluster
- (Optional) Ingress controller (nginx, traefik, etc.)
- (Optional) cert-manager for TLS certificates

### Quick Start

#### Method 1: Install from OCI Registry (Recommended)

```bash
# Install with default distributed mode
helm install observer oci://ghcr.io/stanterprise/observer/charts/observer --version 0.1.0

# Install in AIO mode
helm install observer oci://ghcr.io/stanterprise/observer/charts/observer \
  --version 0.1.0 \
  --set mode=aio \
  --set aio.enabled=true \
  --set distributed.enabled=false
```

#### Method 2: Install from GitHub Pages

```bash
# Add the Helm repository
helm repo add observer https://stanterprise.github.io/observer/
helm repo update

# Install the chart
helm install observer observer/observer

# Install in AIO mode with values file
helm install observer observer/observer --set mode=aio --set aio.enabled=true --set distributed.enabled=false
```

#### Method 3: Install from Source

```bash
# Clone the repository
git clone https://github.com/stanterprise/observer.git
cd observer

# Update Helm dependencies
cd charts/observer
helm dependency update
cd ../..

# Install the chart
helm install observer ./charts/observer

# Or with custom values
helm install observer ./charts/observer -f ./charts/observer/values-production.yaml
```

### Verify Installation

```bash
# Check deployment status
helm status observer

# Watch pods starting
kubectl get pods -w

# Check all resources
kubectl get all -l app.kubernetes.io/instance=observer
```

## Deployment Modes

### AIO (All-in-One) Mode

Best for: Development, testing, proof-of-concept, small-scale deployments

**Features:**
- Single pod deployment
- Embedded MongoDB database
- Embedded NATS server
- All services in one container
- Lower resource requirements
- Simple configuration

**Installation:**

```bash
helm install observer oci://ghcr.io/stanterprise/observer/charts/observer \
  --version 0.1.0 \
  --set mode=aio \
  --set aio.enabled=true \
  --set distributed.enabled=false \
  --set mongodb.enabled=false \
  --set nats.enabled=false
```

**Access:**

```bash
# Port forward to access services
kubectl port-forward svc/observer-aio 3000:80
kubectl port-forward svc/observer-aio 50051:50051

# Web UI: http://localhost:3000
# gRPC: localhost:50051
```

### Distributed Mode

Best for: Production, CI/CD, high-scale deployments, high availability

**Features:**
- Separate pods for each service
- Horizontal scaling with HPA
- External or embedded MongoDB
- External or embedded NATS with JetStream
- Independent service scaling
- Production-ready architecture

**Installation:**

```bash
# With embedded MongoDB and NATS
helm install observer oci://ghcr.io/stanterprise/observer/charts/observer --version 0.1.0

# With external MongoDB
helm install observer oci://ghcr.io/stanterprise/observer/charts/observer \
  --version 0.1.0 \
  --set mongodb.enabled=false \
  --set externalDatabase.host=mongodb.example.com \
  --set externalDatabase.username=observer \
  --set externalDatabase.password=securepassword \
  --set externalDatabase.database=observer
```

**Services:**
- `observer-ingestion`: gRPC endpoint (port 50051)
- `observer-processor`: Event processor (no external port)
- `observer-api`: REST/GraphQL API + WebSocket (port 8080)
- `observer-web`: Web UI (port 80)

## Configuration

### Resource Requirements

#### Minimum (Development)

```yaml
distributed:
  ingestion:
    resources:
      requests: { cpu: 100m, memory: 128Mi }
      limits: { cpu: 500m, memory: 256Mi }
  processor:
    resources:
      requests: { cpu: 200m, memory: 256Mi }
      limits: { cpu: 1000m, memory: 512Mi }
  api:
    resources:
      requests: { cpu: 100m, memory: 128Mi }
      limits: { cpu: 500m, memory: 256Mi }
```

#### Recommended (Production)

See [charts/observer/values-production.yaml](charts/observer/values-production.yaml) for full configuration.

### Persistence

#### AIO Mode

```yaml
aio:
  persistence:
    enabled: true
    size: 10Gi
    storageClass: "standard"  # or your preferred storage class
```

#### Distributed Mode

```yaml
mongodb:
  persistence:
    enabled: true
    size: 50Gi
    storageClass: "fast-ssd"

nats:
  config:
    jetstream:
      enabled: true
      fileStore:
        pvc:
          size: 50Gi
          storageClass: "fast-ssd"
```

### Scaling

#### Manual Scaling

```yaml
distributed:
  ingestion:
    replicaCount: 5
  processor:
    replicaCount: 3
  api:
    replicaCount: 3
```

#### Auto-scaling (HPA)

```yaml
distributed:
  ingestion:
    autoscaling:
      enabled: true
      minReplicas: 3
      maxReplicas: 10
      targetCPUUtilizationPercentage: 70
      targetMemoryUtilizationPercentage: 80
```

### Ingress Configuration

#### Basic Ingress

```yaml
ingress:
  enabled: true
  className: nginx
  hosts:
    - host: observer.example.com
      paths:
        - path: /
          pathType: Prefix
          service: web
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

### Using External Database

```bash
helm install observer oci://ghcr.io/stanterprise/observer/charts/observer \
  --version 0.1.0 \
  --namespace observer \
  --set mongodb.enabled=false \
  --set externalDatabase.host=mongodb.example.com \
  --set externalDatabase.port=27017 \
  --set externalDatabase.username=observer \
  --set externalDatabase.password=securepassword \
  --set externalDatabase.database=observer \
  --set externalDatabase.authSource=admin
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
