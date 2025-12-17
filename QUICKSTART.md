# Observer Quick Start Guide

Get Observer up and running in minutes!

## Choose Your Deployment Method

### Option 1: Docker (Fastest - 2 minutes)

For local development and testing.

```bash
# Pull and run the All-in-One image
docker run -d \
  --name observer \
  -p 3000:80 \
  -p 50051:50051 \
  -v observer-data:/data \
  ghcr.io/stanterprise/observer/aio:latest

# Access:
# - Web UI: http://localhost:3000
# - gRPC endpoint: localhost:50051
```

### Option 2: Docker Compose (5 minutes)

For distributed architecture locally.

```bash
# Clone the repository
git clone https://github.com/stanterprise/observer.git
cd observer

# Update docker-compose.yml to use published images
sed -i 's|image: observer:|image: ghcr.io/stanterprise/observer/|g' docker-compose.yml

# Start in distributed mode
docker compose --profile dist up -d

# Access:
# - Web UI: http://localhost:3000
# - gRPC endpoint: localhost:50051
```

### Option 3: Kubernetes with Helm (10 minutes)

For production deployment.

#### AIO Mode (Simple)

```bash
# Install in AIO mode
helm install observer oci://ghcr.io/stanterprise/observer/charts/observer \
  --version 0.1.0 \
  --set mode=aio \
  --set aio.enabled=true \
  --set distributed.enabled=false \
  --set mongodb.enabled=false \
  --set nats.enabled=false

# Wait for pod to be ready
kubectl wait --for=condition=ready pod -l app.kubernetes.io/component=aio --timeout=300s

# Access the Web UI
kubectl port-forward svc/observer-aio 3000:80

# Access gRPC endpoint
kubectl port-forward svc/observer-aio 50051:50051
```

#### Distributed Mode (Production)

```bash
# Install in distributed mode with embedded MongoDB and NATS
helm install observer oci://ghcr.io/stanterprise/observer/charts/observer \
  --version 0.1.0

# Wait for all pods to be ready
kubectl wait --for=condition=ready pod -l app.kubernetes.io/instance=observer --timeout=300s

# Access the Web UI
kubectl port-forward svc/observer-web 3000:80

# Access gRPC endpoint
kubectl port-forward svc/observer-ingestion 50051:50051
```

## Verify Installation

### Check Web UI

Open your browser and navigate to the Web UI:

- Docker/Compose: http://localhost:3000
- Kubernetes: http://localhost:3000 (after port-forward)

You should see the Observer dashboard.

### Send Test Event

You can test the gRPC endpoint using the Playwright reporter or any gRPC client.

Using grpcurl (if installed):

```bash
# List services
grpcurl -plaintext localhost:50051 list

# Send a test event (example - adjust based on your protobuf schema)
grpcurl -plaintext -d '{
  "test_case": {
    "id": "test-123",
    "name": "Example Test",
    "suite": "QuickStart"
  }
}' localhost:50051 testsystem.v1.TestSystemService/ReportTestBegin
```

## Next Steps

### Configure Test Reporters

Observer works with the Playwright custom reporter. Install it in your test project:

```bash
npm install github:stanterprise/stanterprise-playwright-reporter
```

Configure in `playwright.config.ts`:

```typescript
import { defineConfig } from "@playwright/test";

export default defineConfig({
  reporter: [
    ["list"],
    [
      "github:stanterprise/stanterprise-playwright-reporter",
      {
        endpoint: "localhost:50051", // or your Kubernetes endpoint
      },
    ],
  ],
});
```

### Production Deployment

For production use, see:

- [DEPLOYMENT.md](DEPLOYMENT.md) - Comprehensive deployment guide
- [charts/observer/README.md](charts/observer/README.md) - Helm chart documentation
- [charts/observer/values-production.yaml](charts/observer/values-production.yaml) - Production configuration example

### Scaling

#### Docker Compose

Edit `docker-compose.yml` and use:

```yaml
services:
  ingestion:
    deploy:
      replicas: 3
```

Then:

```bash
docker compose --profile dist up -d --scale ingestion=3
```

#### Kubernetes

Enable auto-scaling:

```bash
helm upgrade observer oci://ghcr.io/stanterprise/observer/charts/observer \
  --version 0.1.0 \
  --set distributed.ingestion.autoscaling.enabled=true \
  --set distributed.ingestion.autoscaling.minReplicas=3 \
  --set distributed.ingestion.autoscaling.maxReplicas=10
```

Or manually scale:

```bash
kubectl scale deployment observer-ingestion --replicas=5
```

## Troubleshooting

### Container/Pod Not Starting

```bash
# Docker
docker logs observer

# Docker Compose
docker compose logs -f

# Kubernetes
kubectl logs -l app.kubernetes.io/instance=observer
kubectl describe pod <pod-name>
```

### Can't Access Web UI

1. Check if service is running:

   ```bash
   # Docker
   docker ps | grep observer

   # Kubernetes
   kubectl get pods -l app.kubernetes.io/instance=observer
   ```

2. Verify port forwarding (Kubernetes):

   ```bash
   kubectl port-forward svc/observer-web 3000:80
   # or for AIO
   kubectl port-forward svc/observer-aio 3000:80
   ```

3. Check firewall rules and network policies

### Database Connection Issues

For distributed deployments:

```bash
# Check MongoDB
kubectl logs -l app.kubernetes.io/name=mongodb

# Check processor logs (connects to database)
kubectl logs -l app.kubernetes.io/component=processor
```

### NATS Connection Issues

```bash
# Check NATS
kubectl logs -l app.kubernetes.io/name=nats

# Check ingestion logs (publishes to NATS)
kubectl logs -l app.kubernetes.io/component=ingestion
```

## Clean Up

### Docker

```bash
docker stop observer
docker rm observer
docker volume rm observer-data
```

### Docker Compose

```bash
docker compose --profile dist down -v
```

### Kubernetes

```bash
helm uninstall observer

# Delete PVCs (optional - this deletes data!)
kubectl delete pvc -l app.kubernetes.io/instance=observer
```

## Get Help

- 📖 [Full Documentation](README.md)
- 🚀 [Deployment Guide](DEPLOYMENT.md)
- ⎈ [Helm Chart README](charts/observer/README.md)
- 🐛 [Report Issues](https://github.com/stanterprise/observer/issues)
