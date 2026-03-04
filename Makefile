# Lab Project Makefile
# =============================================================================
#
# Usage:
#   make dev             Run dev servers (API + web)
#   make test            Run all tests
#   make release         Build release (see Makefile.release)
#   make help            Show all available commands

.PHONY: dev test test-unit test-e2e \
	release release-all \
	help

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

# ── Help ───────────────────────────────────────────────────────────

help:
	@echo "Lab Project Makefile"
	@echo ""
	@echo "Development:"
	@echo "  make dev             Start dev servers (API + web)"
	@echo ""
	@echo "Testing:"
	@echo "  make test            Run all tests (unit + e2e)"
	@echo "  make test-unit       Run API unit tests"
	@echo "  make test-e2e        Run E2E tests"
	@echo ""
	@echo "Release:"
	@echo "  make release         Build release for current platform"
	@echo "  make release-all     Build for all platforms"
	@echo ""
	@echo "See Makefile.release for more release targets"
