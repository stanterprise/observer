#!/bin/bash
# GKE Connection Script for Observer Development
# This script helps configure kubectl to connect to a GKE cluster
set -e

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values (can be overridden by environment variables or .env file)
GKE_PROJECT="${GKE_PROJECT:-}"
GKE_CLUSTER="${GKE_CLUSTER:-observer-dev}"
GKE_REGION="${GKE_REGION:-us-central1}"
GKE_ZONE="${GKE_ZONE:-}"

# Function to print colored output
print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to show usage
usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Configure kubectl to connect to a GKE cluster for Observer development."
    echo ""
    echo "Options:"
    echo "  -p, --project PROJECT    GCP project ID (or set GKE_PROJECT env var)"
    echo "  -c, --cluster CLUSTER    GKE cluster name (default: observer-dev)"
    echo "  -r, --region REGION      GKE region (default: us-central1)"
    echo "  -z, --zone ZONE          GKE zone (use instead of region for zonal clusters)"
    echo "  -n, --namespace NS       Set default namespace after connecting"
    echo "  -h, --help               Show this help message"
    echo ""
    echo "Environment variables:"
    echo "  GKE_PROJECT              GCP project ID"
    echo "  GKE_CLUSTER              GKE cluster name"
    echo "  GKE_REGION               GKE region (for regional clusters)"
    echo "  GKE_ZONE                 GKE zone (for zonal clusters)"
    echo ""
    echo "Examples:"
    echo "  # Connect using environment variables"
    echo "  export GKE_PROJECT=my-project"
    echo "  $0"
    echo ""
    echo "  # Connect with command-line options"
    echo "  $0 --project my-project --cluster observer-dev --region us-central1"
    echo ""
    echo "  # Connect to a zonal cluster"
    echo "  $0 --project my-project --cluster observer-dev --zone us-central1-a"
    exit 0
}

# Parse command-line arguments
NAMESPACE=""
while [[ $# -gt 0 ]]; do
    case $1 in
        -p|--project)
            GKE_PROJECT="$2"
            shift 2
            ;;
        -c|--cluster)
            GKE_CLUSTER="$2"
            shift 2
            ;;
        -r|--region)
            GKE_REGION="$2"
            shift 2
            ;;
        -z|--zone)
            GKE_ZONE="$2"
            shift 2
            ;;
        -n|--namespace)
            NAMESPACE="$2"
            shift 2
            ;;
        -h|--help)
            usage
            ;;
        *)
            print_error "Unknown option: $1"
            usage
            ;;
    esac
done

# Determine script directory
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Load .env file if it exists (only export valid KEY=value lines)
ENV_FILE="${SCRIPT_DIR}/../.env"
if [ -f "$ENV_FILE" ]; then
    print_info "Loading environment from .env file..."
    # Only export lines that look like valid variable assignments (KEY=value)
    while IFS='=' read -r key value; do
        # Skip comments and empty lines
        [[ -z "$key" || "$key" =~ ^[[:space:]]*# ]] && continue
        # Remove leading/trailing whitespace from key
        key=$(echo "$key" | xargs)
        # Only export if key is a valid variable name and not already set
        if [[ "$key" =~ ^[A-Za-z_][A-Za-z0-9_]*$ ]] && [ -z "${!key:-}" ]; then
            export "$key=$value"
        fi
    done < "$ENV_FILE"
fi

echo ""
echo "🔧 GKE Connection Configuration for Observer Development"
echo "========================================================="
echo ""

# Check prerequisites
print_info "Checking prerequisites..."

# Check for gcloud CLI
if ! command_exists gcloud; then
    print_error "gcloud CLI is not installed."
    echo ""
    echo "Install gcloud CLI:"
    echo "  - macOS: brew install google-cloud-sdk"
    echo "  - Linux: curl https://sdk.cloud.google.com | bash"
    echo "  - Windows: Download from https://cloud.google.com/sdk/docs/install"
    exit 1
fi

# Check for kubectl
if ! command_exists kubectl; then
    print_error "kubectl is not installed."
    echo ""
    echo "Install kubectl:"
    echo "  - gcloud: gcloud components install kubectl"
    echo "  - macOS: brew install kubectl"
    echo "  - Linux: See https://kubernetes.io/docs/tasks/tools/install-kubectl-linux/"
    exit 1
fi

# Get tool versions (with simple fallbacks)
GCLOUD_VERSION=$(gcloud version --format='value(Google Cloud SDK)' 2>/dev/null || echo 'version unknown')
KUBECTL_VERSION=$(kubectl version --client -o yaml 2>/dev/null | grep gitVersion | head -1 | cut -d: -f2 | tr -d ' ' || echo 'version unknown')

print_success "gcloud CLI found: $GCLOUD_VERSION"
print_success "kubectl found: $KUBECTL_VERSION"

# Check if GKE_PROJECT is set
if [ -z "$GKE_PROJECT" ]; then
    # Try to get from gcloud config
    GKE_PROJECT=$(gcloud config get-value project 2>/dev/null || true)
    if [ -z "$GKE_PROJECT" ] || [ "$GKE_PROJECT" = "(unset)" ]; then
        print_error "GCP project not set."
        echo ""
        echo "Set your project using one of these methods:"
        echo "  1. Set GKE_PROJECT environment variable"
        echo "  2. Run: gcloud config set project YOUR_PROJECT_ID"
        echo "  3. Use: $0 --project YOUR_PROJECT_ID"
        exit 1
    fi
fi

echo ""
print_info "Configuration:"
echo "  Project: $GKE_PROJECT"
echo "  Cluster: $GKE_CLUSTER"
if [ -n "$GKE_ZONE" ]; then
    echo "  Zone:    $GKE_ZONE"
else
    echo "  Region:  $GKE_REGION"
fi
echo ""

# Check gcloud authentication
print_info "Checking gcloud authentication..."
if ! gcloud auth list --filter=status:ACTIVE --format="value(account)" | head -n1 | grep -q .; then
    print_warning "Not logged in to gcloud. Starting authentication..."
    gcloud auth login
fi
ACTIVE_ACCOUNT=$(gcloud auth list --filter=status:ACTIVE --format="value(account)" | head -n1)
print_success "Authenticated as: $ACTIVE_ACCOUNT"

# Set the project
print_info "Setting GCP project..."
gcloud config set project "$GKE_PROJECT" >/dev/null 2>&1
print_success "Project set to: $GKE_PROJECT"

# Get cluster credentials
print_info "Fetching cluster credentials..."
if [ -n "$GKE_ZONE" ]; then
    # Zonal cluster
    if ! gcloud container clusters get-credentials "$GKE_CLUSTER" --zone "$GKE_ZONE" --project "$GKE_PROJECT"; then
        print_error "Failed to get credentials for cluster $GKE_CLUSTER in zone $GKE_ZONE"
        exit 1
    fi
else
    # Regional cluster
    if ! gcloud container clusters get-credentials "$GKE_CLUSTER" --region "$GKE_REGION" --project "$GKE_PROJECT"; then
        print_error "Failed to get credentials for cluster $GKE_CLUSTER in region $GKE_REGION"
        exit 1
    fi
fi
print_success "Cluster credentials configured"

# Set namespace if specified
if [ -n "$NAMESPACE" ]; then
    print_info "Setting default namespace to: $NAMESPACE"
    kubectl config set-context --current --namespace="$NAMESPACE"
    print_success "Namespace set to: $NAMESPACE"
fi

# Verify connection
print_info "Verifying cluster connection..."
echo ""

if kubectl cluster-info >/dev/null 2>&1; then
    print_success "Connected to cluster successfully!"
    echo ""
    kubectl cluster-info
    echo ""

    # Show current context
    CURRENT_CONTEXT=$(kubectl config current-context)
    print_info "Current kubectl context: $CURRENT_CONTEXT"

    # Show namespaces (limited)
    echo ""
    print_info "Available namespaces:"
    kubectl get namespaces --no-headers | head -10
    NS_COUNT=$(kubectl get namespaces --no-headers | wc -l)
    if [ "$NS_COUNT" -gt 10 ]; then
        echo "  ... and $((NS_COUNT - 10)) more"
    fi

    echo ""
    echo "========================================================="
    print_success "GKE connection configured successfully!"
    echo ""
    echo "Quick commands:"
    echo "  kubectl get pods                    # List pods in current namespace"
    echo "  kubectl get pods -n observer        # List pods in observer namespace"
    echo "  kubectl logs -f <pod-name>          # Follow pod logs"
    echo "  helm list                           # List Helm releases"
    echo "  make helm-dry-run                   # Test Helm chart"
    echo ""
    echo "To deploy Observer to this cluster:"
    echo "  helm install observer ./charts/observer"
    echo ""
else
    print_error "Failed to connect to cluster"
    echo "Please check your credentials and network connectivity."
    exit 1
fi
