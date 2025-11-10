# Helm Chart Overview

Helm enables scalable distributed deployments on Kubernetes.

## Structure

```text
charts/test-observer/
  values.yaml
  templates/
    nats.yaml
    postgres.yaml
    minio.yaml
    ingestion.yaml
    processor.yaml
    api.yaml
    ingress.yaml
```

## Important Values

| Key | Description |
|-----|-------------|
| `ingestion.replicas` | gRPC service scaling |
| `processor.replicas` | event consumer scaling |
| `api.replicas` | API replica count |
| `db.external` | external DB connection |
| `s3.*` | object store credentials |
| `auth.oidc.*` | OIDC config |

## Typical Workflow

1. `helm install observer ./charts/test-observer`  
2. Configure external DB and S3  
3. Reporters send gRPC traffic to service endpoint  
4. Access UI via ingress
