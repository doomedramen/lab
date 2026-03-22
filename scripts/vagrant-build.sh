#!/bin/bash
# scripts/vagrant-build.sh - Build the API server in Vagrant VM
# Usage: ./scripts/vagrant-build.sh

set -e

echo "=== Building API server in Vagrant VM ==="

# Check if VM is running
if ! vagrant status | grep -q "running"; then
    echo "VM not running. Starting..."
    vagrant up
fi

# Build the server
vagrant ssh -c "
    export PATH=/usr/local/go/bin:/home/vagrant/go/bin:\$PATH
    cd /vagrant/apps/api
    go build -o bin/lab-server ./cmd/server
    echo 'Build complete: /vagrant/apps/api/bin/lab-server'
"

echo "✓ Build completed successfully"
