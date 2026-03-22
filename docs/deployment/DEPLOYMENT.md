# Lab Virtualization Platform - Deployment Guide

A lightweight virtualization management platform for home servers, providing a modern web UI for managing VMs, containers, storage, and networks via libvirt.

## Deployment Options

| Method | Best For | Complexity |
|--------|----------|------------|
| **[Systemd (Bare Metal)](DEPLOYMENT_SYSTEMD.md)** | Production home servers | Medium |
| [Manual](#manual-installation) | Custom configurations | High |

**Recommended for bare metal:** See [DEPLOYMENT_SYSTEMD.md](DEPLOYMENT_SYSTEMD.md) for complete systemd-based installation with automated scripts.

## Quick Start

### Development

```bash
# Clone and install
git clone https://github.com/doomedramen/lab.git
cd lab
pnpm install

# Start development servers
pnpm dev
```

Then open http://localhost:3000 in your browser.

### Production

```bash
# Build static export
pnpm --filter web build

# Serve with any static file server
npx serve apps/web/out -l 3000
```

## Architecture

```
┌─────────────────┐
│  Web Browser    │
│  (port 3000)    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Web UI         │
│  (Static files) │
│  port 3000      │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  API (Go)       │
│  libvirt        │
│  port 8080      │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Host libvirt   │
│  QEMU/KVM VMs   │
└─────────────────┘
```

**Components:**
- `apps/api`: Go server with libvirt bindings
- `apps/web`: Next.js UI (static export)

## System Requirements

### Minimum
- **CPU**: x86_64 with virtualization support (Intel VT-x / AMD-V)
- **RAM**: 4GB (8GB+ recommended)
- **Storage**: 10GB for application + space for VMs
- **OS**: Linux with libvirt (Ubuntu 22.04+, Debian 12+, Fedora 39+)

### Required Dependencies
- **libvirt** (0.9.10+)
- **QEMU** (for VM emulation)
- **KVM kernel module** (for hardware acceleration)

## Installation

### Ubuntu/Debian

```bash
# Install libvirt and QEMU
sudo apt update
sudo apt install -y libvirt-daemon-system libvirt-clients qemu-system-x86

# Add your user to libvirt group
sudo usermod -aG libvirt $USER

# Verify libvirt is running
sudo systemctl status libvirtd

# Build and run
pnpm install
pnpm build
```

### Fedora

```bash
# Install libvirt and QEMU
sudo dnf install -y libvirt qemu-system-x86

# Enable and start libvirt
sudo systemctl enable --now libvirtd

# Add your user to libvirt group
sudo usermod -aG libvirt $USER
```

### Arch Linux

```bash
# Install libvirt and QEMU
sudo pacman -S libvirt qemu-full

# Enable and start libvirt
sudo systemctl enable --now libvirtd

# Add your user to libvirt group
sudo usermod -aG libvirt $USER
```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `LAB_CONFIG` | Path to config file | `/etc/lab/config.yaml` |
| `LAB_ENV` | Environment (dev/production) | `production` |
| `LAB_SERVER_PORT` | API server port | `8080` |
| `LAB_LIBVIRT_URI` | libvirt connection URI | `qemu:///system` |
| `LAB_STORAGE_DATADIR` | Data directory | `/var/lib/lab` |
| `LAB_AUTH_JWT_SECRET` | JWT signing secret | (auto-generated) |

### Configuration File

Example `config.yaml`:

```yaml
server:
  port: "8080"
  env: "production"

backend:
  type: "libvirt"

libvirt:
  uri: "qemu:///system"

storage:
  data_dir: "/var/lib/lab"
  iso_dir: "/var/lib/lab/isos"
  vm_disk_dir: "/var/lib/libvirt/images"
  max_iso_size: 5368709120  # 5GB

auth:
  jwt_secret: "change-me-in-production"
  access_token_expiry: "15m"
  refresh_token_expiry: "168h"  # 7 days
  issuer: "lab"

logging:
  level: "info"
  format: "json"
  vm_log_retention_days: 7
```

## Building from Source

### Prerequisites

- Node.js 20+
- pnpm 10+
- Go 1.23+
- libvirt-dev (for Go libvirt bindings)

### Build Web UI

```bash
# Install dependencies
pnpm install

# Build static export
pnpm --filter web build

# Serve static files
npx serve apps/web/out -l 3000
```

### Build API Server

#### Development Build

```bash
cd apps/api
go build -o bin/lab-server ./cmd/server
```

#### Release Build

```bash
cd apps/api
make release VERSION=1.0.0
```

#### Cross-Platform Builds

Build binaries for all supported platforms:

```bash
cd apps/api
make release-all VERSION=1.0.0
```

This creates binaries in `apps/api/bin/`:
- `lab-server-linux-amd64`
- `lab-server-linux-arm64`

### Upgrading

### From Source

```bash
# Pull latest changes
git pull

# Reinstall dependencies
pnpm install

# Rebuild
pnpm build
```

## Troubleshooting

### libvirt Connection Issues

```bash
# Check libvirt is running
sudo systemctl status libvirtd

# Verify socket exists
ls -la /var/run/libvirt/libvirt-sock

# Test connection
virsh -c qemu:///system list
```

### Permission Denied

```bash
# Add user to libvirt group
sudo usermod -aG libvirt $USER

# Log out and back in, or:
newgrp libvirt

# Check socket permissions
ls -la /var/run/libvirt/libvirt-sock
```

### KVM Not Available

```bash
# Check KVM module is loaded
lsmod | grep kvm

# Load module (Intel)
sudo modprobe kvm_intel

# Load module (AMD)
sudo modprobe kvm_amd

# Verify
ls -la /dev/kvm
```

## Health Checks

```bash
# API health
curl http://localhost:8080/health

# Full readiness (includes libvirt check)
curl http://localhost:8080/health/ready

# Web UI
curl http://localhost:3000
```

## Logs

```bash
# API logs (if running as service)
sudo journalctl -u lab-api -f

# Web UI logs
# Check the output of your web server
```

## Backup and Restore

### Database Backup

```bash
# Backup SQLite database
cp /var/lib/lab/metrics.db ./backup.db
```

### VM Backup

Use the built-in backup feature in the web UI, or:

```bash
# Backup VM disk images
tar -czf vm-backup.tar.gz /var/lib/libvirt/images/
```

### Configuration Backup

```bash
# Backup config and data
tar -czf lab-config-backup.tar.gz /etc/lab /var/lib/lab
```

## Support

- **Documentation**: [GitHub Wiki](https://github.com/doomedramen/lab/wiki)
- **Issues**: [GitHub Issues](https://github.com/doomedramen/lab/issues)
- **Discussions**: [GitHub Discussions](https://github.com/doomedramen/lab/discussions)

## License

MIT License - see LICENSE file for details.
