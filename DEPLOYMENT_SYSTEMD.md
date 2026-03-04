# Lab Platform - Systemd Bare Metal Deployment Guide

This guide covers installing and managing the Lab virtualization platform on bare metal Linux servers using systemd for service management.

## Table of Contents

- [Overview](#overview)
- [System Requirements](#system-requirements)
- [Quick Start](#quick-start)
- [Manual Installation](#manual-installation)
- [Configuration](#configuration)
- [Service Management](#service-management)
- [Troubleshooting](#troubleshooting)
- [Backup and Restore](#backup-and-restore)
- [Upgrading](#upgrading)
- [Uninstallation](#uninstallation)

---

## Overview

The systemd deployment approach provides:

- **Native performance** - Direct access to libvirt and hardware
- **System integration** - Logging via journald, automatic startup
- **Security hardening** - Sandboxed services with minimal privileges
- **Production ready** - Restart policies, health checks, resource limits
- **No Node.js runtime** - Web UI is pre-built static files

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        User Browser                          │
│                      (port 8080)                             │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    systemd: lab.service                      │
│                    Go HTTP Server                            │
│                    Port: 8080                                │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  Embedded Static Files (go:embed)                      │ │
│  │  - HTML, CSS, JS (pre-built)                           │ │
│  │  - Images, fonts, assets                               │ │
│  └────────────────────────────────────────────────────────┘ │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  API Server (Connect RPC + WebSocket)                  │ │
│  │  - /lab.v1/*        → RPC handlers                     │ │
│  │  - /ws/*            → WebSocket proxies                │ │
│  │  - /tus/*           → File uploads                     │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    libvirtd.service                          │
│                    QEMU/KVM, LXC                             │
└─────────────────────────────────────────────────────────────┘
```

**Key benefit:** The web UI is compiled to pure static HTML/CSS/JS during build - no Node.js runtime required. Single binary serves everything on port 8080.

---

## System Requirements

### Minimum

| Component | Requirement |
|-----------|-------------|
| **CPU** | x86_64 with virtualization support (Intel VT-x / AMD-V) |
| **RAM** | 4GB (8GB+ recommended) |
| **Storage** | 10GB for application + space for VMs |
| **OS** | Linux (Ubuntu 22.04+, Debian 12+, Fedora 39+, Arch Linux) |

### Required Dependencies

#### Runtime (Production)
- **libvirt** (0.9.10+)
- **QEMU** (for VM emulation)
- **KVM kernel module** (for hardware acceleration)
- **nginx** or **caddy** (for serving static web files)
- **Docker** (optional, for container support)

#### Build-Time (Only if building from source)
- **Node.js 20+** (for building web UI)
- **pnpm** (for installing dependencies)
- **Go 1.23+** (for building API, optional if using pre-built binary)

---

## Quick Start

### Automated Installation

The easiest way to install is using the provided installation script:

```bash
# Clone the repository
git clone https://github.com/doomedramen/lab.git
cd lab/deploy/systemd

# Run installation (installs to /opt/lab)
sudo ./install.sh

# Or install a specific version
sudo ./install.sh 1.0.0
```

The script will:
1. Install system dependencies (libvirt, QEMU, Node.js, etc.)
2. Create the `lab` user and required groups
3. Build the application from source
4. Configure systemd services
5. Start the services

### Verify Installation

```bash
# Check service status
systemctl status lab

# Check API health
curl http://localhost:8080/health

# Access web UI
# Open http://<server-ip>:8080 in your browser
```

---

## Manual Installation

If you prefer manual installation or need custom configuration:

### 1. Install System Dependencies

#### Ubuntu/Debian

```bash
sudo apt update
sudo apt install -y \
    libvirt-daemon-system \
    libvirt-clients \
    qemu-system-x86 \
    qemu-utils \
    lxc \
    docker.io \
    nodejs \
    npm \
    git \
    curl \
    wget
```

#### Fedora

```bash
sudo dnf install -y \
    libvirt \
    libvirt-daemon \
    qemu-system-x86 \
    qemu-img \
    lxc \
    docker \
    nodejs \
    git \
    curl \
    wget
```

#### Arch Linux

```bash
sudo pacman -Sy --noconfirm \
    libvirt \
    qemu-full \
    lxc \
    docker \
    nodejs \
    npm \
    git \
    curl \
    wget
```

### 2. Create User and Groups

```bash
# Create libvirt group if needed
sudo groupadd -f libvirt

# Create docker group if needed
sudo groupadd -f docker

# Create lab user
sudo useradd -r -s /bin/false -d /opt/lab lab

# Add user to groups
sudo usermod -aG libvirt,docker lab
```

### 3. Create Directories

```bash
sudo mkdir -p /opt/lab
sudo mkdir -p /etc/lab
sudo mkdir -p /var/lib/lab
sudo mkdir -p /var/lib/lab/isos
sudo mkdir -p /var/lib/lab/stacks

# Set ownership
sudo chown -R lab:libvirt /opt/lab
sudo chown -R lab:libvirt /var/lib/lab
```

### 4. Build Application (Single Binary)

```bash
# Clone repository
cd /opt/lab
sudo git clone https://github.com/doomedramen/lab.git .
sudo chown -R lab:libvirt .

# Install build dependencies
sudo npm install -g pnpm

# Build single binary (includes web UI)
cd /opt/lab/apps/api
sudo -u lab ./build.sh 1.0.0

# Copy binary to install directory
sudo mkdir -p /opt/lab/api
sudo cp /opt/lab/apps/api/bin/lab-server /opt/lab/api/
sudo chown lab:libvirt /opt/lab/api/lab-server
```

**Note:** The single binary approach embeds the web UI directly into the Go binary. No separate web server or Node.js runtime is needed.

### 5. Configure Application

```bash
# Generate JWT secret
JWT_SECRET=$(openssl rand -base64 32)

# Create environment file
sudo tee /etc/lab/lab-api.env > /dev/null << EOF
LAB_ENV=production
LAB_SERVER_PORT=8080
LAB_LIBVIRT_URI=qemu:///system
LAB_STORAGE_DATADIR=/var/lib/lab
LAB_AUTH_JWT_SECRET=$JWT_SECRET
LAB_LOGGING_LEVEL=info
LAB_LOGGING_FORMAT=text
EOF

# Copy configuration
sudo cp /opt/lab/apps/api/config.example.yaml /etc/lab/config.yaml

# Edit configuration for production
sudo sed -i 's/env: "development"/env: "production"/' /etc/lab/config.yaml
sudo sed -i 's|uri: "qemu:///session"|uri: "qemu:///system"|' /etc/lab/config.yaml
```

**Note:** No web environment file needed - the web UI is embedded in the API binary!

### 6. Install Systemd Services

```bash
# Copy service file (single service - web is embedded)
sudo cp /opt/lab/deploy/systemd/lab.service /etc/systemd/system/

# Reload systemd
sudo systemctl daemon-reload

# Enable service
sudo systemctl enable lab
```

**Note:** Only one service needed! The web UI is served by the API binary.

### 7. Start Services

```bash
# Start libvirt first
sudo systemctl enable --now libvirtd

# Start Lab service (includes web UI)
sudo systemctl start lab-api

# Check status
sudo systemctl status lab-api
```

**Access the application:**
- Web UI: http://localhost:8080/
- API: http://localhost:8080/lab.v1/

---

## Configuration

### Environment Variables

#### API Environment (`/etc/lab/lab-api.env`)

| Variable | Description | Default |
|----------|-------------|---------|
| `LAB_ENV` | Environment mode | `production` |
| `LAB_SERVER_PORT` | API server port | `8080` |
| `LAB_LIBVIRT_URI` | libvirt connection URI | `qemu:///system` |
| `LAB_STORAGE_DATADIR` | Data directory | `/var/lib/lab` |
| `LAB_AUTH_JWT_SECRET` | JWT signing secret | (required) |
| `LAB_LOGGING_LEVEL` | Log level | `info` |
| `LAB_LOGGING_FORMAT` | Log format | `text` |

### Configuration File (`/etc/lab/config.yaml`)

See [`config.production.yaml`](./config.production.yaml) for a complete example with all options documented.

**Note:** No web environment file needed - the web UI is embedded in the API binary!

Key sections:
- `server` - Port and environment settings
- `libvirt` - Connection configuration
- `storage` - Data directories and limits
- `vm_defaults` - VM creation defaults
- `auth` - JWT and MFA settings
- `logging` - Log configuration

---

## Service Management

### Basic Commands

```bash
# Check status
systemctl status lab

# Start service
systemctl start lab

# Stop service
systemctl stop lab

# Restart service
systemctl restart lab

# Enable auto-start on boot
systemctl enable lab

# Disable auto-start
systemctl disable lab
```

### Viewing Logs

```bash
# View all logs
journalctl -u lab -f

# View last 100 lines
journalctl -u lab -n 100

# View logs from specific time
journalctl -u lab --since "2024-01-01 00:00:00"

# View JSON format (for parsing)
journalctl -u lab -o json
```

### Health Checks

```bash
# API health endpoint
curl http://localhost:8080/health

# Full readiness check (includes libvirt)
curl http://localhost:8080/health/ready

# Web UI check (served by same binary)
curl http://localhost:8080/
```

---

## Troubleshooting

### Services Won't Start

```bash
# Check detailed status
systemctl status lab --no-pager

# Check for dependency issues
systemctl list-dependencies lab

# Test configuration
journalctl -u lab -n 50
```

### libvirt Connection Issues

```bash
# Check libvirt status
systemctl status libvirtd

# Verify socket exists
ls -la /var/run/libvirt/libvirt-sock

# Test connection
virsh -c qemu:///system list

# Check user groups
groups lab
```

### Permission Denied Errors

```bash
# Fix directory ownership
sudo chown -R lab:libvirt /opt/lab
sudo chown -R lab:libvirt /var/lib/lab

# Check socket permissions
ls -la /var/run/libvirt/libvirt-sock

# Restart libvirtd
sudo systemctl restart libvirtd
```

### KVM Not Available

```bash
# Check KVM module
lsmod | grep kvm

# Load module (Intel)
sudo modprobe kvm_intel

# Load module (AMD)
sudo modprobe kvm_amd

# Verify device
ls -la /dev/kvm
```

### Port Already in Use

```bash
# Check what's using the port
sudo ss -tlnp | grep :8080

# Change port in environment file
sudo nano /etc/lab/lab-api.env  # Change LAB_SERVER_PORT

# Restart service
sudo systemctl restart lab
```

---

## Backup and Restore

### Database Backup

```bash
# Backup SQLite database
sudo cp /var/lib/lab/metrics.db /backup/metrics-$(date +%Y%m%d).db

# Backup with compression
sudo tar -czf /backup/lab-db-$(date +%Y%m%d).tar.gz /var/lib/lab/metrics.db
```

### Configuration Backup

```bash
# Backup all configuration
sudo tar -czf /backup/lab-config-$(date +%Y%m%d).tar.gz \
    /etc/lab \
    /var/lib/lab
```

### VM Backup

Use the built-in backup feature in the web UI, or:

```bash
# Backup VM disk images
sudo tar -czf /backup/vm-images-$(date +%Y%m%d).tar.gz \
    /var/lib/libvirt/images
```

### Restore

```bash
# Stop service
sudo systemctl stop lab

# Restore database
sudo tar -xzf /backup/lab-db-YYYYMMDD.tar.gz -C /

# Restore configuration
sudo tar -xzf /backup/lab-config-YYYYMMDD.tar.gz -C /

# Fix permissions
sudo chown -R lab:libvirt /var/lib/lab

# Start service
sudo systemctl start lab
```

---

## Upgrading

### Automated Upgrade

```bash
# Run installation script with new version
sudo ./install.sh 1.1.0

# Service will be automatically restarted
```

### Manual Upgrade

```bash
# Stop service
sudo systemctl stop lab

# Navigate to installation
cd /opt/lab

# Pull latest changes
sudo git pull

# Checkout specific version
sudo git checkout v1.1.0

# Rebuild single binary (includes web UI)
cd /opt/lab/apps/api
sudo -u lab ./build.sh 1.1.0

# Update binary
sudo cp /opt/lab/apps/api/bin/lab-server /opt/lab/api/

# Start service
sudo systemctl start lab

# Check status
systemctl status lab
```

---

## Uninstallation

### Automated Uninstall

```bash
# Remove everything including data
sudo ./uninstall.sh

# Remove application but keep data
sudo ./uninstall.sh --keep-data
```

### Manual Uninstall

```bash
# Stop and disable service
sudo systemctl stop lab
sudo systemctl disable lab
sudo systemctl daemon-reload

# Remove service files
sudo rm /etc/systemd/system/lab.service 2>/dev/null || true

# Remove application
sudo rm -rf /opt/lab

# Remove configuration
sudo rm -rf /etc/lab

# Remove data (optional)
sudo rm -rf /var/lib/lab

# Remove user
sudo userdel lab
```

---

## Security Considerations

### Firewall Configuration

```bash
# Allow lab port (API + Web UI)
sudo ufw allow 8080/tcp

# Or only allow from localhost (if using reverse proxy)
sudo ufw allow from 127.0.0.1 to any port 8080
```

### TLS/SSL Setup

The single binary serves both HTTP and can be placed behind a reverse proxy for TLS:

```bash
# Install Caddy
sudo apt install -y debian-keyring debian-archive-keyring apt-transport-https
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-stable.list
sudo apt update
sudo apt install caddy

# Configure Caddy (reverse proxy to lab)
sudo tee /etc/caddy/Caddyfile > /dev/null << EOF
lab.example.com {
    reverse_proxy localhost:8080
}
EOF

sudo systemctl reload caddy
```

### Secure JWT Secret

Always generate a strong JWT secret:

```bash
openssl rand -base64 32
```

Never use the example secrets from documentation in production.

---

## Support

- **Documentation**: [GitHub Wiki](https://github.com/doomedramen/lab/wiki)
- **Issues**: [GitHub Issues](https://github.com/doomedramen/lab/issues)
- **Discussions**: [GitHub Discussions](https://github.com/doomedramen/lab/discussions)
