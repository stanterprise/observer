#!/bin/bash
set -e

echo "🚀 Setting up Observer Service development environment..."

# Install development tools
echo "📦 Installing development tools..."
make tools

# Install Go tools for VS Code
echo "🔧 Installing Go tools for VS Code..."
go install -v golang.org/x/tools/gopls@latest
go install -v github.com/go-delve/delve/cmd/dlv@latest
go install -v honnef.co/go/tools/cmd/staticcheck@latest

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
echo "  make db-up              - Start PostgreSQL"
echo "  make nats-up            - Start NATS"
echo "  make db-psql            - Open PostgreSQL shell"
echo ""
echo "🐳 Docker Compose services (should be running):"
echo "  - PostgreSQL on port 5432"
echo "  - NATS on port 4222 (monitoring on 8222)"
echo ""
echo "🎯 Service ports:"
echo "  - Ingestion gRPC: 50051"
echo "  - Processor gRPC: 50052"
echo "  - API HTTP: 8080"
