# GKE Development Guide

This guide explains how to connect to and use a GKE (Google Kubernetes Engine) cluster for Observer development.

## Prerequisites

1. **Google Cloud SDK (gcloud)** - Install from [cloud.google.com/sdk](https://cloud.google.com/sdk/docs/install)
2. **kubectl** - Usually installed with gcloud: `gcloud components install kubectl`
3. **Helm 3.x** - Install from [helm.sh](https://helm.sh/docs/intro/install/)
4. **GKE Auth Plugin** - Install with: `gcloud components install gke-gcloud-auth-plugin`

If you're using the devcontainer, these tools are pre-installed.

## Quick Start

### 1. Configure GKE Connection

Set your GKE cluster details in your environment or `.env` file:

```bash
# Copy .env.example to .env
cp .env.example .env

# Edit .env and set:
GKE_PROJECT=your-gcp-project-id
GKE_CLUSTER=observer-dev
GKE_REGION=us-central1
```

### 2. Connect to the Cluster

```bash
# Using the connection script
make gke-connect

# Or with the observer namespace set as default
make gke-connect-ns
```

### 3. Deploy Observer

```bash
# Deploy with default settings
make gke-deploy

# Deploy in AIO mode (single pod)
make gke-deploy-aio

# Deploy with production settings
make gke-deploy-prod
```

### 4. Access Services

```bash
# Start port-forwarding to access services locally
make gke-port-forward

# Access:
# - Web UI: http://localhost:3000
# - gRPC: localhost:50051
# - API: http://localhost:8080
```

## Manual Connection

If you prefer to connect manually without the helper script:

```bash
# Authenticate with Google Cloud
gcloud auth login

# Set your project
gcloud config set project YOUR_PROJECT_ID

# Get cluster credentials (regional cluster)
gcloud container clusters get-credentials CLUSTER_NAME --region REGION

# Or for zonal cluster
gcloud container clusters get-credentials CLUSTER_NAME --zone ZONE
```

## Makefile Targets

| Target | Description |
|--------|-------------|
| `make gke-connect` | Connect to GKE cluster using configured settings |
| `make gke-connect-ns` | Connect and set default namespace to `observer` |
| `make gke-deploy` | Deploy Observer using default Helm values |
| `make gke-deploy-aio` | Deploy Observer in AIO mode |
| `make gke-deploy-prod` | Deploy Observer with production values |
| `make gke-status` | Show deployment status |
| `make gke-logs` | View logs from Observer pods |
| `make gke-port-forward` | Port-forward services locally |
| `make gke-uninstall` | Uninstall Observer from cluster |

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `GKE_PROJECT` | GCP project ID | (required) |
| `GKE_CLUSTER` | GKE cluster name | `observer-dev` |
| `GKE_REGION` | GKE region (for regional clusters) | `us-central1` |
| `GKE_ZONE` | GKE zone (for zonal clusters) | (empty) |

## Cluster Configuration Examples

### Regional Cluster (Recommended for Production)

```bash
export GKE_PROJECT=my-project
export GKE_CLUSTER=observer-prod
export GKE_REGION=us-central1

make gke-connect
```

### Zonal Cluster

```bash
export GKE_PROJECT=my-project
export GKE_CLUSTER=observer-dev
export GKE_ZONE=us-central1-a

make gke-connect
```

## Development Workflow

### 1. Connect to Development Cluster

```bash
# Set up connection
make gke-connect-ns

# Verify connection
kubectl get nodes
```

### 2. Deploy Your Changes

```bash
# Build images (if needed)
make docker-build-all

# Deploy to cluster
make gke-deploy

# Watch deployment progress
kubectl get pods -w -n observer
```

### 3. Test Your Changes

```bash
# Check deployment status
make gke-status

# View logs
make gke-logs

# Port-forward to test locally
make gke-port-forward

# In another terminal, test the gRPC endpoint
grpcurl -plaintext localhost:50051 list
```

### 4. Debug Issues

```bash
# Describe pods for events
kubectl describe pods -n observer

# Get detailed logs from a specific component
kubectl logs -n observer -l app.kubernetes.io/component=ingestion -f

# Shell into a pod
kubectl exec -it -n observer <pod-name> -- sh
```

### 5. Clean Up

```bash
# Uninstall the release
make gke-uninstall

# Delete PVCs (if you want to delete data)
kubectl delete pvc -n observer -l app.kubernetes.io/instance=observer

# Delete the namespace
kubectl delete namespace observer
```

## Working with Multiple Clusters

kubectl supports multiple contexts. After connecting to a cluster, you can switch between them:

```bash
# List available contexts
kubectl config get-contexts

# Switch context
kubectl config use-context CONTEXT_NAME

# Run a command against a specific context
kubectl --context=OTHER_CONTEXT get pods
```

## Troubleshooting

### Authentication Issues

```bash
# Re-authenticate
gcloud auth login

# Reset application-default credentials
gcloud auth application-default login

# Check current authentication
gcloud auth list
```

### GKE Auth Plugin Issues

If you see errors about `gke-gcloud-auth-plugin`:

```bash
# Install the plugin
gcloud components install gke-gcloud-auth-plugin

# Set the USE_GKE_GCLOUD_AUTH_PLUGIN environment variable
export USE_GKE_GCLOUD_AUTH_PLUGIN=True
```

### Connection Errors

```bash
# Verify cluster exists
gcloud container clusters list --project YOUR_PROJECT

# Check cluster status
gcloud container clusters describe CLUSTER_NAME --region REGION

# Test connectivity
kubectl cluster-info
```

### Permission Issues

Ensure your GCP account has the following IAM roles:
- `roles/container.clusterViewer` (to view clusters)
- `roles/container.developer` (to deploy workloads)
- Or `roles/container.admin` (full cluster access)

## Security Best Practices

1. **Use Workload Identity** for production deployments instead of service account keys
2. **Limit namespace access** - Use RBAC to restrict access to specific namespaces
3. **Use separate clusters** for development and production
4. **Enable audit logging** on production clusters
5. **Rotate credentials** regularly

## Related Documentation

- [Deployment Guide](DEPLOYMENT.md) - Full deployment documentation
- [Helm Chart README](charts/observer/README.md) - Helm chart configuration
- [Quick Start Guide](QUICKSTART.md) - Getting started with Observer
- [GKE Documentation](https://cloud.google.com/kubernetes-engine/docs) - Official GKE docs
