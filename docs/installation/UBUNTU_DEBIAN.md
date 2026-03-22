# Ubuntu/Debian Installation Guide

This guide provides step-by-step instructions for installing Lab on Ubuntu 22.04+ or Debian 12+ systems, including all dependencies, building the application, and running both the API server and web interface.

---

## 📋 Table of Contents

1. [Prerequisites](#prerequisites)
2. [Install System Dependencies](#install-system-dependencies)
3. [Install Go](#install-go)
4. [Install Node.js and pnpm](#install-nodejs-and-pnpm)
5. [Install libvirt and QEMU](#install-libvirt-and-qemu)
6. [Clone and Setup Project](#clone-and-setup-project)
7. [Configure Lab](#configure-lab)
8. [Build the Application](#build-the-application)
9. [Running Lab](#running-lab)
10. [Production Deployment](#production-deployment)
11. [Troubleshooting](#troubleshooting)

---

## Prerequisites

### System Requirements

- **OS**: Ubuntu 22.04+ or Debian 12+
- **Architecture**: x86_64 (amd64) or ARM64 (aarch64)
- **RAM**: Minimum 4GB (8GB+ recommended)
- **Disk**: 10GB free space minimum
- **Virtualization**: Intel VT-x or AMD-V enabled in BIOS (for VM management features)

### User Permissions

You'll need `sudo` access to install system packages. For libvirt operations, you can either:
- Run as root (not recommended)
- Add your user to the `libvirt` group (recommended)

---

## Install System Dependencies

### Ubuntu 22.04+

```bash
# Update package lists
sudo apt update

# Install build tools and dependencies
sudo apt install -y \
    build-essential \
    gcc \
    libc6-dev \
    pkg-config \
    git \
    curl \
    wget \
    unzip \
    libvirt-dev \
    qemu-system-x86 \
    libvirt-daemon-system \
    libvirt-clients
```

### Debian 12+

```bash
# Update package lists
sudo apt update

# Install build tools and dependencies
sudo apt install -y \
    build-essential \
    gcc \
    libc6-dev \
    pkg-config \
    git \
    curl \
    wget \
    unzip \
    libvirt-dev \
    qemu-system-x86 \
    libvirt-daemon-system \
    libvirt-clients
```

### Add User to Libvirt Group (Recommended)

```bash
# Add your user to the libvirt group
sudo usermod -aG libvirt $USER

# Apply group changes (or log out and back in)
newgrp libvirt

# Verify group membership
groups $USER
```

---

## Install Go

Lab requires Go 1.25.7 or later.

### Option 1: Install from Official Source (Recommended)

```bash
# Download Go 1.25.7
cd /tmp
wget https://go.dev/dl/go1.25.7.linux-amd64.tar.gz

# Remove existing Go installation (if any)
sudo rm -rf /usr/local/go

# Extract to /usr/local
sudo tar -C /usr/local -xzf go1.25.7.linux-amd64.tar.gz

# Add Go to PATH
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.zshrc 2>/dev/null || true

# Apply changes
source ~/.bashrc

# Verify installation
go version
```

### Option 2: Install via Package Manager (May be outdated)

```bash
# Ubuntu/Debian package (may not be latest version)
sudo apt install -y golang-go

# Verify version (ensure it's 1.25.7+)
go version
```

---

## Install Node.js and pnpm

Lab requires Node.js 20+ and pnpm 10.4.1+.

### Install Node.js 20 LTS

```bash
# Using NodeSource repository (recommended)
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt install -y nodejs

# Verify installation
node --version  # Should be v20.x.x
npm --version
```

### Install pnpm

```bash
# Install pnpm globally using npm
sudo npm install -g pnpm@10.4.1

# Verify installation
pnpm --version
```

---

## Install libvirt and QEMU

If you haven't already installed libvirt in the system dependencies step:

```bash
# Install libvirt and QEMU
sudo apt install -y \
    libvirt-daemon-system \
    libvirt-clients \
    qemu-system-x86 \
    ovmf \
    virtinst

# Start and enable libvirt service
sudo systemctl enable --now libvirtd

# Verify libvirt is running
sudo systemctl status libvirtd

# Test libvirt connection
virsh -c qemu:///system list --all
```

### Configure libvirt for User Session (Optional)

For development, you can use libvirt in session mode (no root required):

```bash
# Test session connection
virsh -c qemu:///session list --all
```

---

## Clone and Setup Project

```bash
# Clone the repository
cd ~
git clone https://github.com/doomedramen/lab.git
cd lab

# Install project dependencies
pnpm install

# Generate protocol buffer files
pnpm proto
```

---

## Configure Lab

### Create Configuration File

```bash
# Copy example configuration
cp apps/api/config.example.yaml apps/api/config.yaml
```

### Generate JWT Secret (Required)

```bash
# Generate a secure random JWT secret
JWT_SECRET=$(openssl rand -base64 32)
echo "Generated JWT secret: $JWT_SECRET"
```

### Edit Configuration

Open `apps/api/config.yaml` and update the following:

```yaml
# Server configuration
server:
  port: "8080"
  env: "development"  # Use 'production' for production deployments

# Backend type
backend: "libvirt"

# Libvirt connection
libvirt:
  uri: "qemu:///session"  # Use 'qemu:///system' for system-level (requires sudo)

# Storage configuration
storage:
  iso_dir: "/home/$USER/libvirt-images/isos"
  vm_disk_dir: "/home/$USER/libvirt-images/disks"

# Authentication (REQUIRED)
auth:
  jwt_secret: "YOUR_GENERATED_SECRET_HERE"  # Paste the secret from above
  access_token_expiry: "15m"
  refresh_token_expiry: "168h"  # 7 days

# Reverse proxy (optional)
proxy:
  enabled: true
  http_port: 80
  https_port: 443
```

### Create Storage Directories

```bash
# Create directories for ISOs and VM disks
mkdir -p ~/libvirt-images/isos
mkdir -p ~/libvirt-images/disks
```

---

## Build the Application

### Build the API Server

```bash
cd apps/api

# Run go vet (optional but recommended)
go vet ./...

# Build the server
go build -o bin/lab-server ./cmd/server

# Verify build
ls -lh bin/lab-server
```

### Build the Web UI (for Production)

```bash
# From project root
cd ~/lab

# Build the web UI
pnpm --filter web build

# The built files will be in apps/web/.next/
```

---

## Running Lab

### Development Mode (Recommended for Testing)

Start both API and web servers with hot-reload:

```bash
# From project root
cd ~/lab

# Start both servers
pnpm dev
```

This will:
- Start the API server on `http://localhost:8080`
- Start the Next.js dev server on `http://localhost:3000`

Access the dashboard at: **http://localhost:3000**

### Run API Server Separately

If you want to run the API server independently:

```bash
cd apps/api

# Set environment variables
export JWT_SECRET="your-jwt-secret-here"

# Run the server
./bin/lab-server
```

Or with configuration file:

```bash
cd apps/api
./bin/lab-server -config config.yaml
```

### Run Web Server Separately

```bash
cd apps/web

# Development server
pnpm dev

# Production server (after build)
pnpm start
```

---

## Production Deployment

For production deployments, see the [Deployment Guide](../deployment/DEPLOYMENT.md).

### Quick Production Setup

```bash
# Build the production binary
cd apps/api
go build -o bin/lab-server -ldflags="-s -w" ./cmd/server

# Create systemd service (see DEPLOYMENT_SYSTEMD.md)
sudo cp systemd/lab.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable lab
sudo systemctl start lab

# Verify service
sudo systemctl status lab
```

### Environment Variables for Production

Set these in `/etc/environment` or your systemd service file:

```bash
# Required
JWT_SECRET="your-secure-random-secret"

# Optional
LAB_CONFIG="/etc/lab/config.yaml"
LAB_PORT="8080"
LAB_ENV="production"
```

---

## Troubleshooting

### libvirt Connection Issues

**Error: "Failed to connect to libvirt"**

```bash
# Check if libvirtd is running
sudo systemctl status libvirtd

# Restart libvirt service
sudo systemctl restart libvirtd

# Check user permissions
groups $USER  # Should include 'libvirt'

# Test connection
virsh -c qemu:///system list
```

### Go Build Errors

**Error: "libvirt.h: No such file or directory"**

```bash
# Install libvirt development files
sudo apt install -y libvirt-dev

# Verify pkg-config can find libvirt
pkg-config --cflags libvirt
```

**Error: "pkg-config not found"**

```bash
# Install pkg-config
sudo apt install -y pkg-config
```

### Node.js/pnpm Issues

**Error: "pnpm: command not found"**

```bash
# Reinstall pnpm
sudo npm install -g pnpm@10.4.1

# Verify PATH includes npm global bin
echo $PATH | grep npm
```

### Port Already in Use

**Error: "address already in use"**

```bash
# Find process using port 8080
sudo lsof -i :8080

# Kill the process
sudo kill -9 <PID>

# Or change port in config.yaml
server:
  port: "8081"  # Use different port
```

### Permission Denied for VM Operations

**Error: "permission denied" when creating VMs**

```bash
# For system-level libvirt, add user to libvirt group
sudo usermod -aG libvirt $USER
newgrp libvirt

# Or use session mode in config.yaml
libvirt:
  uri: "qemu:///session"
```

### WebSocket Connection Failed

**Error: "WebSocket connection failed" in browser console**

```bash
# Ensure API server is running
curl http://localhost:8080/health

# Check CORS configuration in config.yaml
# For development, ensure API allows connections from web dev server
```

---

## Next Steps

- **[Development Guide](development/DEVELOPMENT.md)** — Learn about development workflows
- **[Deployment Guide](../deployment/DEPLOYMENT.md)** — Production deployment instructions
- **[Architecture](../deployment/DEPLOYMENT_ARCHITECTURE.md)** — Understand system architecture
- **[API Documentation](../api/AUTH.md)** — API authentication and usage

---

## Quick Reference

### Essential Commands

```bash
# Start development
pnpm dev

# Run tests
make test-unit

# Build production server
cd apps/api && go build -o bin/lab-server ./cmd/server

# Check libvirt status
sudo systemctl status libvirtd

# View API logs
journalctl -u lab -f  # If using systemd
```

### Default Ports

| Service | Port | URL |
|---------|------|-----|
| Web UI (dev) | 3000 | http://localhost:3000 |
| API Server | 8080 | http://localhost:8080 |
| Reverse Proxy (HTTP) | 80 | http://your-domain.com |
| Reverse Proxy (HTTPS) | 443 | https://your-domain.com |

### Configuration Files

| File | Purpose |
|------|---------|
| `apps/api/config.yaml` | Main configuration |
| `apps/web/next.config.js` | Next.js configuration |
| `package.json` | Node.js dependencies |
| `apps/api/go.mod` | Go dependencies |

---

**Last updated:** March 22, 2026
