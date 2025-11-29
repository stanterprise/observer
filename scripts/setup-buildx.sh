#!/usr/bin/env bash
# Quick setup script for optimized Docker builds

set -e

echo "🚀 Observer Docker Build Optimization Setup"
echo "=========================================="
echo ""

# Check if BuildKit is available
if ! command -v docker &> /dev/null; then
    echo "❌ Docker not found. Please install Docker first."
    exit 1
fi

# Check buildx
if ! docker buildx version &> /dev/null; then
    echo "❌ Docker buildx not available. Please update Docker to a newer version."
    exit 1
fi

echo "✅ Docker buildx available"

# Enable BuildKit
export DOCKER_BUILDKIT=1
echo "✅ BuildKit enabled"

# Check if multiarch builder exists
if docker buildx ls | grep -q "multiarch"; then
    echo "✅ Multi-architecture builder already exists"
else
    echo "📦 Creating multi-architecture builder..."
    docker buildx create --name multiarch --driver docker-container --use
    docker buildx inspect --bootstrap
    echo "✅ Multi-architecture builder created"
fi

# Check cache directory
CACHE_DIR="/tmp/.buildx-cache"
if [ -d "$CACHE_DIR" ]; then
    CACHE_SIZE=$(du -sh "$CACHE_DIR" 2>/dev/null | cut -f1)
    echo "✅ Build cache exists ($CACHE_SIZE)"
else
    echo "📁 Creating cache directory: $CACHE_DIR"
    mkdir -p "$CACHE_DIR"
    echo "✅ Cache directory created"
fi

echo ""
echo "🎉 Setup complete! You can now build with:"
echo ""
echo "  # Fast single-platform build (current architecture)"
echo "  make docker-build-aio"
echo ""
echo "  # Optimized multi-platform build (AMD64 + ARM64)"
echo "  make docker-buildx-aio"
echo ""
echo "📊 Expected build times:"
echo "  - First build: ~8-12 minutes"
echo "  - Subsequent builds: ~2-4 minutes (with cache)"
echo ""
echo "🧹 To clean cache and free disk space:"
echo "  make docker-buildx-clean"
echo ""
echo "📚 For more details, see: docs/BUILD_OPTIMIZATION.md"
