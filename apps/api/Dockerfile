# =============================================================================
# Lab Platform - Single Binary Docker Build
# 
# This Dockerfile builds both the web UI and Go API into a single container.
# The web UI is embedded into the Go binary at build time using go:embed.
#
# Result: Single container serving both API and web UI on port 8080
# =============================================================================

# -----------------------------------------------------------------------------
# Stage 1: Build Web UI (static export)
# -----------------------------------------------------------------------------
FROM node:22-alpine AS web-builder

# Enable pnpm
RUN corepack enable && corepack prepare pnpm@latest --activate

WORKDIR /app

# Install dependencies
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml turbo.json ./
COPY packages ./packages
COPY apps/web/package.json ./apps/web/
RUN pnpm install --frozen-lockfile --ignore-scripts

# Build web UI (static export)
COPY apps/web ./apps/web
WORKDIR /app/apps/web
RUN pnpm build

# -----------------------------------------------------------------------------
# Stage 2: Build Go API with embedded web
# -----------------------------------------------------------------------------
FROM golang:1.25-alpine AS api-builder

# Install build dependencies
RUN apk add --no-cache \
    gcc \
    musl-dev \
    libvirt-dev \
    pkgconfig \
    git

WORKDIR /app

# Copy go.mod and go.sum
COPY apps/api/go.mod apps/api/go.sum ./apps/api/
WORKDIR /app/apps/api
RUN go mod download

# Copy API source
WORKDIR /app
COPY apps/api ./apps/api

# Copy web build to embed directory
COPY --from=web-builder /app/apps/web/out ./apps/api/internal/router/web

# Build single binary with embedded web
WORKDIR /app/apps/api
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /app/api \
    ./cmd/server

# -----------------------------------------------------------------------------
# Stage 3: Production runtime
# -----------------------------------------------------------------------------
FROM alpine:latest AS production

# Install runtime dependencies
RUN apk add --no-cache \
    libvirt-daemon \
    libvirt-client \
    qemu-system-x86_64 \
    qemu-img \
    lxc \
    dbus \
    docker-compose \
    util-linux

WORKDIR /app

# Copy the binary and entrypoint
COPY --from=api-builder /app/api .
COPY apps/api/entrypoint.sh .
RUN chmod +x entrypoint.sh

# Create directory for data
RUN mkdir -p /var/lib/lab

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Use the entrypoint script to start services
ENTRYPOINT ["./entrypoint.sh"]
