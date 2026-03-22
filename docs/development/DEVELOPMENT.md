# Development Setup Guide

This guide covers three ways to run backend validation (go vet, tests, builds) on the Lab project.

## Quick Comparison

| Method | Speed | Best For | Requirements |
|--------|-------|----------|--------------|
| **Direct** | ⚡ Fastest | macOS/Linux devs with libvirt | Go, libvirt-dev, buf |
| **Docker** | 🐢 Slow (emulation on ARM) | Consistent CI, Linux x86_64 | Docker |
| **Vagrant** | 🐌 Slowest (full VM) | ARM Mac devs needing x86_64 | Vagrant, QEMU |

---

## Method 1: Direct (Host Machine)

**Fastest option** if you have the dependencies installed.

### Prerequisites

#### macOS (ARM or Intel)
```bash
# Install Go
brew install go

# Install libvirt (required for CGO bindings)
brew install libvirt pkg-config

# Install buf for proto generation
brew install buf

# Set up Go path
export PATH="/opt/homebrew/bin:$PATH"  # ARM Mac
export PATH="/usr/local/bin:$PATH"     # Intel Mac
```

#### Linux (Ubuntu/Debian)
```bash
# Install Go
sudo apt-get install -y golang-go

# Install libvirt development files
sudo apt-get install -y libvirt-dev pkg-config gcc libc6-dev

# Install buf
curl -sSL "https://github.com/bufbuild/buf/releases/latest/download/buf-$(uname -s)-$(uname -m)" -o /usr/local/bin/buf
chmod +x /usr/local/bin/buf
```

### Commands

```bash
# Generate proto files
cd packages/proto && buf generate --template buf.gen.api.yaml

# Run go vet
cd apps/api && go vet ./...

# Run tests
cd apps/api && JWT_SECRET="test-secret" go test -v -short ./...

# Build server
cd apps/api && go build -o bin/lab-server ./cmd/server
```

### Makefile Targets

```bash
make proto          # Generate proto files
make test-unit      # Run unit tests
```

---

## Method 2: Docker (x86_64 Linux)

**Consistent environment** matching CI/CD. Slower on ARM Macs due to emulation.

### Prerequisites

```bash
# Install Docker Desktop or Orbstack
brew install --cask docker
# OR
brew install --cask orbstack
```

### Commands

```bash
# Run go vet
make docker-vet

# Run tests
make docker-test

# Run linters
make docker-lint

# Build server (outputs to ./apps/api/bin/)
make docker-build
```

### How It Works

- Builds Linux x86_64 container
- Installs Go, libvirt-dev, buf
- Generates protos inside container
- Runs validation
- **Note:** ARM Mac users experience ~20-50% speed due to QEMU emulation

---

## Method 3: Vagrant VM (x86_64 Linux)

**Full Linux VM** with x86_64 emulation. Best for ARM Mac devs who need native x86_64 testing.

### Prerequisites

```bash
# Install Vagrant and QEMU
brew install --cask vagrant
brew install qemu

# Install vagrant-qemu plugin
vagrant plugin install vagrant-qemu
```

### Setup (First Time Only)

```bash
# Start VM and provision (5-10 minutes first time)
make vagrant-up

# Or manually
vagrant up
```

### Commands

```bash
# Sync your latest code to VM
make vagrant-rsync

# Run go vet
make vagrant-vet

# Run tests
make vagrant-test

# Build server
make vagrant-build

# Interactive shell
make vagrant-shell
```

### Workflow

```bash
# 1. Start VM (first time provisions everything)
make vagrant-up

# 2. Make code changes in your editor

# 3. Sync changes to VM
make vagrant-rsync

# 4. Run validation
make vagrant-vet
make vagrant-test

# 5. When done, stop VM (saves state)
make vagrant-halt
```

### Performance

- **First run:** 5-10 minutes (downloads Ubuntu, installs packages)
- **Subsequent starts:** ~10 seconds (resumes from saved state)
- **go vet:** ~1-2 minutes (emulation overhead)
- **Tests:** ~2-3x slower than native x86_64

---

## Which Method Should I Use?

### Use **Direct** if:
- ✅ You're on Linux (native x86_64)
- ✅ You're on macOS and just need quick iteration
- ✅ You have libvirt installed
- ✅ You want fastest feedback loop

### Use **Docker** if:
- ✅ You want CI-consistent environment
- ✅ You're on Linux x86_64 (native speed)
- ✅ You don't want to install Go/libvirt locally
- ⚠️ ARM Mac: Expect slower performance (emulation)

### Use **Vagrant** if:
- ✅ You're on ARM Mac (M1/M2/M3)
- ✅ You need to validate x86_64 behavior
- ✅ Docker emulation fails for your use case
- ✅ You want persistent VM state
- ⚠️ Expect slower performance (full emulation)

---

## Troubleshooting

### Direct Method Issues

**"pkg-config not found"**
```bash
# macOS
brew install pkg-config

# Linux
sudo apt-get install pkg-config
```

**"libvirt.h: No such file or directory"**
```bash
# macOS
brew install libvirt
export CGO_CFLAGS="-I/opt/homebrew/opt/libvirt/include"  # ARM
export CGO_CFLAGS="-I/usr/local/opt/libvirt/include"     # Intel

# Linux
sudo apt-get install libvirt-dev
```

### Docker Method Issues

**"Cannot connect to Docker daemon"**
```bash
# Start Docker Desktop
open -a Docker

# Or restart Orbstack
orbstack restart
```

**"permission denied"**
```bash
# Add user to docker group (Linux only)
sudo usermod -aG docker $USER
newgrp docker
```

### Vagrant Method Issues

**"VM not starting"**
```bash
# Check QEMU installation
qemu-system-x86_64 --version

# Reinstall plugin
vagrant plugin uninstall vagrant-qemu
vagrant plugin install vagrant-qemu

# Destroy and recreate VM
vagrant destroy -f
vagrant up
```

**"Out of disk space"**
```bash
# SSH into VM
vagrant ssh

# Check disk usage
df -h

# Clean apt cache
sudo apt-get clean
sudo apt-get autoremove --purge

# Exit VM
exit
```

**"Files not syncing"**
```bash
# Force sync
vagrant rsync

# Or use make target
make vagrant-rsync
```

---

## CI/CD Integration

GitHub Actions uses **native Ubuntu x86_64** (fastest option):

```yaml
# .github/workflows/validate.yml
name: Backend Validation
on: [push, pull_request]
jobs:
  vet:
    runs-on: ubuntu-24.04
    steps:
      - uses: actions/checkout@v4
      - name: Setup Go
        uses: actions/setup-go@v5
      - name: Install deps
        run: sudo apt-get install -y libvirt-dev pkg-config
      - name: Generate protos
        run: |
          go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
          cd packages/proto && buf generate
      - name: Run go vet
        run: cd apps/api && go vet ./...
```

---

## Summary

| Task | Direct | Docker | Vagrant |
|------|--------|--------|---------|
| **Setup Time** | 5 min | 2 min | 10 min |
| **First Run** | Fast | Medium | Slow |
| **Subsequent** | ⚡ Fastest | 🐢 Medium | 🐌 Slow |
| **ARM Mac** | ✅ Works | ⚠️ Emulation | ⚠️ Emulation |
| **Linux x86** | ✅ Best | ✅ Great | ⚠️ Unnecessary |
| **CI Match** | ✅ Yes | ✅ Exact | ✅ Exact |

**Recommendation:** Use **Direct** for daily development, **Docker/Vagrant** for final validation before pushing.
