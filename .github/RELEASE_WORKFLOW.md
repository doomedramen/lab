# Release Workflow Summary

## What Was Created

### GitHub Actions Workflows

| File | Purpose |
|------|---------|
| `.github/workflows/release.yml` | Main release workflow (binaries + Docker) |
| `.github/workflows/ci.yml` | CI pipeline (lint, test, build, E2E) |

### Release Infrastructure

| File | Purpose |
|------|---------|
| `Makefile.release` | Release automation targets |
| `RELEASE.md` | Release process documentation |
| `Makefile` (updated) | Includes release targets |

---

## Release Workflow Features

### Triggers

1. **Automatic** - Push tag matching `v*`
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. **Manual** - Workflow dispatch from GitHub Actions UI
   - Custom version
   - Selective builds (Linux, Docker)
   - Debug builds
   - Control release publishing

### Jobs

#### 1. `build-linux`
- Builds web UI (static export)
- Embeds web in Go binary
- Creates tarball with checksums
- **Output:** `lab-server-linux-amd64.tar.gz`

#### 2. `build-docker`
- Multi-stage Docker build
- Pushes to GitHub Container Registry
- Generates SBOM
- **Output:** `ghcr.io/doomedramen/lab:<version>`

#### 3. `release`
- Downloads all artifacts
- Generates combined checksums
- Creates GitHub Release with:
  - Binaries
  - Checksums
  - SBOM
  - Release notes

### Concurrency
- Cancels in-progress runs for same ref
- Prevents duplicate releases

### Permissions
- `contents: write` - Create releases
- `packages: write` - Push Docker images

---

## Usage Examples

### Create Release

```bash
# 1. Tag release
git tag v1.0.0
git push origin v1.0.0

# 2. Wait for GitHub Actions (~5-10 minutes)

# 3. Release published at:
# https://github.com/doomedramen/lab/releases/tag/v1.0.0
```

### Manual Release (Workflow Dispatch)

1. Go to **Actions** → **Release** → **Run workflow**
2. Configure:
   ```yaml
   version: 1.0.0
   build_linux: true
   build_docker: true
   docker_registry: ghcr.io
   publish_release: true
   ```
3. Click **Run workflow**

### Local Release Build

```bash
# Build for current platform
make release VERSION=1.0.0

# Build for all platforms
make release-all VERSION=1.0.0

# Build Docker image
make docker-build VERSION=1.0.0

# Push to registry
make docker-push VERSION=1.0.0 REGISTRY=ghcr.io

# Full release (all platforms + Docker)
make release-full VERSION=1.0.0
```

---

## Release Artifacts

### GitHub Release

```
Release v1.0.0
├── lab-server-linux-amd64.tar.gz    # Binary archive
├── lab-server-linux-amd64.tar.gz.sha256  # Checksum
├── SHA256SUMS.txt                   # All checksums
├── lab-server-1.0.0.sbom.spdx.json  # SBOM
└── Release notes
```

### Docker Images

```
ghcr.io/doomedramen/lab:1.0.0    # Version tag
ghcr.io/doomedramen/lab:latest   # Latest tag
```

---

## CI Workflow

Runs on every PR and push to `main`:

### Jobs

1. **lint** - ESLint, Prettier, Go lint
2. **test** - Unit tests with coverage
3. **build** - Full build (web + binary)
4. **e2e** - Playwright E2E tests
5. **docker** - Docker build test (PRs only)

### Requirements

All must pass before merge:
- ✅ Lint
- ✅ Tests
- ✅ Build

---

## Version Strategy

### Tag Format
- Releases: `v1.0.0`, `v1.0.0-rc.1`, `v1.0.0-beta.2`
- Auto-detected from tag name

### Binary Version
Set via ldflags:
```bash
go build -ldflags="-X main.version=1.0.0"
```

### Docker Tags
- Version: `ghcr.io/doomedramen/lab:1.0.0`
- Latest: `ghcr.io/doomedramen/lab:latest` (release only)

---

## Pre-Release Checklist

- [ ] Tests passing locally (`make test`)
- [ ] Build succeeds (`make release VERSION=x.y.z`)
- [ ] Changelog updated
- [ ] Documentation updated
- [ ] Version bumped (if applicable)

## Post-Release Checklist

- [ ] GitHub Release published
- [ ] Binaries downloadable
- [ ] Docker image pulls successfully
- [ ] Release notes accurate
- [ ] Announcement made (if applicable)

---

## Comparison with Vykar Workflow

| Feature | Vykar | Lab Platform |
|---------|-------|--------------|
| Multi-platform | ✅ Rust (cross) | ✅ Go (CGO for libvirt) |
| Docker image | ❌ | ✅ |
| SBOM | ✅ | ✅ |
| Manual trigger | ✅ | ✅ |
| Debug builds | ✅ | ⏸️ Future |
| Code signing | ✅ macOS | ⏸️ Future |
| Notarization | ✅ macOS | N/A |
| Matrix builds | ✅ | ⏸️ Linux only (libvirt CGO) |

**Notes:**
- Lab Platform requires CGO for libvirt bindings (Linux only)
- macOS/Windows builds possible without libvirt (API-only mode)
- Code signing can be added later

---

## Future Enhancements

### Potential Additions

1. **Multi-platform builds**
   - Linux ARM64 (Raspberry Pi, M1/M2 servers)
   - API-only builds for macOS/Windows

2. **Code signing**
   - GPG signatures for binaries
   - Cosign for Docker images

3. **Package formats**
   - `.deb` for Debian/Ubuntu
   - `.rpm` for Fedora/RHEL
   - Homebrew tap for macOS

4. **Auto-update**
   - Built-in update checker
   - GitHub Releases API integration

5. **Testing**
   - Integration tests in CI
   - E2E tests with real libvirt

---

## Troubleshooting

### Workflow Fails

1. Check logs: **Actions** → **Release** → Failed job
2. Re-run: Click **Re-run jobs**
3. Test locally: `make release VERSION=1.0.0`

### Docker Push Fails

```bash
# Check registry credentials
# For GHCR: GITHUB_TOKEN is automatic
# For Docker Hub: docker login first

# Verify image name
echo ${{ github.repository }}  # Should be: doomedramen/lab
```

### Binary Build Fails

```bash
# Check libvirt installed
virsh -V

# Check Go version
go version  # Should be 1.23+

# Clean and rebuild
make clean
make release VERSION=1.0.0
```

---

## Support

- **Workflow issues**: [GitHub Issues](https://github.com/doomedramen/lab/issues)
- **Documentation**: [RELEASE.md](./RELEASE.md)
- **Examples**: [Makefile.release](./Makefile.release)
