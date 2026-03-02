# Lab Project Makefile
# =============================================================================
#
# Usage:
#   make dev             Run dev servers (API + web) in Lima VM
#   make test            Run all tests
#   make help            Show all available commands

.PHONY: dev test test-unit test-e2e \
        lima-shell lima-setup lima-stop lima-start lima-restart lima-delete \
        help

# Project path inside Lima VM (virtiofs mount from lima.yaml)
APP := /app

# ── Development ────────────────────────────────────────────────────

# Start dev servers (API hot-reload + Next.js) in Lima VM
dev:
	limactl shell lab -- bash -c "cd $(APP) && pnpm dev"

# ── Testing ────────────────────────────────────────────────────────

# Run all tests (unit + e2e) in Lima VM
test: test-unit test-e2e

# Run API unit tests in Lima VM
test-unit:
	limactl shell lab -- bash -c "cd $(APP) && pnpm --filter api test"

# Run E2E tests in Lima VM
test-e2e:
	limactl shell lab -- bash -c "cd $(APP) && pnpm --filter web test:e2e"

# ── Lima VM Management ─────────────────────────────────────────────

lima-shell:
	limactl shell lab

lima-setup:
	./scripts/setup-lima.sh

lima-stop:
	limactl stop lab

lima-start:
	limactl start lab

lima-restart:
	limactl stop lab && limactl start lab

lima-delete:
	limactl delete lab

# ── Help ───────────────────────────────────────────────────────────

help:
	@echo "Lab Project Makefile"
	@echo ""
	@echo "Development:"
	@echo "  make dev             Start dev servers (API + web) in Lima VM"
	@echo ""
	@echo "Testing:"
	@echo "  make test            Run all tests (unit + e2e) in Lima VM"
	@echo "  make test-unit       Run API unit tests in Lima VM"
	@echo "  make test-e2e        Run E2E tests in Lima VM"
	@echo ""
	@echo "Lima VM:"
	@echo "  make lima-shell      Shell into Lima VM"
	@echo "  make lima-setup      One-time Lima VM setup"
	@echo "  make lima-stop       Stop Lima VM"
	@echo "  make lima-start      Start Lima VM"
	@echo "  make lima-restart    Restart Lima VM"
	@echo "  make lima-delete     Delete Lima VM"
