#!/bin/bash
#
# Lab Platform - Bare Metal Installation Script
# 
# This script installs the Lab virtualization platform on a Linux server
# using systemd for service management.
#
# Usage: sudo ./install.sh [version]
# Example: sudo ./install.sh 1.0.0
#
set -euo pipefail

# =============================================================================
# Configuration
# =============================================================================

VERSION="${1:-latest}"
APP_USER="lab"
APP_GROUP="libvirt"
INSTALL_DIR="/opt/lab"
CONFIG_DIR="/etc/lab"
DATA_DIR="/var/lib/lab"
SYSTEMD_DIR="/etc/systemd/system"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# =============================================================================
# Helper Functions
# =============================================================================

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[OK]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root (use sudo)"
        exit 1
    fi
}

check_os() {
    if [[ ! -f /etc/os-release ]]; then
        log_error "Cannot detect OS. This script supports Ubuntu/Debian, Fedora, and Arch Linux."
        exit 1
    fi
    
    source /etc/os-release
    case "$ID" in
        ubuntu|debian)
            OS_FAMILY="debian"
            PACKAGE_MANAGER="apt"
            ;;
        fedora)
            OS_FAMILY="fedora"
            PACKAGE_MANAGER="dnf"
            ;;
        arch|manjaro)
            OS_FAMILY="arch"
            PACKAGE_MANAGER="pacman"
            ;;
        *)
            log_warn "Unsupported OS: $ID. Proceeding with caution..."
            OS_FAMILY="unknown"
            ;;
    esac
    
    log_info "Detected OS: $ID $VERSION_ID ($OS_FAMILY family)"
}

install_dependencies() {
    log_info "Installing system dependencies..."
    
    case "$OS_FAMILY" in
        debian)
            apt update -qq
            apt install -y -qq \
                libvirt-daemon-system \
                libvirt-clients \
                qemu-system-x86 \
                qemu-utils \
                lxc \
                docker.io \
                nodejs \
                npm \
                git \
                curl \
                wget \
                jq
            ;;
        fedora)
            dnf install -y \
                libvirt \
                libvirt-daemon \
                qemu-system-x86 \
                qemu-img \
                lxc \
                docker \
                nodejs \
                git \
                curl \
                wget \
                jq
            ;;
        arch)
            pacman -Sy --noconfirm \
                libvirt \
                qemu-full \
                lxc \
                docker \
                nodejs \
                npm \
                git \
                curl \
                wget \
                jq
            ;;
        *)
            log_warn "Package manager not recognized. Please install dependencies manually."
            return 1
            ;;
    esac
    
    log_success "System dependencies installed"
}

create_user_and_groups() {
    log_info "Creating application user and groups..."
    
    # Create libvirt group if it doesn't exist
    if ! getent group libvirt > /dev/null 2>&1; then
        groupadd libvirt
    fi
    
    # Create docker group if it doesn't exist
    if ! getent group docker > /dev/null 2>&1; then
        groupadd docker
    fi
    
    # Create lab user if it doesn't exist
    if ! id "$APP_USER" > /dev/null 2>&1; then
        useradd -r -s /bin/false -d "$INSTALL_DIR" "$APP_USER"
        log_success "Created user: $APP_USER"
    else
        log_info "User $APP_USER already exists"
    fi
    
    # Add user to required groups
    usermod -aG libvirt,docker "$APP_USER"
    log_success "Added $APP_USER to libvirt and docker groups"
}

create_directories() {
    log_info "Creating directories..."
    
    mkdir -p "$INSTALL_DIR"
    mkdir -p "$CONFIG_DIR"
    mkdir -p "$DATA_DIR"
    mkdir -p "$DATA_DIR/isos"
    mkdir -p "$DATA_DIR/stacks"
    mkdir -p "/var/lib/libvirt/images"
    
    # Set ownership
    chown -R "$APP_USER:$APP_GROUP" "$INSTALL_DIR"
    chown -R "$APP_USER:$APP_GROUP" "$DATA_DIR"
    chown -R "$APP_USER:libvirt" "/var/lib/libvirt/images"
    
    # Set permissions
    chmod 750 "$INSTALL_DIR"
    chmod 750 "$CONFIG_DIR"
    chmod 755 "$DATA_DIR"
    
    log_success "Directories created"
}

install_nodejs() {
    log_info "Checking Node.js installation..."
    
    if ! command -v node &> /dev/null; then
        log_warn "Node.js not found. Installing..."
        
        # Install Node.js 20.x
        case "$OS_FAMILY" in
            debian)
                curl -fsSL https://deb.nodesource.com/setup_20.x | bash -
                apt install -y nodejs
                ;;
            fedora)
                dnf install -y nodejs
                ;;
            arch)
                pacman -Sy --noconfirm nodejs npm
                ;;
        esac
    fi
    
    NODE_VERSION=$(node --version)
    log_success "Node.js installed: $NODE_VERSION"
}

install_pnpm() {
    log_info "Checking pnpm installation..."
    
    if ! command -v pnpm &> /dev/null; then
        log_warn "pnpm not found. Installing..."
        npm install -g pnpm
    fi
    
    PNPM_VERSION=$(pnpm --version)
    log_success "pnpm installed: $PNPM_VERSION"
}

build_application() {
    log_info "Building application..."
    
    cd "$INSTALL_DIR"
    
    # Clone repository if not already present
    if [[ ! -d "$INSTALL_DIR/.git" ]]; then
        log_info "Cloning repository..."
        git clone https://github.com/doomedramen/lab.git "$INSTALL_DIR"
    fi
    
    cd "$INSTALL_DIR"
    
    # Checkout specific version if not 'latest'
    if [[ "$VERSION" != "latest" ]]; then
        log_info "Checking out version: $VERSION"
        git fetch --tags
        git checkout "$VERSION" 2>/dev/null || git checkout "v$VERSION" 2>/dev/null || {
            log_error "Version $VERSION not found"
            exit 1
        }
    fi
    
    # Install pnpm (build-time only)
    log_info "Installing pnpm..."
    npm install -g pnpm
    
    # Build single binary (includes embedded web UI)
    log_info "Building single binary (API + web UI)..."
    cd "$INSTALL_DIR/apps/api"
    sudo -u lab ./build.sh "${VERSION:-dev}"
    
    # Copy binary to install directory
    mkdir -p "$INSTALL_DIR/api"
    cp "$INSTALL_DIR/apps/api/bin/lab-server" "$INSTALL_DIR/api/"
    chown "$APP_USER:$APP_GROUP" "$INSTALL_DIR/api/lab-server"
    
    cd "$INSTALL_DIR"
    log_success "Application built"
}

install_config() {
    log_info "Installing configuration..."
    
    # Generate JWT secret if not exists
    if [[ ! -f "$CONFIG_DIR/lab-api.env" ]]; then
        JWT_SECRET=$(openssl rand -base64 32)
        
        # Copy environment file
        cp "$INSTALL_DIR/deploy/systemd/lab-api.env" "$CONFIG_DIR/"
        
        # Update JWT secret in config
        sed -i "s/change-me-in-production/$JWT_SECRET/" "$CONFIG_DIR/lab-api.env"
        
        log_success "Configuration files created with new JWT secret"
    else
        log_info "Configuration files already exist"
    fi
    
    # Copy config.yaml if not exists
    if [[ ! -f "$CONFIG_DIR/config.yaml" ]]; then
        cp "$INSTALL_DIR/apps/api/config.example.yaml" "$CONFIG_DIR/config.yaml"
        
        # Update config with production settings
        sed -i 's/env: "development"/env: "production"/' "$CONFIG_DIR/config.yaml"
        sed -i 's|uri: "qemu:///session"|uri: "qemu:///system"|' "$CONFIG_DIR/config.yaml"
        
        log_success "config.yaml created"
    fi
}

install_systemd_services() {
    log_info "Installing systemd services..."
    
    # Copy service file
    cp "$INSTALL_DIR/deploy/systemd/lab.service" "$SYSTEMD_DIR/"
    
    # Reload systemd
    systemctl daemon-reload
    
    # Enable service
    systemctl enable lab.service
    
    log_success "Systemd service installed and enabled"
}

configure_libvirt() {
    log_info "Configuring libvirt..."
    
    # Start libvirtd
    systemctl enable --now libvirtd
    
    # Verify libvirt is running
    if ! systemctl is-active --quiet libvirtd; then
        log_error "Failed to start libvirtd service"
        exit 1
    fi
    
    # Test connection
    if ! virsh -c qemu:///system list > /dev/null 2>&1; then
        log_warn "libvirt connection test failed. Check permissions."
    else
        log_success "libvirt is running and accessible"
    fi
}

start_services() {
    log_info "Starting services..."
    
    # Start Lab service (includes web UI)
    systemctl start lab
    sleep 2
    
    # Check service health
    if systemctl is-active --quiet lab; then
        log_success "lab started (includes web UI)"
    else
        log_error "Failed to start lab"
        systemctl status lab --no-pager
        exit 1
    fi
}

print_summary() {
    echo ""
    echo "================================================================="
    echo -e "${GREEN}Installation completed successfully!${NC}"
    echo "================================================================="
    echo ""
    echo "Service:"
    echo "  - lab:  http://localhost:8080 (API + Web UI)"
    echo ""
    echo "Configuration:"
    echo "  - Config:   $CONFIG_DIR/config.yaml"
    echo "  - Env:      $CONFIG_DIR/lab-api.env"
    echo "  - Data:     $DATA_DIR"
    echo ""
    echo "Useful commands:"
    echo "  systemctl status lab"
    echo "  journalctl -u lab -f"
    echo "  systemctl restart lab"
    echo ""
    echo "Open http://$(hostname -I | awk '{print $1}'):8080 in your browser"
    echo "================================================================="
}

# =============================================================================
# Main Installation
# =============================================================================

main() {
    echo "================================================================="
    echo "  Lab Platform - Bare Metal Installation"
    echo "  Version: $VERSION"
    echo "================================================================="
    echo ""
    
    check_root
    check_os
    install_dependencies
    create_user_and_groups
    create_directories
    install_nodejs
    install_pnpm
    build_application
    install_config
    install_systemd_services
    configure_libvirt
    start_services
    
    print_summary
}

main "$@"
