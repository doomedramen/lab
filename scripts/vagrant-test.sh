#!/bin/bash
# scripts/vagrant-test.sh - Run go tests in Vagrant VM
# Usage: ./scripts/vagrant-test.sh [test-args]

set -e

echo "=== Running go tests in Vagrant VM ==="

# Check if VM is running
if ! vagrant status | grep -q "running"; then
    echo "VM not running. Starting..."
    vagrant up
fi

# Run tests with optional arguments
vagrant ssh -c "
    export PATH=/usr/local/go/bin:/home/vagrant/go/bin:\$PATH
    export JWT_SECRET='test-secret-at-least-32-characters-long'
    cd /vagrant/apps/api
    go test -v -short \$@
" -- bash -s "$@"

echo "✓ go tests completed successfully"
