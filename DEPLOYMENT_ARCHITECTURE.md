# Lab Platform - Deployment Architecture

This document explains the deployment architecture and available options.

## Architecture Overview

### Single Binary Deployment (Recommended)

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

**Benefits:**
- ✅ Single binary (~40-50MB)
- ✅ No Node.js runtime required
- ✅ No separate web server (nginx/caddy)
- ✅ Single TLS certificate
- ✅ Atomic upgrades
- ✅ Simplified deployment

---

## Build Process

### 1. Web UI Build (Static Export)

```bash
cd apps/web
pnpm build
# Output: apps/web/out/
```

**Configuration:**
```javascript
// next.config.mjs
const nextConfig = {
  output: 'export',      // Static HTML export
  images: { unoptimized: true },
  trailingSlash: true,
}
```

**Result:** Pure static HTML/CSS/JS files - no Node.js needed.

### 2. Go API Build (with Embedded Web)

```bash
cd apps/api
./build.sh 1.0.0
# Output: bin/lab-server
```

**Embed configuration:**
```go
// apps/api/internal/router/static.go
//go:embed web/*
var webFS embed.FS
```

**Result:** Single binary with web UI embedded.

---

## Deployment Options

### Option 1: Systemd Service (Bare Metal)

**Best for:** Production home servers

```bash
# Install
sudo ./deploy/systemd/install.sh

# Manage
systemctl status lab
journalctl -u lab -f
systemctl restart lab
```

**Files:**
- Binary: `/opt/lab/api/lab-server`
- Config: `/etc/lab/config.yaml`
- Data: `/var/lib/lab/`

**See:** [`DEPLOYMENT_SYSTEMD.md`](./DEPLOYMENT_SYSTEMD.md)

---

### Option 2: Docker Compose

**Best for:** Isolated deployment, easy updates

```bash
# Build and run
export JWT_SECRET=$(openssl rand -base64 32)
docker compose up -d --build

# Access
curl http://localhost:8080
```

**Container:**
- Single container serving API + Web
- Privileged mode for libvirt access
- Host networking for performance

**See:** [`docker-compose.yml`](./docker-compose.yml)

---

### Option 3: Manual Binary Deployment

**Best for:** Custom configurations

```bash
# Build
cd apps/api && ./build.sh 1.0.0

# Deploy
sudo cp bin/lab-server /usr/local/bin/
sudo lab-server
```

---

## Runtime Requirements

### Production (Runtime)

| Component | Version | Purpose |
|-----------|---------|---------|
| libvirt | 0.9.10+ | VM/container management |
| QEMU | 4.0+ | VM emulation |
| KVM kernel module | - | Hardware acceleration |
| Docker | 20.10+ | Container support (optional) |

### Build-Time (Development Only)

| Component | Version | Purpose |
|-----------|---------|---------|
| Node.js | 20+ | Build web UI |
| pnpm | 10+ | Package management |
| Go | 1.23+ | Build API binary |
| libvirt-dev | - | Go libvirt bindings |

---

## Configuration

### Environment Variables

```bash
# /etc/lab/lab-api.env
LAB_ENV=production
LAB_SERVER_PORT=8080
LAB_LIBVIRT_URI=qemu:///system
LAB_STORAGE_DATADIR=/var/lib/lab
LAB_AUTH_JWT_SECRET=<generate-with-openssl>
LAB_LOGGING_LEVEL=info
```

### Configuration File

```yaml
# /etc/lab/config.yaml
server:
  port: "8080"
  env: "production"

backend: "libvirt"

libvirt:
  uri: "qemu:///system"

storage:
  data_dir: "/var/lib/lab"
  iso_dir: "/var/lib/lab/isos"
  vm_disk_dir: "/var/lib/libvirt/images"

auth:
  jwt_secret: "<from-env>"
```

---

## Ports

| Port | Service | Description |
|------|---------|-------------|
| 8080 | lab | API + Web UI (single port) |

**Note:** Web UI and API share the same port. No reverse proxy needed.

---

## Upgrading

### Systemd Service

```bash
# Pull changes
cd /opt/lab && git pull

# Rebuild
cd apps/api && sudo -u lab ./build.sh 1.1.0

# Restart
sudo systemctl restart lab
```

### Docker Compose

```bash
# Pull and rebuild
git pull
docker compose up -d --build
```

---

## Troubleshooting

### Check Service Status

```bash
systemctl status lab
journalctl -u lab -f
```

### API Health Check

```bash
curl http://localhost:8080/health
curl http://localhost:8080/health/ready
```

### Web UI Access

```bash
# Should return index.html
curl http://localhost:8080/

# Should return API health
curl http://localhost:8080/api/health
```

### libvirt Connection

```bash
# Check libvirt is running
systemctl status libvirtd

# Test connection
virsh -c qemu:///system list
```

---

## Migration from Dual-Service Deployment

If you previously had separate `lab-api` and `lab-web` services:

```bash
# Stop old services
sudo systemctl stop lab-api lab-web
sudo systemctl disable lab-api lab-web

# Remove old service files
sudo rm /etc/systemd/system/lab-api.service
sudo rm /etc/systemd/system/lab-web.service

# Install new single service
sudo cp deploy/systemd/lab.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable lab
sudo systemctl start lab
```

**Note:** The new binary serves both API and web UI from a single process.

---

## Security Considerations

### File Permissions

```bash
# Binary ownership
chown lab:libvirt /opt/lab/api/lab-server

# Data directory
chown -R lab:libvirt /var/lib/lab
```

### Systemd Hardening

The service file includes:
- `NoNewPrivileges=true`
- `ProtectSystem=strict`
- `ProtectHome=read-only`
- `ReadWritePaths=/var/lib/lab`

### Firewall

```bash
# Allow lab port
ufw allow 8080/tcp

# Or restrict to localhost (if using reverse proxy)
ufw allow from 127.0.0.1 to any port 8080
```

---

## Performance

### Resource Usage

| Component | Memory | CPU |
|-----------|--------|-----|
| lab-server (idle) | ~50MB | <1% |
| lab-server (loaded) | ~150MB | varies |
| Static file serving | minimal | minimal |

### Optimization Tips

1. **Use CGO build** for libvirt performance
2. **Enable HTTP/2** (built into Go server)
3. **Use systemd socket activation** for faster startup
4. **Enable libvirt caching** for frequent operations

---

## Support

- **Documentation**: [GitHub Wiki](https://github.com/doomedramen/lab/wiki)
- **Issues**: [GitHub Issues](https://github.com/doomedramen/lab/issues)
- **Discussions**: [GitHub Discussions](https://github.com/doomedramen/lab/discussions)
