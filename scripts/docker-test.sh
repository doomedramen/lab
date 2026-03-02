#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

usage() {
  cat << EOF
Docker Testing Helper

Usage: $(basename "$0") [command] [options]

Commands:
  api         Run API unit tests in Docker
  e2e         Run E2E tests in Docker (requires local servers)
  build       Build Docker images
  shell       Open interactive shell
  clean       Clean up Docker volumes

Options:
  -o, --os    Target OS: debian, ubuntu, fedora, alpine (default: debian)
  -h, --help  Show this help message

Examples:
  $(basename "$0") api
  $(basename "$0") api --os ubuntu
  $(basename "$0") e2e --os fedora
  $(basename "$0") build
  $(basename "$0") clean

Note: For E2E tests, make sure your API and Web servers are running locally:
  Terminal 1: cd apps/api && go run ./cmd/server
  Terminal 2: pnpm --filter web dev
EOF
}

OS="debian"
ACTION=""

while [[ $# -gt 0 ]]; do
  case $1 in
    api|e2e|build|shell|clean)
      ACTION="$1"
      shift
      ;;
    -o|--os)
      OS="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown option: $1"
      usage
      exit 1
      ;;
  esac
done

if [ -z "$ACTION" ]; then
  usage
  exit 1
fi

case $ACTION in
  api)
    echo "Running API tests in Docker ($OS)..."
    docker compose run --rm "test-$OS" pnpm --filter api test
    ;;
  e2e)
    echo "Running E2E tests in Docker ($OS)..."
    echo "Make sure your API and Web servers are running locally."
    docker compose run --rm "test-$OS" pnpm --filter web test:e2e
    ;;
  build)
    echo "Building Docker images..."
    docker compose build
    ;;
  shell)
    echo "Opening shell in Docker ($OS)..."
    docker compose run --rm "test-$OS" bash
    ;;
  clean)
    echo "Cleaning up Docker volumes..."
    docker compose down -v
    docker volume rm "${PWD##*/}_playwright-cache" 2>/dev/null || true
    docker volume rm "${PWD##*/}_go-modules" 2>/dev/null || true
    echo "Cleanup complete."
    ;;
esac
