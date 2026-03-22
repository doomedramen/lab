#!/bin/bash
#
# Build Lab Platform - Single Binary with Embedded Web UI
#
# This script builds the web UI and embeds it into the Go API binary.
# The result is a single binary that serves both the API and web UI.
#
# Usage: ./build.sh [version]
# Example: ./build.sh 1.0.0
#

set -euo pipefail

VERSION="${1:-dev}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
API_DIR="$SCRIPT_DIR"
WEB_DIR="$SCRIPT_DIR/../../web"
EMBED_DIR="$API_DIR/internal/router/web"

echo "========================================"
echo "  Lab Platform - Single Binary Build"
echo "  Version: $VERSION"
echo "========================================"
echo ""

# Step 1: Build web UI
echo "[1/4] Building web UI..."
cd "$WEB_DIR"

# Set API URL for static export (relative path works for same-origin)
export NEXT_PUBLIC_API_URL=""

pnpm build

# Verify output exists
if [[ ! -d "$WEB_DIR/out" ]]; then
    echo "Error: Web build output not found at $WEB_DIR/out"
    exit 1
fi

echo "✓ Web UI built successfully"
echo ""

# Step 2: Prepare embed directory
echo "[2/4] Preparing embed directory..."

# Remove old embed directory
rm -rf "$EMBED_DIR"

# Create fresh embed directory
mkdir -p "$EMBED_DIR"

# Copy web output to embed directory
cp -r "$WEB_DIR/out/"* "$EMBED_DIR/"

echo "✓ Embedded files prepared at $EMBED_DIR"
echo "  Total size: $(du -sh "$EMBED_DIR" | cut -f1)"
echo ""

# Step 3: Build Go API with embedded web
echo "[3/4] Building Go API with embedded web..."
cd "$API_DIR"

# Build with version info
CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-s -w -X main.version=$VERSION" \
    -o "$SCRIPT_DIR/bin/lab-server" \
    ./cmd/server

if [[ ! -f "$SCRIPT_DIR/bin/lab-server" ]]; then
    echo "Error: Build failed, binary not found"
    exit 1
fi

echo "✓ Binary built successfully"
echo "  Location: $SCRIPT_DIR/bin/lab-server"
echo "  Size: $(du -h "$SCRIPT_DIR/bin/lab-server" | cut -f1)"
echo ""

# Step 4: Summary
echo "[4/4] Build Summary"
echo "========================================"
echo ""
echo "Single binary deployment ready!"
echo ""
echo "Binary: $SCRIPT_DIR/bin/lab-server"
echo "Size:   $(du -h "$SCRIPT_DIR/bin/lab-server" | cut -f1)"
echo ""
echo "The binary includes:"
echo "  - Go API server (port 8080)"
echo "  - Static web UI (served at /)"
echo "  - libvirt client bindings"
echo ""
echo "Deploy with:"
echo "  sudo cp bin/lab-server /opt/lab/"
echo "  sudo systemctl restart lab-api"
echo ""
echo "========================================"
