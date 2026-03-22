#!/bin/bash
set -e

# --- 1. Start System Services ---
echo "Starting dbus..."
mkdir -p /var/run/dbus
rm -f /var/run/dbus/system_bus_socket
dbus-daemon --config-file=/usr/share/dbus-1/system.conf --print-address --fork

echo "Starting virtlogd..."
virtlogd -d

echo "Starting libvirtd..."
libvirtd -d

# Wait for libvirt to be ready
echo "Waiting for libvirtd to be ready..."
MAX_TRIES=15
TRY=0
while [ $TRY -lt $MAX_TRIES ]; do
    if virsh list --all >/dev/null 2>&1; then
        echo "libvirtd is ready!"
        break
    fi
    echo "Waiting... ($TRY/$MAX_TRIES)"
    sleep 2
    TRY=$((TRY + 1))
done

if [ $TRY -eq $MAX_TRIES ]; then
    echo "Error: libvirtd failed to start"
    exit 1
fi

# --- 2. Setup Network ---
if ! virsh net-info default >/dev/null 2>&1; then
    echo "Creating default libvirt network..."
    modprobe bridge || true
    virsh net-define /etc/libvirt/qemu/networks/default.xml || true
    virsh net-start default || true
    virsh net-autostart default || true
fi

# --- 3. Run Tests ---
export CI=true
# Ensure we use the pre-built binary
export LAB_BINARY_PATH="/app/apps/api/bin/lab-server"
export JWT_SECRET="test-secret-only-for-local-development-and-testing"
export ENV=development

# Clean up background processes on exit
trap 'kill $(jobs -p) 2>/dev/null || true' EXIT

echo "Running API unit tests..."
pnpm test:unit

echo "Running API integration tests (libvirt)..."
cd apps/api
go test -v -tags=integration ./internal/repository/libvirt/...
cd ../..

# Optional: Run E2E tests if PLAYWRIGHT_SKIP is not set
if [ -z "$PLAYWRIGHT_SKIP" ]; then
    echo "Installing Playwright browsers..."
    pnpm --filter web exec playwright install --with-deps chromium

    echo "Running E2E tests..."
    pnpm test:e2e
fi

echo ""
echo "========================================"
echo "  All tests completed successfully!  "
echo "========================================"
