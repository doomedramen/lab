#!/bin/bash
# scripts/vagrant-vet.sh - Run go vet in Vagrant VM
# Usage: ./scripts/vagrant-vet.sh

set -e

echo "=== Running go vet in Vagrant VM ==="

# Check if VM is running
if ! vagrant status | grep -q "running"; then
    echo "VM not running. Starting..."
    vagrant up
fi

# Sync files first
echo "Syncing files to VM..."
vagrant rsync

# Run vet (protos should already be generated)
vagrant ssh -c "
export PATH=/usr/local/go/bin:/home/vagrant/go/bin:\$PATH
cd /vagrant/apps/api
go vet ./...
"

echo "✓ go vet completed successfully"
