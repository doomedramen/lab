# Build stage
FROM golang:1.25-alpine AS build

# Install build dependencies
RUN apk add --no-cache \
    gcc \
    musl-dev \
    libvirt-dev \
    pkgconfig

WORKDIR /app

# Copy go.mod and go.sum from apps/api
COPY apps/api/go.mod apps/api/go.sum ./apps/api/
WORKDIR /app/apps/api
RUN go mod download

# Copy the rest of the source
WORKDIR /app
COPY apps/api ./apps/api

# Build the API
WORKDIR /app/apps/api
RUN CGO_ENABLED=1 GOOS=linux go build -o /app/api ./cmd/server

# Production stage
FROM alpine:latest AS production

# Install runtime dependencies including libvirt and qemu/lxc
# Alpine unified libvirt into fewer packages compared to Debian/Ubuntu
RUN apk add --no-cache \
    libvirt-daemon \
    libvirt-client \
    qemu-img \
    qemu-system-x86_64 \
    qemu-system-aarch64 \
    qemu-modules \
    lxc \
    dbus \
    docker-compose \
    util-linux

WORKDIR /app

# Copy the binary and entrypoint
COPY --from=build /app/api .
COPY apps/api/entrypoint.sh .
RUN chmod +x entrypoint.sh

# Create directory for data
RUN mkdir -p /var/lib/lab

EXPOSE 8080

# Use the entrypoint script to start services
ENTRYPOINT ["./entrypoint.sh"]
