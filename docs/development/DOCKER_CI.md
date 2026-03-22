# Docker CI Setup

This directory contains Docker configuration for running CI-like builds and tests locally. This is useful for:

- Consistent testing environments without installing all dependencies locally
- Pre-push hooks to verify changes before pushing
- Debugging CI issues locally

## Important: Platform Emulation

On ARM64 Macs (Apple Silicon), this setup uses QEMU to emulate x86_64. This is required because libvirt and qemu primarily support x86_64.

The `--platform linux/amd64` flag is set in both the Dockerfile and docker-compose file, so emulation happens automatically.

**Note:** First builds will be slower due to QEMU emulation, but subsequent builds benefit from layer caching.

## Files

- `Dockerfile.ci` - Multi-stage Dockerfile that mirrors the GitHub Actions CI workflow
- `docker-compose.ci.yml` - Docker Compose helper with multiple services
- `.dockerignore` - Excludes unnecessary files from Docker build context

## Quick Start

### Build the CI image and run full CI

```bash
docker-compose -f docker-compose.ci.yml ci
```

This will:
1. Build the Docker image with all dependencies (x86_64 emulation on ARM64)
2. Run linters and formatters
3. Run unit tests
4. Build the web UI
5. Build the final `lab-server` binary
6. Copy the binary to `./apps/api/bin/`

### Run tests only (requires full CI build first)

```bash
# First time: build the image
docker-compose -f docker-compose.ci.yml build ci

# Then run tests (mounts local source code)
docker-compose -f docker-compose.ci.yml test
```

### Run linters only (requires full CI build first)

```bash
docker-compose -f docker-compose.ci.yml lint
```

### Build binary only (skip tests, requires web build first)

```bash
docker-compose -f docker-compose.ci.yml build
```

### Interactive shell for debugging

```bash
docker-compose -f docker-compose.ci.yml shell
```

## Docker Compose Services

| Service | Description |
|---------|-------------|
| `ci` | Full CI build and test (mirrors GitHub Actions) |
| `test` | Run tests only (mounts local source code) |
| `lint` | Run linters and formatters (mounts local source code) |
| `shell` | Interactive bash shell for debugging |
| `build` | Build binary only (requires web assets from ci build) |

**Note:** The `test`, `lint`, and `build` services require the `ci` service to be built first, as they depend on the `lab-ci:latest` image with all tools and dependencies installed.

## Environment Variables

The Dockerfile uses these environment variables (matching CI):

- `JWT_SECRET` - Required for tests (default: `"ci-secret-at-least-16-chars-long"`)
- `NEXT_PUBLIC_API_URL` - Empty string for static export build
- `CGO_ENABLED=1` - For final build with libvirt support

## What Gets Built

The CI Dockerfile produces a `lab-server` binary at:

- In-container: `/app/apps/api/lab-server`
- Host: `./apps/api/bin/lab-server` (when using docker-compose)

## Integration with Pre-push Hooks

You can integrate Docker-based testing with lefthook by adding a command to `lefthook.yml`:

```yaml
pre-push:
  parallel: false
  commands:
    docker-tests:
      run: docker-compose -f docker-compose.ci.yml test
      fail_text: "Docker tests failed. Run 'docker-compose -f docker-compose.ci.yml test' to debug."
```

## Stages

The Dockerfile uses these stages:

1. **base** - Install all dependencies (Go, Node.js, pnpm, buf, proto tools, libvirt-dev)
2. **prepare** - Setup workspace and generate proto code
3. **lint** - Run linters, formatters, and dependency checks
4. **test-and-build** - Run unit tests and build the binary
5. **final** - Minimal runtime image with just the binary

## Notes

- Unit tests run with `-short` flag, which skips integration tests
- `libvirt-dev` headers are installed for CGO compilation, but the libvirtd daemon is not started in the container
- The binary includes embedded web UI (built via `pnpm --filter web build`)
- Build version is set to `ci-<git-hash>` or `ci-unknown` if no git context
- All services run with `platform: linux/amd64` for x86_64 emulation

## Troubleshooting

### Build is slow on first run

The first build downloads all dependencies (Go modules, npm packages, tools). On ARM64 Macs, QEMU emulation adds overhead. Subsequent builds will use Docker layer caching and will be much faster.

### Build fails with "no matching manifest"

Make sure you're using the docker-compose.ci.yml file which sets `platform: linux/amd64`. If building directly with docker, use:

```bash
docker build --platform linux/amd64 -f Dockerfile.ci -t lab-ci .
```

### Out of space errors

Docker images can consume significant disk space. Clean up with:

```bash
docker system prune -a
```

### Tests failing but passing locally

Check that you're using the same versions of dependencies. The Dockerfile pins specific versions:
- Go: 1.25.7
- Node.js: 22
- pnpm: 10.4.1
- buf: latest stable

### "test" service fails with "no such file or directory"

The `test` service requires the `ci` service to be built first. Run:

```bash
docker-compose -f docker-compose.ci.yml build ci
```

Then run the test service again.

### Permission errors with libvirt

The CI image does not run libvirtd daemon since unit tests use `-short` flag. If you need integration tests, you'll need to run libvirt in the container or run tests on the host.

### Emulation performance issues on ARM64 Mac

On Apple Silicon, QEMU emulation can be slow. For faster builds, consider:
1. Using a remote Linux/x86_64 builder with `docker buildx`
2. Building directly on an x86_64 machine

Example with remote builder:
```bash
docker buildx create --name remote-builder --use
docker buildx build --platform linux/amd64 -f Dockerfile.ci -t lab-ci --load .
```
