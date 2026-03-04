# Cleanup Summary

## What Was Removed

### Outdated Files & Directories

| Path | Reason |
|------|--------|
| `apps/web/Dockerfile` | Replaced by single Dockerfile in apps/api |
| `apps/api/internal/uiserver/` | Obsolete - web UI now embedded via go:embed |
| `apps/api/internal/uiserver/uiserver.go` | Obsolete - no longer running separate Node.js server |
| `deploy/systemd/lab-api.service` | Replaced by unified lab.service |
| `deploy/systemd/lab-web.service` | Obsolete - web UI embedded in API binary |
| `deploy/systemd/lab-web.env` | Obsolete - no separate web service |
| `deploy/systemd/config.production.yaml` | Redundant - config.example.yaml sufficient |
| `.github/workflows/release-api.yml` | Replaced by unified release.yml |
| `bin/` | Empty directory |

### Outdated Documentation References

Updated in `DEPLOYMENT_SYSTEMD.md`:
- Removed all `lab-api` and `lab-web` service references
- Removed `lab-web.env` environment file references
- Removed `.next/standalone` build references
- Updated architecture diagram (single binary)
- Updated service commands (single `lab` service)
- Updated port references (8080 only, not 3000 + 8080)
- Updated TLS setup (reverse proxy to 8080)

---

## What Was Updated

### Configuration Files

| File | Change |
|------|--------|
| `.gitignore` | Added `apps/api/internal/router/web/` ignore |
| `docker-compose.yml` | Single container (lab) instead of api + web |
| `apps/api/Dockerfile` | Multi-stage build with embedded web |
| `Makefile` | Added release includes |

### Documentation

| File | Status |
|------|--------|
| `DEPLOYMENT_SYSTEMD.md` | ✅ Updated for single binary |
| `DEPLOYMENT.md` | ✅ Updated Docker section |
| `DEPLOYMENT_ARCHITECTURE.md` | ✅ New - architecture docs |
| `RELEASE.md` | ✅ New - release guide |
| `.github/RELEASE_WORKFLOW.md` | ✅ New - workflow docs |

---

## Current Deployment Structure

### Single Binary Deployment

```
lab-server (single binary ~40-50MB)
├── Go HTTP Server (port 8080)
│   ├── Connect RPC handlers (/lab.v1/*)
│   ├── WebSocket proxies (/ws/*)
│   ├── File uploads (/tus/*)
│   └── Embedded Static Files
│       ├── HTML, CSS, JS
│       ├── Images, fonts
│       └── All web assets
```

### Systemd Service

```
lab.service
├── User: lab
├── Group: libvirt
├── Port: 8080
└── Dependencies: libvirtd
```

### Docker Container

```
lab (single container)
├── Port: 8080
├── Volumes: /var/run/libvirt, /var/lib/lab
└── Privileged: true (for libvirt)
```

---

## Build Process

### Local Build

```bash
# Build single binary
make release VERSION=1.0.0

# Output
dist/lab-server-linux-amd64.tar.gz
```

### GitHub Actions Build

```yaml
# Trigger
git tag v1.0.0
git push origin v1.0.0

# Outputs
- lab-server-linux-amd64.tar.gz
- ghcr.io/doomedramen/lab:1.0.0
- GitHub Release with artifacts
```

---

## Migration from Old Deployment

### If You Have Old Installation

```bash
# Stop old services
sudo systemctl stop lab-api lab-web
sudo systemctl disable lab-api lab-web

# Remove old files
sudo rm /etc/systemd/system/lab-api.service
sudo rm /etc/systemd/system/lab-web.service
sudo rm /etc/lab/lab-web.env

# Install new single binary
cd /opt/lab
sudo git pull
cd apps/api
sudo -u lab ./build.sh 1.0.0

# Install new service
sudo cp deploy/systemd/lab.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable lab
sudo systemctl start lab
```

---

## Benefits of Cleanup

1. **Simplified Deployment** - One binary, one service, one port
2. **No Runtime Dependencies** - No Node.js, no separate web server
3. **Smaller Footprint** - ~50MB vs ~200MB+ (with Node.js)
4. **Easier Updates** - Single binary replace
5. **Better Security** - Fewer components, smaller attack surface
6. **Cleaner Codebase** - Removed ~300 lines of obsolete code

---

## Remaining TODO

- [ ] Update STYLE_GUIDE.md to remove uiserver reference
- [ ] Consider removing `contrib/` if outdated
- [ ] Review `lima.yaml` setup for single binary
- [ ] Update any CI/CD references to old deployment

---

## Verification

Run these to verify cleanup:

```bash
# Check for outdated references
grep -r "lab-web" --include="*.md" --include="*.sh" .
grep -r "lab-api.service" --include="*.md" --include="*.sh" .
grep -r "\.next/standalone" --include="*.md" --include="*.sh" .

# Should return no results (or only historical references)
```

---

## Support

If you encounter issues after cleanup:

1. Check [DEPLOYMENT_ARCHITECTURE.md](./DEPLOYMENT_ARCHITECTURE.md)
2. Review [RELEASE.md](./RELEASE.md)
3. See [DEPLOYMENT_SYSTEMD.md](./DEPLOYMENT_SYSTEMD.md)
4. Open issue on GitHub
