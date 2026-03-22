#!/bin/bash
# scripts/vagrant-rsync.sh - Sync files to Vagrant VM
# Usage: ./scripts/vagrant-rsync.sh

set -e

echo "=== Syncing files to Vagrant VM ==="

# Check if VM is running
if ! vagrant status | grep -q "running"; then
    echo "VM not running. Please run 'vagrant up' first."
    exit 1
fi

# Sync files
vagrant rsync

echo "✓ Files synced successfully"
echo ""
echo "Note: Changes in /vagrant are from the LAST sync."
echo "Run this script again after making changes to update the VM."
