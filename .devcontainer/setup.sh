#!/bin/bash
set -e

echo "🚀 Setting up Observer Service development environment..."

# Repair docker socket group membership if bind-mounted from host (fixes permission denied "connect: permission denied")
# This creates a local group matching the socket GID and adds the 'vscode' user to it.
# Note: a reopen / reattach may be required for the new group membership to take effect.
if [ -S "/var/run/docker.sock" ]; then
    sock_gid=$(stat -c "%g" /var/run/docker.sock || true)
    if [ -n "$sock_gid" ]; then
        existing_group=$(getent group "$sock_gid" | cut -d: -f1 || true)
        if [ -z "$existing_group" ]; then
            groupname="dockersock${sock_gid}"
            echo "🛠 Detected docker.sock gid=$sock_gid, creating group '$groupname'..."
            if ! getent group "$groupname" >/dev/null; then
                groupadd -g "$sock_gid" "$groupname" || true
            fi
            target_group="$groupname"
        else
            target_group="$existing_group"
        fi

        echo "➕ Adding user 'vscode' to group '$target_group'..."
        usermod -aG "$target_group" vscode || true
        echo "⚠️  If you still see docker permission errors, please reopen the Codespace (Reload Window) to pick up new group membership."
    fi
fi

# Install development tools
echo "📦 Installing development tools..."
make tools

# Install Go tools for VS Code
echo "🔧 Installing Go tools for VS Code..."
go install -v golang.org/x/tools/gopls@latest
go install -v github.com/go-delve/delve/cmd/dlv@latest
go install -v honnef.co/go/tools/cmd/staticcheck@latest

# Verify Node.js and npm installation
echo "📦 Verifying Node.js and npm..."
node --version
npm --version

# Install TypeScript globally
echo "📦 Installing TypeScript globally..."
npm install -g typescript

# Download dependencies
echo "📥 Downloading Go dependencies..."
go mod download

# Create .env file if it doesn't exist
if [ ! -f .env ]; then
    echo "📝 Creating .env file from .env.example..."
    cp .env.example .env
fi

# Build all components to verify setup
echo "🔨 Building all components..."
make build-all

# Run tests to verify everything works
echo "✅ Running tests..."
make test

echo "✨ Development environment setup complete!"
echo ""
echo "📋 Available commands:"
echo "  make build-all          - Build all services"
echo "  make run-dev            - Run legacy server with DB"
echo "  make test               - Run all tests"
echo "  make mongo-up           - Start MongoDB"
echo "  make nats-up            - Start NATS"
echo "  make mongo-shell        - Open MongoDB shell"
echo ""
echo "🐳 Docker Compose services (should be running):"
echo "  - MongoDB on port 27017"
echo "  - NATS on port 4222 (monitoring on 8222)"
echo ""
echo "🎯 Service ports:"
echo "  - Ingestion gRPC: 50051"
echo "  - API HTTP: 8080"
echo ""
echo "🌐 Web development tools:"
echo "  - Node.js (LTS)"
echo "  - npm"
echo "  - TypeScript"
echo "  - Yarn"
