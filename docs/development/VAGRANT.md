# Vagrant x86_64 Emulation on Apple Silicon

This folder contains Vagrant configuration for running x86_64 Linux emulation on Apple Silicon (M1/M2/M3) Macs.

Based on: https://medium.com/@lijia1/x86-64-emulation-on-apple-silicon-1086639f6dfc

## Quick Start

### 1. Install Prerequisites

```bash
# Option A: Use the setup script
make vagrant-setup

# Option B: Manual installation
brew install --cask vagrant
brew install qemu
vagrant plugin install vagrant-qemu
```

### 2. Start the VM

```bash
# Start VM and run provisioning (installs Go, libvirt, runs tests)
make vagrant-up

# Or just start without provisioning
vagrant up
```

### 3. Run Commands

```bash
# Run go vet
make vagrant-vet

# Run tests
make vagrant-test

# Build server
make vagrant-build

# Interactive shell
make vagrant-shell
```

### 4. Stop/Cleanup

```bash
# Stop VM (preserves state)
make vagrant-halt

# Delete VM completely
make vagrant-destroy
```

## Available Commands

| Command | Description |
|---------|-------------|
| `make vagrant-setup` | Install Vagrant, QEMU, and vagrant-qemu plugin |
| `make vagrant-up` | Start VM and provision (first time only) |
| `make vagrant-vet` | Run `go vet ./...` in VM |
| `make vagrant-test` | Run `go test -v -short ./...` in VM |
| `make vagrant-build` | Build API server in VM |
| `make vagrant-shell` | Open interactive shell in VM |
| `make vagrant-halt` | Stop the VM |
| `make vagrant-destroy` | Delete the VM |

## Manual Vagrant Commands

```bash
# Start VM
vagrant up

# SSH into VM
vagrant ssh

# Run specific command
vagrant ssh -c "cd /vagrant && go vet ./..."

# Re-provision (re-run setup script)
vagrant reload --provision

# Check status
vagrant status

# Stop VM
vagrant halt

# Delete VM
vagrant destroy -f
```

## Configuration

The VM is configured in `Vagrantfile` with:
- **OS:** Ubuntu 22.04 (x86_64)
- **CPU:** 4 cores (Skylake-Client emulation)
- **Memory:** 8GB RAM
- **Disk:** Default (expandable with `qemu-img resize`)

### Adjusting Resources

Edit `Vagrantfile` to change allocated resources:

```ruby
config.vm.provider "qemu" do |qe|
  qe.smp = "cpus=8,sockets=1,cores=8,threads=1"  # More cores
  qe.memory = "16384"  # More RAM (16GB)
end
```

## Performance Notes

- **Emulation Speed:** QEMU x86_64 emulation is slower than native (~20-50% of native speed)
- **First Run:** Initial `vagrant up` takes 5-10 minutes (downloads box, installs packages)
- **Subsequent Runs:** `vagrant up` is fast (VM resumes from saved state)
- **File Sync:** Folder mounting uses rsync, changes sync on `vagrant reload`

## Troubleshooting

### VM Won't Start

```bash
# Check QEMU installation
qemu-system-x86_64 --version

# Reinstall vagrant-qemu plugin
vagrant plugin uninstall vagrant-qemu
vagrant plugin install vagrant-qemu
```

### Out of Disk Space

```bash
# SSH into VM
vagrant ssh

# Check disk usage
df -h

# Clean apt cache
sudo apt-get clean

# Remove old kernels
sudo apt-get autoremove --purge
```

### Network Issues

```bash
# Restart networking in VM
vagrant ssh -c "sudo systemctl restart networking"

# Regenerate SSH keys
vagrant destroy -f
vagrant up
```

## Comparison with Docker

| Feature | Vagrant + QEMU | Docker |
|---------|---------------|--------|
| Performance | Slower (full emulation) | Faster (partial emulation) |
| Resource Limits | Can exceed host | Limited to host |
| File I/O | Slower (rsync) | Faster (bind mounts) |
| Persistence | Full VM state | Container stateless |
| Setup Time | Longer (5-10 min) | Shorter (1-2 min) |

## When to Use

**Use Vagrant when:**
- You need more CPU/RAM than your host has
- You need full VM persistence
- Docker emulation fails

**Use Docker when:**
- You want faster builds
- You need better file I/O performance
- You're on Linux (native x86_64)
