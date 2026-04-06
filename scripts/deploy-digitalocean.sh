#!/bin/bash

# ============================================================================
# Observer DigitalOcean Deployment Script
# ============================================================================
# This script automates the deployment of Observer on a DigitalOcean droplet.
#
# USAGE:
#   ./scripts/deploy-digitalocean.sh [OPTIONS]
#
# OPTIONS:
#   --domain <domain>          Domain name (required)
#   --domain-alias <domain>    Additional domain/SAN (repeatable)
#   --version <tag>            Observer image version to deploy (default: latest)
#   --jwt-secret <secret>      JWT secret (optional, will be generated)
#   --ssl <type>              SSL type: letsencrypt, self-signed, none (default: none)
#   --backup-enabled           Enable automated backups
#   --retain-messages          Enable raw NATS message retention for troubleshooting
#   --email <email>            Email for Let's Encrypt (required if using letsencrypt)
#   --help                     Show this help message
#
# EXAMPLES:
#   # Basic deployment
#   ./scripts/deploy-digitalocean.sh --domain observer.example.com
#
#   # With Let's Encrypt SSL
#   ./scripts/deploy-digitalocean.sh \
#     --domain observer.example.com \
#     --domain-alias www.observer.example.com \
#     --ssl letsencrypt \
#     --email admin@example.com
#
#   # With backups enabled
#   ./scripts/deploy-digitalocean.sh \
#     --domain observer.example.com \
#     --backup-enabled
#
# ============================================================================

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
DOMAIN_NAME=""
DOMAIN_ALIASES=""
TLS_SERVER_NAMES=""
JWT_SECRET="${JWT_SECRET:-}"   # preserve env var if already exported
OBSERVER_VERSION="latest"
SSL_TYPE="none"
BACKUP_ENABLED=false
RETAIN_MESSAGES=false
LE_EMAIL=""
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Functions
print_header() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ $1${NC}"
}

show_help() {
    # Print the header comment (usage) starting from line 3 until the first non-comment line
    awk 'NR>=3 && /^#/ {print; next} NR>3 {exit}' "$0"
    exit 0
}

check_requirements() {
    print_header "Checking Requirements"

    local missing_tools=()

    # Check for Docker
    if ! command -v docker &> /dev/null; then
        missing_tools+=("Docker")
    else
        print_success "Docker installed"
    fi

    # Check for Docker Compose
    if ! docker compose version &> /dev/null; then
        missing_tools+=("Docker Compose")
    else
        print_success "Docker Compose installed"
    fi

    # Check for curl
    if ! command -v curl &> /dev/null; then
        missing_tools+=("curl")
    else
        print_success "curl installed"
    fi

    # Check for openssl
    if ! command -v openssl &> /dev/null; then
        missing_tools+=("openssl")
    else
        print_success "openssl installed"
    fi

    # Check if running as root or with sudo
    if [[ $EUID -ne 0 ]]; then
        missing_tools+=("Root/sudo privileges")
    else
        print_success "Running with root privileges"
    fi

    if [[ ${#missing_tools[@]} -gt 0 ]]; then
        print_error "Missing requirements:"
        for tool in "${missing_tools[@]}"; do
            echo "  - $tool"
        done
        echo ""
        echo "Installation instructions:"
        echo "  Ubuntu/Debian:"
        echo "    sudo apt update"
        echo "    sudo apt install -y docker.io docker-compose-plugin curl openssl"
        echo "    sudo usermod -aG docker \$USER"
        exit 1
    fi

    print_success "All requirements met"
}

validate_domain() {
    if [[ -z "$DOMAIN_NAME" ]]; then
        print_error "Domain name is required"
        exit 1
    fi

    if ! [[ "$DOMAIN_NAME" =~ ^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$ ]]; then
        print_error "Invalid domain name: $DOMAIN_NAME"
        exit 1
    fi

    print_success "Domain name validated: $DOMAIN_NAME"
}

build_tls_server_names() {
    local aliases="$DOMAIN_ALIASES"

    # Normalize separators and trim whitespace.
    aliases=$(echo "$aliases" | tr ',' ' ' | xargs)

    if [[ -n "$aliases" ]]; then
        TLS_SERVER_NAMES="$DOMAIN_NAME $aliases"
    else
        TLS_SERVER_NAMES="$DOMAIN_NAME"
    fi

    print_info "TLS server names: $TLS_SERVER_NAMES"
}

generate_secrets() {
    print_header "Generating Secrets"

    if [[ -z "$JWT_SECRET" ]]; then
        JWT_SECRET=$(openssl rand -base64 32)
        print_info "Generated JWT secret"
    fi

    print_success "Secrets generated"
}

setup_directories() {
    print_header "Setting Up Directories"

    cd "$SCRIPT_DIR"

    mkdir -p backups
    mkdir -p data

    chmod 700 backups
    chmod 700 data

    print_success "Directories created"
}

create_env_file() {
    print_header "Creating Environment File"

    local env_file="$SCRIPT_DIR/.env"

    if [[ -f "$env_file" ]]; then
        print_warning ".env file already exists, backing up to .env.bak"
        cp "$env_file" "$env_file.bak"
    fi

    # Derive SSL-mode-specific settings.
    local ssl_cert_path ssl_enabled tls_cert_file tls_cert_key_file storage_base_url
    case "$SSL_TYPE" in
        letsencrypt)
            ssl_cert_path="/etc/letsencrypt"
            ssl_enabled="true"
            tls_cert_file="/etc/letsencrypt/live/$DOMAIN_NAME/fullchain.pem"
            tls_cert_key_file="/etc/letsencrypt/live/$DOMAIN_NAME/privkey.pem"
            storage_base_url="https://$DOMAIN_NAME/api/attachments"
            ;;
        self-signed)
            ssl_cert_path="/etc/ssl/observer"
            ssl_enabled="true"
            tls_cert_file="/etc/ssl/observer/certificate.crt"
            tls_cert_key_file="/etc/ssl/observer/private.key"
            storage_base_url="https://$DOMAIN_NAME/api/attachments"
            ;;
        none)
            ssl_cert_path="/etc/letsencrypt"
            ssl_enabled="false"
            tls_cert_file=""
            tls_cert_key_file=""
            storage_base_url="http://$DOMAIN_NAME/api/attachments"
            ;;
    esac

    cat > "$env_file" << EOF
# Generated by deploy-digitalocean.sh on $(date)

# Domain & SSL Configuration
DOMAIN_NAME=$DOMAIN_NAME
DOMAIN_ALIASES=$DOMAIN_ALIASES
TLS_SERVER_NAMES=$TLS_SERVER_NAMES
SSL_ENABLED=$ssl_enabled
SSL_CERT_PATH=$ssl_cert_path
TLS_CERT_FILE=$tls_cert_file
TLS_CERT_KEY_FILE=$tls_cert_key_file

# Observer Image Version
OBSERVER_VERSION=$OBSERVER_VERSION

# Security Configuration
JWT_SECRET=$JWT_SECRET

# Network & Port Configuration
GRPC_PORT=50051
API_PORT=8080
WEB_PORT=3000
NATS_HTTP_PORT=8222

# Application Configuration
LOG_LEVEL=info
MONGODB_DATABASE=observer
NATS_STREAM=tests_events
NATS_SUBJECT_PREFIX=tests.events.v1
RETAIN_MESSAGES=$RETAIN_MESSAGES

# Storage Configuration
STORAGE_DRIVER=local
STORAGE_LOCAL_BASE_PATH=/data/artifacts
STORAGE_LOCAL_BASE_URL=$storage_base_url

# Container Resource Limits (used by docker compose native resource keys)
CPU_LIMIT=2.0
MEMORY_LIMIT=2g
MEMORY_RESERVATION=1g

# Backup Configuration
BACKUP_ENABLED=$BACKUP_ENABLED
BACKUP_FREQUENCY=daily
BACKUP_RETENTION_DAYS=30
BACKUP_DESTINATION=local
EOF

    chmod 600 "$env_file"
    print_success "Environment file created: .env"
}

pull_images() {
    print_header "Pulling Docker Images"

    cd "$SCRIPT_DIR"

    docker compose -f docker-compose.digitalocean.yml pull

    print_success "Docker images pulled"
}

build_and_start() {
    print_header "Building and Starting Services"

    cd "$SCRIPT_DIR"

    if [[ "$SSL_TYPE" == "letsencrypt" ]]; then
        docker compose -f docker-compose.digitalocean.yml --profile certbot up -d
    else
        docker compose -f docker-compose.digitalocean.yml up -d
    fi

    print_success "Services started"
}

wait_for_healthy() {
    print_header "Waiting for Services to Be Healthy"

    cd "$SCRIPT_DIR"

    local max_attempts=30
    local attempt=0

    while [[ $attempt -lt $max_attempts ]]; do
        if docker compose -f docker-compose.digitalocean.yml ps --services --filter "status=running" | grep -q observer; then
            if curl -sf http://localhost/health > /dev/null 2>&1; then
                print_success "Services are healthy"
                return 0
            fi
        fi

        attempt=$((attempt + 1))
        echo -ne "Waiting for services to be healthy... ($attempt/$max_attempts)\r"
        sleep 2
    done

    print_error "Services failed to become healthy after ${max_attempts} attempts"
    print_warning "Check logs with: docker compose -f docker-compose.digitalocean.yml logs"
    return 1
}

setup_letsencrypt() {
    print_header "Setting Up Let's Encrypt SSL"

    if [[ -z "$LE_EMAIL" ]]; then
        print_error "Email address required for Let's Encrypt"
        exit 1
    fi

    # Let's Encrypt cannot issue certificates for bare IPv4 addresses.
    if [[ "$DOMAIN_NAME" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        print_error "Let's Encrypt cannot issue certificates for IP addresses. Use a domain name with --ssl letsencrypt."
        exit 1
    fi

    local cert_path="/etc/letsencrypt/live/$DOMAIN_NAME/fullchain.pem"

    # Skip bootstrap if certificate already exists.
    if [[ -f "$cert_path" ]]; then
        print_info "Existing certificate found for $DOMAIN_NAME, skipping bootstrap"
        return
    fi

    print_info "Bootstrapping first Let's Encrypt certificate for $DOMAIN_NAME"
    print_warning "DNS must point $DOMAIN_NAME to this droplet before continuing"

    mkdir -p /etc/letsencrypt
    mkdir -p /var/www/certbot

    # Ensure compose services are stopped so standalone certbot can bind port 80.
    cd "$SCRIPT_DIR"
    docker compose -f docker-compose.digitalocean.yml down || true

    local domains=()
    domains+=("-d" "$DOMAIN_NAME")

    # Normalize DOMAIN_ALIASES so both comma- and space-separated values work.
    local normalized_aliases
    normalized_aliases=$(echo "$DOMAIN_ALIASES" | tr ',' ' ')
    for alias in $normalized_aliases; do
        domains+=("-d" "$alias")
    done

    docker run --rm \
        -p 80:80 \
        -v /etc/letsencrypt:/etc/letsencrypt \
        -v /var/www/certbot:/var/www/certbot \
        docker.io/certbot/certbot:v2.10.0 certonly \
        --standalone \
        --preferred-challenges http \
        --agree-tos \
        --no-eff-email \
        --email "$LE_EMAIL" \
        "${domains[@]}"

    if [[ ! -f "$cert_path" ]]; then
        print_error "Let's Encrypt certificate bootstrap failed for $DOMAIN_NAME"
        exit 1
    fi

    print_success "Let's Encrypt certificate obtained"
}

setup_self_signed_cert() {
    print_header "Setting Up Self-Signed Certificate"

    local cert_dir="/etc/ssl/observer"
    mkdir -p "$cert_dir"

    openssl req -x509 -nodes -days 365 \
        -newkey rsa:2048 \
        -keyout "$cert_dir/private.key" \
        -out "$cert_dir/certificate.crt" \
        -subj "/C=US/ST=State/L=City/O=Observer/CN=$DOMAIN_NAME"

    chmod 600 "$cert_dir/private.key"
    chmod 644 "$cert_dir/certificate.crt"

    print_success "Self-signed certificate created"
}

setup_backup_cron() {
    print_header "Setting Up Automated Backups"

    if [[ "$BACKUP_ENABLED" == false ]]; then
        print_info "Backups not enabled, skipping"
        return
    fi

    local backup_script="$SCRIPT_DIR/backup.sh"

    if [[ ! -f "$backup_script" ]]; then
        # Create backup script
        cat > "$backup_script" << 'BACKUP_SCRIPT_EOF'
#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKUP_DIR="$SCRIPT_DIR/backups"
BACKUP_NAME="observer-backup-$(date +%Y%m%d-%H%M%S)"
RETENTION_DAYS=30
LOG_FILE="$BACKUP_DIR/backup.log"

mkdir -p "$BACKUP_DIR"
cd "$SCRIPT_DIR"

echo "[$(date)] Starting backup: $BACKUP_NAME" | tee -a "$LOG_FILE"

# Run mongodump via a separate mongo-tools container sharing the observer container's
# network namespace. This avoids a dependency on mongodump being present inside the
# AIO container (e.g., mongodump is not available on arm64 builds).
if ! docker run --rm \
    --network container:observer \
    -v "$BACKUP_DIR:/backup" \
    docker.io/mongo:6 \
    mongodump --host 127.0.0.1 --port 27017 --out "/backup/$BACKUP_NAME"; then
    echo "[$(date)] ERROR: mongodump failed" | tee -a "$LOG_FILE"
    exit 1
fi

if ! tar -czf "$BACKUP_DIR/${BACKUP_NAME}.tar.gz" -C "$BACKUP_DIR" "$BACKUP_NAME"; then
    echo "[$(date)] ERROR: tar failed" | tee -a "$LOG_FILE"
    rm -rf "$BACKUP_DIR/$BACKUP_NAME"
    exit 1
fi

rm -rf "$BACKUP_DIR/$BACKUP_NAME"

# Clean old backups
find "$BACKUP_DIR" -name "*.tar.gz" -mtime +"$RETENTION_DAYS" -delete 2>/dev/null || true

echo "[$(date)] Backup completed: ${BACKUP_NAME}.tar.gz" | tee -a "$LOG_FILE"
BACKUP_SCRIPT_EOF

        chmod +x "$backup_script"
    fi

    # Add to crontab
    local cron_entry="0 2 * * * $backup_script >> $SCRIPT_DIR/backups/backup.log 2>&1"

    if ! crontab -l 2>/dev/null | grep -q "$backup_script"; then
        (crontab -l 2>/dev/null || true; echo "$cron_entry") | crontab -
        print_success "Automated backups scheduled (daily at 2 AM)"
    else
        print_info "Backup cron job already exists"
    fi
}

print_summary() {
    print_header "Deployment Summary"

    echo ""
    echo "✓ Observer has been successfully deployed!"
    echo ""
    local web_scheme="http"
    if [[ "$SSL_TYPE" == "letsencrypt" || "$SSL_TYPE" == "self-signed" ]]; then
        web_scheme="https"
    fi

    # Use the configured gRPC port (default 50051 if not set in environment).
    local grpc_port="${GRPC_PORT:-50051}"

    echo "Access Information:"
    echo "  Web UI:     ${web_scheme}://$DOMAIN_NAME"
    echo "  gRPC:       $DOMAIN_NAME:$grpc_port"
    echo "  API:        ${web_scheme}://$DOMAIN_NAME/api"
    echo ""
    echo "Management:"
    echo "  View logs:     docker compose -f docker-compose.digitalocean.yml logs -f"
    echo "  Status:        docker compose -f docker-compose.digitalocean.yml ps"
    echo "  Stop service:  docker compose -f docker-compose.digitalocean.yml down"
    echo "  Restart:       docker compose -f docker-compose.digitalocean.yml restart"
    echo ""
    echo "Configuration:"
    echo "  Environment:   $SCRIPT_DIR/.env"
    echo "  Backups:       $SCRIPT_DIR/backups"
    echo ""
    if [[ "$SSL_TYPE" == "none" ]]; then
        echo "⚠ Warning: SSL/TLS is not configured. Configure it before using in production:"
        echo "    $SCRIPT_DIR/scripts/deploy-digitalocean.sh --help"
    fi
    echo ""
    echo "Next Steps:"
    echo "  1. Update DNS to point to this server"
    echo "  2. Configure SSL/TLS certificate"
    echo "  3. Configure test reporter integration"
    echo "  4. Set up monitoring and backups"
    echo ""
    echo "Documentation: $SCRIPT_DIR/DIGITALOCEAN_DEPLOYMENT.md"
    echo ""
}

main() {
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --domain)
                DOMAIN_NAME="$2"
                shift 2
                ;;
            --domain-alias)
                if [[ -n "$DOMAIN_ALIASES" ]]; then
                    DOMAIN_ALIASES="$DOMAIN_ALIASES $2"
                else
                    DOMAIN_ALIASES="$2"
                fi
                shift 2
                ;;
            --jwt-secret)
                JWT_SECRET="$2"
                shift 2
                ;;
            --version)
                OBSERVER_VERSION="$2"
                shift 2
                ;;
            --ssl)
                SSL_TYPE="$2"
                shift 2
                ;;
            --email)
                LE_EMAIL="$2"
                shift 2
                ;;
            --backup-enabled)
                BACKUP_ENABLED=true
                shift
                ;;
            --retain-messages)
                RETAIN_MESSAGES=true
                shift
                ;;
            --help)
                show_help
                ;;
            *)
                print_error "Unknown option: $1"
                show_help
                ;;
        esac
    done

    # Run deployment steps
    check_requirements
    validate_domain

    # Validate --ssl value before doing any work.
    case $SSL_TYPE in
        letsencrypt|self-signed|none) ;;
        *)
            print_error "Invalid --ssl value: '$SSL_TYPE'. Must be one of: letsencrypt, self-signed, none"
            exit 1
            ;;
    esac

    build_tls_server_names
    generate_secrets
    setup_directories
    create_env_file
    pull_images

    # SSL prerequisites must be in place before nginx starts.
    case $SSL_TYPE in
        letsencrypt)
            setup_letsencrypt
            ;;
        self-signed)
            setup_self_signed_cert
            ;;
    esac

    build_and_start

    if wait_for_healthy; then
        case $SSL_TYPE in
            letsencrypt)
                print_info "Let's Encrypt mode enabled"
                ;;
            none)
                print_warning "SSL/TLS not configured. Configure it later with --ssl option."
                ;;
        esac

        if [[ "$BACKUP_ENABLED" == true ]]; then
            setup_backup_cron
        fi

        print_summary
    else
        print_error "Deployment failed. Check logs above."
        exit 1
    fi
}

main "$@"
