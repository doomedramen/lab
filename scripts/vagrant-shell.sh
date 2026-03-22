#!/bin/bash
# scripts/vagrant-shell.sh - Open interactive shell in Vagrant VM
# Usage: ./scripts/vagrant-shell.sh

set -e

echo "=== Opening shell in Vagrant VM ==="

# Check if VM is running
if ! vagrant status | grep -q "running"; then
    echo "VM not running. Starting..."
    vagrant up
fi

# Open interactive shell
vagrant ssh -c "
    export PATH=/usr/local/go/bin:/home/vagrant/go/bin:\$PATH
    cd /vagrant
    exec bash -l
"
