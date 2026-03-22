#!/bin/bash
#
# Lab Platform - Uninstallation Script
# 
# This script removes the Lab virtualization platform from a Linux server.
#
# Usage: sudo ./uninstall.sh [--keep-data]
#
set -euo pipefail

# =============================================================================
# Configuration
# =============================================================================

KEEP_DATA=false
APP_USER="lab"
INSTALL_DIR="/opt/lab"
CONFIG_DIR="/etc/lab"
DATA_DIR="/var/lib/lab"

# Parse arguments
if [[ "${1:-}" == "--keep-data" ]]; then
    KEEP_DATA=true
fi

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

confirm() {
    local prompt="$1"
    local response
    
    if [[ "$KEEP_DATA" == "true" ]]; then
        return 0
    fi
    
    echo -n -e "${YELLOW}$prompt [y/N]${NC} "
    read -r response
    
    case "$response" in
        [yY][eE][sS]|[yY])
            return 0
            ;;
        *)
            return 1
            ;;
    esac
}

# =============================================================================
# Uninstallation Steps
# =============================================================================

stop_services() {
    log_info "Stopping services..."

    systemctl stop lab 2>/dev/null || true
    systemctl stop lab-api 2>/dev/null || true
    systemctl stop lab-web 2>/dev/null || true

    log_success "Services stopped"
}

disable_services() {
    log_info "Disabling systemd services..."

    systemctl disable lab.service 2>/dev/null || true
    systemctl disable lab-api.service 2>/dev/null || true
    systemctl disable lab-web.service 2>/dev/null || true

    # Remove service files
    rm -f /etc/systemd/system/lab.service 2>/dev/null || true
    rm -f /etc/systemd/system/lab-api.service 2>/dev/null || true
    rm -f /etc/systemd/system/lab-web.service 2>/dev/null || true

    # Reload systemd
    systemctl daemon-reload
    systemctl reset-failed 2>/dev/null || true

    log_success "Services disabled and removed"
}

remove_application() {
    log_info "Removing application files..."
    
    if [[ -d "$INSTALL_DIR" ]]; then
        rm -rf "$INSTALL_DIR"
        log_success "Removed $INSTALL_DIR"
    else
        log_info "$INSTALL_DIR does not exist"
    fi
}

remove_config() {
    log_info "Removing configuration files..."
    
    if [[ -d "$CONFIG_DIR" ]]; then
        rm -rf "$CONFIG_DIR"
        log_success "Removed $CONFIG_DIR"
    else
        log_info "$CONFIG_DIR does not exist"
    fi
}

remove_data() {
    if [[ "$KEEP_DATA" == "true" ]]; then
        log_info "Keeping data directory (--keep-data specified)"
        return 0
    fi
    
    log_info "Removing data directory..."
    
    if [[ -d "$DATA_DIR" ]]; then
        rm -rf "$DATA_DIR"
        log_success "Removed $DATA_DIR"
    else
        log_info "$DATA_DIR does not exist"
    fi
}

remove_user() {
    log_info "Removing application user..."
    
    if id "$APP_USER" > /dev/null 2>&1; then
        userdel "$APP_USER" 2>/dev/null || true
        log_success "Removed user: $APP_USER"
    else
        log_info "User $APP_USER does not exist"
    fi
}

remove_groups() {
    log_info "Note: libvirt and docker groups are kept (may be used by other applications)"
}

print_summary() {
    echo ""
    echo "================================================================="
    if [[ "$KEEP_DATA" == "true" ]]; then
        echo -e "${GREEN}Uninstallation completed (data preserved)!${NC}"
    else
        echo -e "${GREEN}Uninstallation completed!${NC}"
    fi
    echo "================================================================="
    echo ""
    echo "The following were removed:"
    echo "  - Systemd service (lab)"
    echo "  - Application files ($INSTALL_DIR)"
    echo "  - Configuration files ($CONFIG_DIR)"
    if [[ "$KEEP_DATA" != "true" ]]; then
        echo "  - Data directory ($DATA_DIR)"
    fi
    echo "  - Application user ($APP_USER)"
    echo ""
    if [[ "$KEEP_DATA" != "true" ]]; then
        echo "Note: VMs and container data may still exist in:"
        echo "  - /var/lib/libvirt/images/"
        echo "  - Docker volumes"
        echo ""
        echo "To completely remove all data, manually delete these directories."
    fi
    echo "================================================================="
}

# =============================================================================
# Main Uninstallation
# =============================================================================

main() {
    echo "================================================================="
    echo "  Lab Platform - Uninstallation"
    if [[ "$KEEP_DATA" == "true" ]]; then
        echo "  Mode: Keep data"
    else
        echo "  Mode: Full removal"
    fi
    echo "================================================================="
    echo ""
    
    check_root
    
    if ! confirm "Are you sure you want to uninstall Lab? This will remove all application files and configuration."; then
        log_info "Uninstallation cancelled"
        exit 0
    fi
    
    stop_services
    disable_services
    remove_application
    remove_config
    remove_data
    remove_user
    remove_groups
    
    print_summary
}

main "$@"
