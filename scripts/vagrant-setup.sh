#!/bin/bash
# scripts/vagrant-setup.sh - Install Vagrant and required plugins
# Usage: ./scripts/vagrant-setup.sh

set -e

echo "=== Setting up Vagrant for x86_64 emulation ==="

# Check if Homebrew is installed
if ! command -v brew &> /dev/null; then
    echo "❌ Homebrew not installed. Install from https://brew.sh"
    exit 1
fi

echo "✓ Homebrew found"

# Install Vagrant
if ! command -v vagrant &> /dev/null; then
    echo "Installing Vagrant..."
    brew install --cask vagrant
else
    echo "✓ Vagrant already installed"
fi

# Install QEMU
if ! command -v qemu-system-x86_64 &> /dev/null; then
    echo "Installing QEMU..."
    brew install qemu
else
    echo "✓ QEMU already installed"
fi

# Install vagrant-qemu plugin
if ! vagrant plugin list | grep -q "vagrant-qemu"; then
    echo "Installing vagrant-qemu plugin..."
    vagrant plugin install vagrant-qemu
else
    echo "✓ vagrant-qemu plugin already installed"
fi

echo ""
echo "=== Setup complete! ==="
echo ""
echo "Next steps:"
echo "1. Run 'vagrant up' to start the VM and provision"
echo "2. Run 'vagrant ssh' to connect to the VM"
echo "3. Use helper scripts:"
echo "   - ./scripts/vagrant-vet.sh    - Run go vet"
echo "   - ./scripts/vagrant-test.sh   - Run tests"
echo "   - ./scripts/vagrant-build.sh  - Build server"
echo "   - ./scripts/vagrant-shell.sh  - Interactive shell"
echo ""
