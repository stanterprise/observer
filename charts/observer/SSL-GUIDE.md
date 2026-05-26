# Observer Helm Chart - SSL/HTTPS Configuration Guide

## Industry Standard Practices Implemented

### 1. **Image Tag Management** ✅

The chart follows semantic versioning and immutable tag best practices:

```yaml
# Chart.yaml
appVersion: "0.1.0"  # Default version for production

# values.yaml
image:
  tag: ""  # Empty = use Chart.appVersion
  pullPolicy: IfNotPresent  # Overridable

# values-aio.yaml (environment-specific override)
image:
  tag: "sha-3f1c0e4"  # Immutable SHA tag for this deployment
```

**Auto-detected Pull Policy:**

- `latest`, `main`, `develop` tags → `Always` pull
- SHA tags, semantic versions → `IfNotPresent`
- Manual override available via `image.pullPolicy`

### 2. **SSL/TLS Certificate Management** ✅

**Google-Managed Certificates (Recommended for GKE):**

```yaml
ingress:
  managedCertificate:
    enabled: true # Auto-provisions Let's Encrypt certs via Google
```

**Cert-Manager (Kubernetes-native, cloud-agnostic):**

```yaml
ingress:
  managedCertificate:
    enabled: false
  web:
    annotations:
      cert-manager.io/cluster-issuer: "letsencrypt-prod"
    tls:
      - secretName: observer-web-tls
        hosts:
          - observer.example.com
```

### 3. **HTTPS Redirect** ✅

Enable after certificates are provisioned:

```yaml
ingress:
  ssl:
    redirect: true # HTTP → HTTPS (301 permanent redirect)
    policy: "MODERN" # TLS 1.2+, strong ciphers
```

**SSL Policies (GCP):**

- `MODERN`: TLS 1.2+, recommended for most use cases
- `COMPATIBLE`: TLS 1.0+, for legacy clients
- `RESTRICTED`: TLS 1.2+, most restrictive

### 4. **Health Checks** ✅

```yaml
# GCP BackendConfig for gRPC
ingress:
  grpc:
    backendConfig:
      enabled: true
      healthCheck:
        type: HTTP2
        checkIntervalSec: 10
        timeoutSec: 5
```

## Deployment Workflow

### Initial Deployment (HTTP only)

```bash
# 1. Deploy with managed certificates enabled, redirect disabled
helm install observer charts/observer/ \
  --namespace observer-prod \
  --create-namespace \
  --values charts/observer/values-aio.yaml

# 2. Verify DNS is configured (required for cert provisioning)
kubectl get ingress -n observer-prod
# Copy IP addresses, add DNS A records

# 3. Monitor certificate provisioning (15-60 min)
kubectl describe managedcertificate observer-cert -n observer-prod
# Wait for Status: Active
```

### Enable HTTPS Redirect (After Certs Active)

```bash
# 1. Update values-aio.yaml
# ingress.ssl.redirect: true

# 2. Upgrade deployment
helm upgrade observer charts/observer/ \
  --namespace observer-prod \
  --values charts/observer/values-aio.yaml

# 3. Test HTTPS
curl -I https://observer.example.com
```

## Multi-Environment Strategy

### Development

```yaml
# values-dev.yaml
image:
  tag: "main" # Always pull latest
ingress:
  managedCertificate:
    enabled: false # Use HTTP for dev
```

### Staging

```yaml
# values-staging.yaml
image:
  tag: "sha-abc123" # Specific build
ingress:
  managedCertificate:
    enabled: true
  ssl:
    redirect: true # Test HTTPS redirect
```

### Production

```yaml
# values-production.yaml
image:
  tag: "v1.2.3" # Semantic version
ingress:
  managedCertificate:
    enabled: true
  ssl:
    redirect: true
    policy: "MODERN"
```

## Troubleshooting

### Certificate Not Provisioning

```bash
# Check certificate status
kubectl describe managedcertificate observer-cert -n observer-prod

# Common issues:
# 1. DNS not pointing to load balancer
kubectl get ingress -n observer-prod -o wide
# Verify DNS A records match ADDRESS column

# 2. Domain ownership verification failing
# Ensure ingress is accessible via HTTP (port 80)
curl -I http://observer.example.com
```

### HTTPS Redirect Not Working

```bash
# Check FrontendConfig was created
kubectl get frontendconfig -n observer-prod

# Check ingress annotations
kubectl get ingress observer-web -n observer-prod -o yaml | grep -A 5 annotations
```

## Alternative: Cert-Manager

For multi-cloud or non-GCP deployments:

```bash
# Install cert-manager
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml

# Create ClusterIssuer
cat <<EOF | kubectl apply -f -
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: ops@example.com
    privateKeySecretRef:
      name: letsencrypt-prod
    solvers:
    - http01:
        ingress:
          class: nginx
EOF

# Deploy with cert-manager
helm install observer charts/observer/ \
  --set ingress.managedCertificate.enabled=false \
  --set ingress.web.annotations."cert-manager\.io/cluster-issuer"=letsencrypt-prod \
  --set-string ingress.web.tls[0].secretName=observer-web-tls \
  --set-string ingress.web.tls[0].hosts[0]=observer.example.com
```

## Security Best Practices

1. **Always use HTTPS in production** (`ssl.redirect: true`)
2. **Use immutable image tags** (SHA or semantic versions)
3. **Enable SSL policy** (`ssl.policy: "MODERN"`)
4. **Rotate secrets regularly** (if using custom TLS secrets)
5. **Monitor certificate expiry** (Google-managed auto-renews)
6. **Use separate values files per environment**

## References

- [Google-managed SSL certificates](https://cloud.google.com/kubernetes-engine/docs/how-to/managed-certs)
- [GKE Ingress for HTTPS](https://cloud.google.com/kubernetes-engine/docs/concepts/ingress)
- [Cert-Manager Documentation](https://cert-manager.io/docs/)
- [Helm Best Practices](https://helm.sh/docs/chart_best_practices/)
