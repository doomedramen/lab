# Lab Project Makefile
# =============================================================================
#
# Usage:
#   make dev             Run dev servers (API + web)
#   make test            Run all tests
#   make release         Build release (see Makefile.release)
#   make help            Show all available commands

# ── Development Workflow ─────────────────────────────────────────
# Three ways to run backend validation:
#
# 1. DIRECT (fastest, requires Go + libvirt on host):
#    make proto && cd apps/api && go vet ./...
#
# 2. DOCKER (consistent with CI, any platform):
#    make docker-vet
#
# 3. VAGRANT (full x86_64 VM, best for ARM Mac):
#    make vagrant-up && make vagrant-vet
#
# See docs/DEVELOPMENT.md for complete setup guide

.PHONY: proto vet test build

# ── Direct Development (Host Machine) ────────────────────────────
# Fastest option if you have Go + libvirt installed locally
# See docs/DEVELOPMENT.md for setup instructions

# Generate proto files
proto:
	cd packages/proto && buf generate --template buf.gen.api.yaml

# Run go vet directly on host
vet:
	cd apps/api && go vet ./...

# Run tests directly on host
test-direct:
	cd apps/api && JWT_SECRET="test-secret-for-local-dev" go test -v -short ./...

# Build directly on host
build-direct:
	cd apps/api && go build -o bin/lab-server ./cmd/server

# Clean build artifacts
clean:
	rm -rf apps/api/bin/ apps/api/coverage.out apps/api/coverage.html

# Include release makefile
include Makefile.release

# ── Development ────────────────────────────────────────────────────

# Start dev servers (API hot-reload + Next.js)
dev:
	pnpm dev

# ── Testing ────────────────────────────────────────────────────────

# Run all tests (unit + e2e)
test: test-unit test-e2e

# Run API unit tests
test-unit:
	pnpm test:unit

# Run E2E tests
test-e2e:
	pnpm test:e2e

# ── Docker CI ─────────────────────────────────────────────────────
# Standalone Docker builds (no docker-compose required)
# Each Dockerfile is self-contained and builds from scratch
# IMPORTANT: All builds use linux/amd64 for CI consistency

# Run Go vet in Docker
docker-vet:
	docker build --platform linux/amd64 -f Dockerfile.vet -t lab-vet .
	docker run --rm lab-vet

# Run Go tests in Docker
docker-test:
	docker build --platform linux/amd64 -f Dockerfile.test -t lab-test .
	docker run --rm lab-test

# Run linters in Docker (ESLint, Prettier, syncpack)
docker-lint:
	docker build --platform linux/amd64 -f Dockerfile.lint -t lab-lint .
	docker run --rm lab-lint

# Full build: API server with embedded web UI
# Output binary to ./apps/api/bin/
docker-build:
	docker build --platform linux/amd64 -f Dockerfile.build -t lab-build .
	docker run --rm -v $(pwd)/apps/api/bin:/output lab-build \
		cp /app/lab-server /output/

# Build and extract binary (convenience target)
docker-ci-build: docker-build
	@echo "Binary available at ./apps/api/bin/lab-server"

# Legacy docker-ci (alias for docker-ci-build)
docker-ci: docker-ci-build

# Interactive Docker shell for debugging
docker-shell:
	docker build --platform linux/amd64 -f Dockerfile.test -t lab-test .
	docker run --rm -it lab-test /bin/bash

# ── Vagrant (x86_64 Emulation on Apple Silicon) ───────────────────
# Based on: https://medium.com/@lijia1/x86-64-emulation-on-apple-silicon-1086639f6dfc
# Prerequisites: brew install --cask vagrant qemu && vagrant plugin install vagrant-qemu

# Setup Vagrant (install dependencies)
vagrant-setup:
	./scripts/vagrant-setup.sh

# Start VM and provision
vagrant-up:
	vagrant up

# Sync files to VM (rsync)
vagrant-rsync:
	./scripts/vagrant-rsync.sh

# Run go vet in VM
vagrant-vet:
	./scripts/vagrant-vet.sh

# Run tests in VM
vagrant-test:
	./scripts/vagrant-test.sh

# Build server in VM
vagrant-build:
	./scripts/vagrant-build.sh

# Interactive shell in VM
vagrant-shell:
	./scripts/vagrant-shell.sh

# Stop VM
vagrant-halt:
	vagrant halt

# Delete VM
vagrant-destroy:
	vagrant destroy -f

# ── Help ───────────────────────────────────────────────────────────

help:
	@echo "Lab Project Makefile"
	@echo ""
	@echo "Development (choose one method):"
	@echo ""
	@echo "  DIRECT (fastest, requires Go + libvirt):"
	@echo "    make proto         Generate proto files"
	@echo "    make vet           Run go vet on host"
	@echo "    make test-direct   Run tests on host"
	@echo "    make build-direct  Build server on host"
	@echo ""
	@echo "  DOCKER (consistent with CI):"
	@echo "    make docker-vet      Run go vet in Docker"
	@echo "    make docker-test     Run tests in Docker"
	@echo "    make docker-lint     Run linters in Docker"
	@echo "    make docker-build    Build server in Docker"
	@echo "    make docker-ci-build Build and extract binary"
	@echo ""
	@echo "  VAGRANT (full x86_64 VM for ARM Mac):"
	@echo "    make vagrant-setup   Install Vagrant + QEMU"
	@echo "    make vagrant-up      Start VM and provision"
	@echo "    make vagrant-rsync   Sync files to VM"
	@echo "    make vagrant-vet     Run go vet in VM"
	@echo "    make vagrant-test    Run tests in VM"
	@echo "    make vagrant-build   Build server in VM"
	@echo "    make vagrant-shell   Interactive shell in VM"
	@echo "    make vagrant-halt    Stop VM"
	@echo "    make vagrant-destroy Delete VM"
	@echo ""
	@echo "General:"
	@echo "  make dev             Start dev servers (API + web)"
	@echo "  make test            Run all tests (unit + e2e)"
	@echo "  make test-unit       Run API unit tests"
	@echo "  make test-e2e        Run E2E tests"
	@echo "  make clean           Clean build artifacts"
	@echo ""
	@echo "Release:"
	@echo "  make release         Build release for current platform"
	@echo "  make release-all     Build for all platforms"
	@echo ""
	@echo "Documentation:"
	@echo "  See docs/DEVELOPMENT.md for complete setup guide"
