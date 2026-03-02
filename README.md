# Lab - Virtualization Management Platform

[![CI](https://github.com/doomedramen/lab/actions/workflows/ci.yml/badge.svg)](https://github.com/doomedramen/lab/actions/workflows/ci.yml)
[![Release](https://github.com/doomedramen/lab/actions/workflows/release.yml/badge.svg)](https://github.com/doomedramen/lab/actions/workflows/release.yml)
[![License](https://img.shields.io/github/license/doomedramen/lab)](LICENSE)

A modern, lightweight virtualization management platform for home servers. Manage VMs, containers, storage, and networks through a beautiful web interface.

![Dashboard](./docs/screenshot.png)

## Quick Start

```bash
# Clone and start development
git clone https://github.com/doomedramen/lab.git
cd lab
pnpm install
pnpm dev
```

Then open http://localhost:3000 in your browser.

## Features

- **VM Management**: Create, clone, snapshot, and backup VMs with a modern UI
- **Container Support**: LXC containers and Docker Compose stacks
- **Storage Management**: Multiple storage pools, disk resizing, ISO management
- **Networking**: Virtual networks, firewall rules, DHCP management
- **Live Metrics**: Real-time CPU, memory, disk, and network statistics
- **VNC Console**: Built-in VNC console for VM access
- **Serial Console**: WebSocket-based serial console access
- **Alerts**: Configurable alerts for resource usage and VM state changes
- **Authentication**: JWT-based auth with MFA support
- **API**: Full ConnectRPC/protobuf API for automation

## Deployment

### Production

Build and serve the static web UI:

```bash
# Build static export
pnpm --filter web build

# Serve with any static file server
npx serve apps/web/out -l 3000
```

### Development

```bash
# Install dependencies
pnpm install

# Start development servers
pnpm dev
```

See [DEPLOYMENT.md](./DEPLOYMENT.md) for detailed installation instructions.

## System Requirements

- **OS**: Linux with libvirt (Ubuntu 22.04+, Debian 12+, Fedora 39+)
- **CPU**: x86_64 with virtualization support (Intel VT-x / AMD-V)
- **RAM**: 4GB minimum (8GB+ recommended)
- **Storage**: 10GB for application + space for VMs

## Development

### Prerequisites

- Node.js 20+
- pnpm 10+
- Go 1.23+
- libvirt (for API development)

### Setup

```bash
# Clone repository
git clone https://github.com/doomedramen/lab.git
cd lab

# Install dependencies
pnpm install

# Start development servers
pnpm dev
```

### Build

```bash
# Build web UI
pnpm --filter web build

# Build API server
cd apps/api && go build -o bin/lab-server ./cmd/server
```

### Test

```bash
# Run all tests
pnpm test

# API unit tests
pnpm --filter api test

# E2E tests
pnpm --filter web test:e2e
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
│  (Next.js)      │
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

## Tech Stack

- **Frontend**: Next.js 16, React, TypeScript, Tailwind CSS, shadcn/ui
- **Backend**: Go 1.23, chi router, ConnectRPC
- **Database**: SQLite (embedded)
- **Virtualization**: libvirt, QEMU/KVM, LXC
- **Build**: pnpm workspaces, Turbo

## Documentation

- [Deployment Guide](./DEPLOYMENT.md) - Installation and configuration
- [Project Plan](./PLAN.md) - Feature roadmap and implementation status
- [API Documentation](./apps/api/README.md) - API reference

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

See [CONTRIBUTING.md](./CONTRIBUTING.md) for guidelines.

## License

MIT License - see [LICENSE](./LICENSE) for details.

## Acknowledgments

- [libvirt](https://libvirt.org/) - Virtualization API
- [Proxmox VE](https://www.proxmox.com/en/proxmox-virtual-environment/) - Inspiration for features
- [shadcn/ui](https://ui.shadcn.com/) - UI components
- [ConnectRPC](https://connectrpc.com/) - Type-safe API
