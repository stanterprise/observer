# Observer Helm Chart

This Helm chart deploys the Observer test observability system on Kubernetes.

## Prerequisites

- Kubernetes 1.23+
- Helm 3.8+
- PV provisioner support in the underlying infrastructure (for persistent storage)

## Installation

### Add Helm Dependencies

Before installing, you need to add the required Helm repositories and update dependencies:

```bash
cd charts/observer
helm dependency update
```

### Install with Default Configuration (Distributed Mode)

```bash
helm install observer ./charts/observer
```

### Install in AIO (All-in-One) Mode

```bash
helm install observer ./charts/observer --set mode=aio --set aio.enabled=true --set distributed.enabled=false
```

### Install from OCI Registry (GitHub Container Registry)

Once published, you can install directly from the registry:

```bash
helm install observer oci://ghcr.io/stanterprise/charts/observer --version 0.1.0
```

## Configuration

### Deployment Modes

The chart supports two deployment modes:

1. **AIO (All-in-One)**: Single container with all services embedded
   - Best for: Development, testing, small-scale deployments

- Includes: Embedded MongoDB, NATS, all services in one pod

2. **Distributed**: Multi-container deployment with separate services
   - Best for: Production, CI/CD, high-scale deployments
   - Includes: Separate pods for ingestion, processor, API, web UI

- Requires: MongoDB and NATS (can be embedded or external)

### Key Configuration Parameters

| Parameter          | Description                             | Default                 |
| ------------------ | --------------------------------------- | ----------------------- |
| `mode`             | Deployment mode: `aio` or `distributed` | `distributed`           |
| `image.registry`   | Docker registry for images              | `ghcr.io`               |
| `image.repository` | Repository path for images              | `stanterprise/observer` |
| `image.tag`        | Image tag (overrides appVersion)        | `""`                    |
| `image.pullPolicy` | Image pull policy                       | `IfNotPresent`          |

#### AIO Mode Parameters

| Parameter                       | Description               | Default |
| ------------------------------- | ------------------------- | ------- |
| `aio.enabled`                   | Enable AIO mode           | `false` |
| `aio.replicaCount`              | Number of replicas        | `1`     |
| `aio.persistence.enabled`       | Enable persistent storage | `true`  |
| `aio.persistence.size`          | Storage size              | `10Gi`  |
| `aio.resources.requests.cpu`    | CPU request               | `500m`  |
| `aio.resources.requests.memory` | Memory request            | `512Mi` |

#### Distributed Mode Parameters

| Parameter                            | Description             | Default |
| ------------------------------------ | ----------------------- | ------- |
| `distributed.enabled`                | Enable distributed mode | `true`  |
| `distributed.ingestion.replicaCount` | Ingestion replicas      | `2`     |
| `distributed.processor.replicaCount` | Processor replicas      | `2`     |
| `distributed.api.replicaCount`       | API replicas            | `2`     |
| `distributed.web.replicaCount`       | Web UI replicas         | `2`     |

#### Database Parameters

| Parameter                   | Description                                   | Default    |
| --------------------------- | --------------------------------------------- | ---------- |
| `mongodb.enabled`           | Deploy MongoDB                                | `true`     |
| `mongodb.auth.rootUser`     | MongoDB root username                         | `root`     |
| `mongodb.auth.rootPassword` | MongoDB root password                         | `password` |
| `externalDatabase.host`     | External DB host (when mongodb.enabled=false) | `""`       |

#### NATS Parameters

| Parameter                       | Description                                 | Default |
| ------------------------------- | ------------------------------------------- | ------- |
| `nats.enabled`                  | Deploy NATS                                 | `true`  |
| `nats.config.jetstream.enabled` | Enable JetStream                            | `true`  |
| `externalNats.url`              | External NATS URL (when nats.enabled=false) | `""`    |

#### Ingress Parameters

The chart provides configurable ingresses for each service:

| Parameter                    | Description            | Default                     |
| ---------------------------- | ---------------------- | --------------------------- |
| `ingress.web.enabled`        | Enable Web UI ingress  | `false`                     |
| `ingress.web.className`      | Web ingress class      | `nginx`                     |
| `ingress.web.hosts[0].host`  | Web UI hostname        | `observer.example.com`      |
| `ingress.web.tls`            | Web TLS configuration  | `[]`                        |
| `ingress.api.enabled`        | Enable API ingress     | `false`                     |
| `ingress.api.className`      | API ingress class      | `nginx`                     |
| `ingress.api.hosts[0].host`  | API hostname           | `api.observer.example.com`  |
| `ingress.api.tls`            | API TLS configuration  | `[]`                        |
| `ingress.grpc.enabled`       | Enable gRPC ingress    | `false`                     |
| `ingress.grpc.className`     | gRPC ingress class     | `nginx`                     |
| `ingress.grpc.hosts[0].host` | gRPC hostname          | `grpc.observer.example.com` |
| `ingress.grpc.tls`           | gRPC TLS configuration | `[]`                        |

### Example Configurations

#### Production Distributed Deployment with External Database

```yaml
mode: distributed
distributed:
  enabled: true
  ingestion:
    replicaCount: 3
    autoscaling:
      enabled: true
      minReplicas: 3
      maxReplicas: 10
  processor:
    replicaCount: 3
    autoscaling:
      enabled: true
  api:
    replicaCount: 2
    autoscaling:
      enabled: true

mongodb:
  enabled: false

externalDatabase:
  host: "mongo.example.com"
  port: 27017
  username: "observer"
  password: "secure-password"
  database: "observer"
  authSource: admin

nats:
  enabled: true
  config:
    jetstream:
      enabled: true
      fileStore:
        pvc:
          size: 50Gi

# Configurable ingresses for each service
ingress:
  web:
    enabled: true
    className: nginx
    annotations:
      cert-manager.io/cluster-issuer: "letsencrypt-prod"
    hosts:
      - host: observer.example.com
        paths:
          - path: /
            pathType: Prefix
    tls:
      - secretName: observer-web-tls
        hosts:
          - observer.example.com
  api:
    enabled: true
    className: nginx
    annotations:
      cert-manager.io/cluster-issuer: "letsencrypt-prod"
    hosts:
      - host: api.observer.example.com
        paths:
          - path: /
            pathType: Prefix
    tls:
      - secretName: observer-api-tls
        hosts:
          - api.observer.example.com
  grpc:
    enabled: true
    className: nginx
    annotations:
      nginx.ingress.kubernetes.io/backend-protocol: "GRPC"
      cert-manager.io/cluster-issuer: "letsencrypt-prod"
    hosts:
      - host: grpc.observer.example.com
        paths:
          - path: /
            pathType: Prefix
    tls:
      - secretName: observer-grpc-tls
        hosts:
          - grpc.observer.example.com
```

#### Development AIO Deployment

```yaml
mode: aio
aio:
  enabled: true
  replicaCount: 1
  persistence:
    enabled: true
    size: 5Gi
  resources:
    requests:
      cpu: 200m
      memory: 256Mi
    limits:
      cpu: 1000m
      memory: 1Gi

distributed:
  enabled: false

mongodb:
  enabled: false

nats:
  enabled: false
```

## Upgrading

```bash
helm upgrade observer ./charts/observer
```

## Uninstalling

```bash
helm uninstall observer
```

This will remove all Kubernetes resources associated with the chart and delete the release.

## Testing the Chart

You can test the chart installation in dry-run mode:

```bash
helm install observer ./charts/observer --dry-run --debug
```

Or use Helm's template command to see the rendered templates:

```bash
helm template observer ./charts/observer
```

## Accessing the Application

After installation, follow the instructions in the NOTES output to access the application:

```bash
helm status observer
```

### Port Forwarding (for ClusterIP services)

```bash
# Web UI
kubectl port-forward svc/observer-web 8080:80

# gRPC Ingestion
kubectl port-forward svc/observer-ingestion 50051:50051

# API
kubectl port-forward svc/observer-api 8080:8080
```

## Architecture

The Observer system consists of:

- **Ingestion Service**: gRPC endpoint for receiving test events
- **Processor Service**: Consumes events from NATS and persists to database
- **API Service**: REST/GraphQL API and WebSocket for real-time updates
- **Web UI**: React-based web interface
- **MongoDB**: Database for storing test results (optional, can use external)
- **NATS**: Message broker for event streaming (optional, can use external)

## Support

For issues and questions, please visit: https://github.com/stanterprise/observer

## License

See the main repository for license information.
