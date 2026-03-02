#!/bin/bash
# One-time Lima VM setup script for Lab project
# Run this on your Mac to create and configure the Debian VM
#
# Usage: ./scripts/setup-lima.sh

set -euo pipefail

echo "=== Lima VM Setup for Lab Project ==="
echo ""

# ── Prerequisites ──────────────────────────────────────────────────

if ! command -v limactl &>/dev/null; then
    echo "Installing Lima via Homebrew..."
    brew install lima
else
    echo "Lima is already installed: $(limactl --version)"
fi

# ── Resolve project root (relative to this script) ────────────────

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# ── Check if VM already exists ─────────────────────────────────────

if limactl list --json 2>/dev/null | grep -q '"name":"lab"'; then
    echo ""
    echo "Lima VM 'lab' already exists."
    echo "  To recreate: limactl delete lab && ./scripts/setup-lima.sh"
    echo "  To start:    limactl start lab"
    echo ""
    exit 0
fi

# ── Create and start VM ───────────────────────────────────────────

echo ""
echo "Creating Lima VM..."
echo "  Name:    lab"
echo "  Arch:    host (aarch64 on Apple Silicon)"
echo "  Type:    vz (Apple Virtualization Framework)"
echo "  Mount:   $PROJECT_ROOT -> /app"
echo ""

# Start from the project root so "." in lima.yaml resolves correctly
cd "$PROJECT_ROOT"
limactl start --name=lab lima.yaml

echo ""
echo "=== VM Created Successfully ==="
echo ""

# ── Install project dependencies inside VM ─────────────────────────

echo "Installing project dependencies in VM..."
limactl shell lab -- bash -c "
    cd /app &&
    pnpm install &&
    cd apps/web && pnpm exec playwright install --with-deps chromium
"

echo ""
echo "=== Setup Complete ==="
echo ""
echo "Next steps:"
echo "  1. Connect VS Code: Cmd+Shift+P -> Remote-SSH: Connect to Host -> lima-lab"
echo "  2. Open /app in the remote file browser"
echo "  3. Or use terminal: limactl shell lab"
echo ""
echo "Useful commands:"
echo "  make dev          # Start dev servers (API + web) in VM"
echo "  make test         # Run all tests in VM"
echo "  make test-x86     # Run tests with x86_64 emulation"
echo "  make lima-shell   # Shell into VM"
echo ""
