#!/bin/bash
set -e

# Deploy Observer with HTTPS support
# Usage: ./scripts/deploy-https.sh [image-tag]

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
NAMESPACE="${NAMESPACE:-observer-test}"
IMAGE_TAG="${1:-latest}"

echo "🚀 Deploying Observer with HTTPS support"
echo "   Namespace: $NAMESPACE"
echo "   Image tag: $IMAGE_TAG"
echo ""

# Check prerequisites
echo "📋 Checking prerequisites..."

# Check if nginx-ingress is installed
if ! kubectl get namespace ingress-nginx &>/dev/null; then
    echo "⚙️  Installing nginx-ingress-controller..."
    helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx 2>/dev/null || true
    helm repo update

    helm install ingress-nginx ingress-nginx/ingress-nginx \
        --namespace ingress-nginx \
        --create-namespace \
        --set controller.service.type=LoadBalancer \
        --set controller.service.externalTrafficPolicy=Local \
        --wait

    echo "✅ nginx-ingress-controller installed"
else
    echo "✅ nginx-ingress-controller already installed"
fi

# Check if cert-manager is installed
if ! kubectl get namespace cert-manager &>/dev/null; then
    echo "⚙️  Installing cert-manager..."
    helm repo add jetstack https://charts.jetstack.io 2>/dev/null || true
    helm repo update

    helm install cert-manager jetstack/cert-manager \
        --namespace cert-manager \
        --create-namespace \
        --set crds.enabled=true \
        --set global.leaderElection.namespace=cert-manager \
        --set startupapicheck.enabled=false \
        --wait

    echo "✅ cert-manager installed"
else
    echo "✅ cert-manager already installed"
fi

# Wait for cert-manager to be ready
echo "⏳ Waiting for cert-manager to be ready..."
kubectl wait --for=condition=available --timeout=120s deployment/cert-manager -n cert-manager
kubectl wait --for=condition=available --timeout=120s deployment/cert-manager-webhook -n cert-manager
sleep 5

# Create Let's Encrypt ClusterIssuer if it doesn't exist
if ! kubectl get clusterissuer letsencrypt-prod &>/dev/null; then
    echo "⚙️  Creating Let's Encrypt ClusterIssuer..."
    cat <<EOF | kubectl apply -f -
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: admin@stanterprise.dev
    privateKeySecretRef:
      name: letsencrypt-prod
    solvers:
    - http01:
        ingress:
          class: nginx
EOF
    echo "✅ Let's Encrypt ClusterIssuer created"
else
    echo "✅ Let's Encrypt ClusterIssuer already exists"
fi

# Get nginx-ingress external IP
echo "⏳ Getting nginx-ingress LoadBalancer IP..."
INGRESS_IP=""
for i in {1..30}; do
    INGRESS_IP=$(kubectl get svc ingress-nginx-controller -n ingress-nginx -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "")
    if [ -n "$INGRESS_IP" ]; then
        break
    fi
    echo "   Waiting for LoadBalancer IP... (attempt $i/30)"
    sleep 5
done

if [ -z "$INGRESS_IP" ]; then
    echo "❌ Failed to get nginx-ingress LoadBalancer IP"
    exit 1
fi

echo "✅ nginx-ingress IP: $INGRESS_IP"

# Update DNS records
echo "⚙️  Updating DNS records..."
gcloud dns record-sets update observer.stanterprise.dev. \
    --type=A \
    --ttl=300 \
    --rrdatas="$INGRESS_IP" \
    --zone=stanterprise-dev 2>/dev/null || echo "   ⚠️  observer.stanterprise.dev DNS update failed (may already be correct)"

gcloud dns record-sets update api.observer.stanterprise.dev. \
    --type=A \
    --ttl=300 \
    --rrdatas="$INGRESS_IP" \
    --zone=stanterprise-dev 2>/dev/null || echo "   ⚠️  api.observer.stanterprise.dev DNS update failed (may already be correct)"

echo "✅ DNS records updated to point to $INGRESS_IP"

# Deploy/upgrade Observer with Helm
echo "⚙️  Deploying Observer..."
helm upgrade observer "$PROJECT_ROOT/charts/observer" \
    --install \
    --namespace "$NAMESPACE" \
    --create-namespace \
    --set image.tag="$IMAGE_TAG" \
    --set ingress.managedCertificate.enabled=false \
    --set ingress.web.enabled=true \
    --set ingress.web.className=nginx \
    --set ingress.web.annotations."cert-manager\.io/cluster-issuer"=letsencrypt-prod \
    --set ingress.api.enabled=true \
    --set ingress.api.className=nginx \
    --set ingress.api.annotations."cert-manager\.io/cluster-issuer"=letsencrypt-prod \
    --set ingress.grpc.enabled=false \
    --set ingress.web.hosts[0].host=observer.stanterprise.dev \
    --set ingress.web.hosts[0].paths[0].path=/ \
    --set ingress.web.hosts[0].paths[0].pathType=Prefix \
    --set ingress.api.hosts[0].host=api.observer.stanterprise.dev \
    --set ingress.api.hosts[0].paths[0].path=/ \
    --set ingress.api.hosts[0].paths[0].pathType=Prefix \
    --set ingress.web.tls[0].secretName=observer-web-tls \
    --set ingress.web.tls[0].hosts[0]=observer.stanterprise.dev \
    --set ingress.api.tls[0].secretName=observer-api-tls \
    --set ingress.api.tls[0].hosts[0]=api.observer.stanterprise.dev \
    --wait

echo "✅ Observer deployed"

# Wait for certificates to be issued
echo "⏳ Waiting for SSL certificates to be issued..."
for i in {1..60}; do
    WEB_READY=$(kubectl get certificate observer-web-tls -n "$NAMESPACE" -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null || echo "False")
    API_READY=$(kubectl get certificate observer-api-tls -n "$NAMESPACE" -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null || echo "False")

    if [ "$WEB_READY" = "True" ] && [ "$API_READY" = "True" ]; then
        echo "✅ SSL certificates issued successfully"
        break
    fi

    if [ $i -eq 60 ]; then
        echo "⚠️  Certificate issuance taking longer than expected"
        echo "   Check status with: kubectl get certificate -n $NAMESPACE"
        break
    fi

    echo "   Waiting for certificates... (attempt $i/60)"
    sleep 5
done

# Display deployment information
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ Deployment Complete!"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "🌐 Web UI:  https://observer.stanterprise.dev"
echo "🔌 API:     https://api.observer.stanterprise.dev"
echo ""
echo "📦 Image:   ghcr.io/stanterprise/observer:$IMAGE_TAG"
echo "🔐 SSL:     Let's Encrypt (auto-renewal enabled)"
echo ""
echo "📋 Useful commands:"
echo "   kubectl get pods -n $NAMESPACE"
echo "   kubectl logs -n $NAMESPACE -l app.kubernetes.io/component=aio -f"
echo "   kubectl get certificate -n $NAMESPACE"
echo "   kubectl get ingress -n $NAMESPACE"
echo ""
