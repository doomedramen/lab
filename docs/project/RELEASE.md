# Lab Platform - Release Guide

This document describes the release process for Lab Platform.

## Table of Contents

- [Versioning](#versioning)
- [Automated Release (GitHub Actions)](#automated-release-github-actions)
- [Manual Release](#manual-release)
- [Docker Release](#docker-release)
- [Release Checklist](#release-checklist)

---

## Versioning

Lab Platform uses [Semantic Versioning](https://semver.org/):

- **MAJOR.MINOR.PATCH** (e.g., `1.0.0`)
- Pre-release: `1.0.0-rc.1`, `1.0.0-beta.2`, `1.0.0-alpha.3`

**Tag format:** `v1.0.0`

---

## Automated Release (GitHub Actions)

### Triggering a Release

#### Option 1: Push a Tag

```bash
# Create and push tag
git tag v1.0.0
git push origin v1.0.0

# GitHub Actions will automatically:
# 1. Build Linux binary
# 2. Build Docker image
# 3. Create GitHub Release
```

#### Option 2: Workflow Dispatch (Manual)

1. Go to **Actions** → **Release** → **Run workflow**
2. Fill in parameters:
   - `version`: `1.0.0` (optional, auto-detected from tag)
   - `build_linux`: `true`
   - `build_docker`: `true`
   - `docker_registry`: `ghcr.io` (or leave empty for GitHub Container Registry)
   - `publish_release`: `true`
3. Click **Run workflow**

### What the Workflow Does

1. **Build Linux Binary**
   - Builds web UI (static export)
   - Embeds web into Go binary
   - Creates tarball with checksums

2. **Build Docker Image**
   - Multi-stage build with embedded web
   - Pushes to GitHub Container Registry
   - Generates SBOM (Software Bill of Materials)

3. **Create GitHub Release**
   - Downloads all artifacts
   - Creates release with binaries, checksums, SBOM
   - Generates release notes

### Artifacts Produced

| File | Description |
|------|-------------|
| `lab-server-linux-amd64.tar.gz` | Pre-built binary (Linux x86_64) |
| `lab-server-linux-amd64.tar.gz.sha256` | SHA256 checksum |
| `SHA256SUMS.txt` | All checksums |
| `lab-server-*.sbom.spdx.json` | Software Bill of Materials |
| Docker image | `ghcr.io/doomedramen/lab:<version>` |

---

## Manual Release

### Prerequisites

```bash
# Required tools
go version  # 1.23+
node -v     # 20+
pnpm -v     # 10+
docker -v   # 20.10+ (optional, for Docker builds)
```

### Build Release

```bash
# Clone and checkout
git clone https://github.com/doomedramen/lab.git
cd lab
git checkout v1.0.0

# Build for current platform
make -f Makefile.release release VERSION=1.0.0

# Build for all platforms
make -f Makefile.release release-all VERSION=1.0.0

# Build Docker image
make -f Makefile.release docker-build VERSION=1.0.0
```

### Build Specific Platform

```bash
# Linux x86_64
make -f Makefile.release release-linux VERSION=1.0.0

# Linux ARM64
make -f Makefile.release release-linux-arm64 VERSION=1.0.0
```

### Verify Build

```bash
# List artifacts
make -f Makefile.release list-artifacts

# Verify checksums
make -f Makefile.release verify

# Test binary
./dist/lab-server-linux-amd64 --version
```

---

## Docker Release

### Build Locally

```bash
# Build image
make -f Makefile.release docker-build VERSION=1.0.0

# Test image
docker run --rm lab:1.0.0 --help
```

### Push to Registry

```bash
# Push to GitHub Container Registry
make -f Makefile.release docker-push \
  VERSION=1.0.0 \
  REGISTRY=ghcr.io \
  IMAGE_NAME=doomedramen/lab

# Push to Docker Hub
make -f Makefile.release docker-push \
  VERSION=1.0.0 \
  REGISTRY=docker.io \
  IMAGE_NAME=yourusername/lab
```

### Multi-Architecture Build

```bash
# Build and push multi-arch image (amd64 + arm64)
make -f Makefile.release docker-buildx \
  VERSION=1.0.0 \
  PLATFORMS=linux/amd64,linux/arm64
```

---

## Release Checklist

### Pre-Release

- [ ] All tests passing (`make test`)
- [ ] Build succeeds locally (`make release VERSION=x.y.z`)
- [ ] Changelog updated
- [ ] Documentation updated
- [ ] Version bumped in code (if applicable)
- [ ] Run security scan (optional: `gosec ./...`)

### Release

- [ ] Create release branch (for major/minor releases)
  ```bash
  git checkout -b release-1.0
  ```
- [ ] Tag release
  ```bash
  git tag v1.0.0
  git push origin v1.0.0
  ```
- [ ] Monitor GitHub Actions
  - Binary build: ✅
  - Docker build: ✅
  - Release published: ✅

### Post-Release

- [ ] Verify GitHub Release
  - Binaries downloadable
  - Checksums present
  - Release notes correct
- [ ] Verify Docker image
  ```bash
  docker pull ghcr.io/doomedramen/lab:1.0.0
  docker run --rm ghcr.io/doomedramen/lab:1.0.0 --version
  ```
- [ ] Update documentation website
- [ ] Announce release (Discord, Twitter, etc.)
- [ ] Create GitHub Discussion for release

---

## Release Artifacts Structure

```
dist/
├── lab-server-linux-amd64          # Binary
├── lab-server-linux-amd64.tar.gz   # Compressed binary
├── lab-server-linux-amd64.tar.gz.sha256  # Checksum
├── SHA256SUMS.txt                  # All checksums
└── lab-server-1.0.0.sbom.spdx.json # SBOM
```

---

## Troubleshooting

### Build Fails with "embed: no matching files"

```bash
# Ensure web build exists
cd apps/web && pnpm build

# Verify embed directory
ls -la apps/api/internal/router/web/
```

### Docker Build Fails

```bash
# Check Docker daemon
docker info

# Clear build cache
docker builder prune -a

# Rebuild
make -f Makefile.release docker-build VERSION=1.0.0
```

### GitHub Actions Fails

1. Check workflow logs
2. Re-run failed jobs
3. If persistent, test locally:
   ```bash
   make -f Makefile.release release VERSION=1.0.0
   ```

---

## Version Matrix

| Component | Version Command |
|-----------|-----------------|
| Go | `go version` |
| Node.js | `node -v` |
| pnpm | `pnpm -v` |
| Docker | `docker -v` |
| libvirt | `virsh -V` |

---

## Security

### Signing Binaries (Future)

```bash
# Generate GPG key
gpg --full-generate-key

# Sign binary
gpg --detach-sign lab-server-linux-amd64.tar.gz

# Upload .sig file with release
```

### SBOM Generation

SBOM (Software Bill of Materials) is automatically generated for Docker images using [Anchore sbom-action](https://github.com/anchore/sbom-action).

```bash
# Download SBOM from release
wget https://github.com/doomedramen/lab/releases/download/v1.0.0/lab-server-1.0.0.sbom.spdx.json

# View SBOM
cat lab-server-1.0.0.sbom.spdx.json | jq
```

---

## Support

- **Issues**: [GitHub Issues](https://github.com/doomedramen/lab/issues)
- **Discussions**: [GitHub Discussions](https://github.com/doomedramen/lab/discussions)
- **Documentation**: [DEPLOYMENT.md](DEPLOYMENT.md)
