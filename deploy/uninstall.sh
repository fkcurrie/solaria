#!/usr/bin/env bash
# ==============================================================================
# Solaria: Standalone Uninstaller (uninstall.sh)
# ==============================================================================
# Usage:
#   Interactive:     ./uninstall.sh
#   Non-Interactive: ./uninstall.sh -y
#   Keep Configs:    ./uninstall.sh --keep-config
# ==============================================================================
set -euo pipefail

# ANSI Color Codes
if [ -t 1 ]; then
    GREEN='\033[0;32m'
    CYAN='\033[0;36m'
    YELLOW='\033[1;33m'
    RED='\033[0;31m'
    BOLD='\033[1m'
    NC='\033[0m'
else
    GREEN=''
    CYAN=''
    YELLOW=''
    RED=''
    BOLD=''
    NC=''
fi

AUTO_YES=false
KEEP_CONFIG=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        -y|--yes|--non-interactive)
            AUTO_YES=true
            shift
            ;;
        --keep-config)
            KEEP_CONFIG=true
            shift
            ;;
        -h|--help)
            echo "Solaria Standalone Uninstaller"
            echo "Usage: ./uninstall.sh [OPTIONS]"
            echo "Options:"
            echo "  -y, --yes          Non-interactive, run without prompts"
            echo "  --keep-config      Preserve /etc/solaria configuration files"
            echo "  -h, --help         Show this help message"
            exit 0
            ;;
        *)
            shift
            ;;
    esac
done

SUDO=""
if [ "$(id -u)" -ne 0 ]; then
    if command -v sudo &>/dev/null; then
        SUDO="sudo"
    else
        echo -e "${RED}[ERROR] This script requires root privileges or sudo.${NC}"
        exit 1
    fi
fi

echo -e "${CYAN}======================================================================${NC}"
echo -e "${BOLD}Solaria Application Uninstaller${NC}"
echo -e "${CYAN}======================================================================${NC}"

if [ "$AUTO_YES" = false ]; then
    read -p "Are you sure you want to uninstall Solaria and stop all services? [y/N] " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo -e "${YELLOW}Uninstallation cancelled.${NC}"
        exit 0
    fi
fi

# Step 1: Stop and disable systemd services
echo -e "\n${CYAN}[1/5] Stopping and Disabling Systemd Services...${NC}"
SERVICES=("solaria-bridge.service" "solaria-sre.service" "solaria-cloud.service")

for SVC in "${SERVICES[@]}"; do
    if systemctl is-active --quiet "$SVC" 2>/dev/null; then
        echo -e "  Stopping ${SVC}..."
        $SUDO systemctl stop "$SVC" || true
    fi
    if systemctl is-enabled --quiet "$SVC" 2>/dev/null; then
        echo -e "  Disabling ${SVC}..."
        $SUDO systemctl disable "$SVC" || true
    fi
    if [ -f "/etc/systemd/system/${SVC}" ]; then
        echo -e "  Removing /etc/systemd/system/${SVC}..."
        $SUDO rm -f "/etc/systemd/system/${SVC}"
    fi
done

$SUDO systemctl daemon-reload 2>/dev/null || true
echo -e "  ${GREEN}[OK] Systemd services stopped and removed.${NC}"

# Step 2: Remove Avahi mDNS service
echo -e "\n${CYAN}[2/5] Removing Network Discovery (mDNS) Service...${NC}"
if [ -f "/etc/avahi/services/solaria.service" ]; then
    $SUDO rm -f "/etc/avahi/services/solaria.service"
    if command -v systemctl &>/dev/null && systemctl is-active --quiet avahi-daemon 2>/dev/null; then
        $SUDO systemctl restart avahi-daemon 2>/dev/null || true
    fi
    echo -e "  ${GREEN}[OK] mDNS service removed.${NC}"
else
    echo -e "  [OK] No mDNS service found."
fi

# Step 3: Remove system drop-in configs
echo -e "\n${CYAN}[3/5] Cleaning System Resilience Configurations...${NC}"
$SUDO rm -f /etc/systemd/journald.conf.d/solaria-volatile.conf
$SUDO rm -f /etc/systemd/system.conf.d/solaria-watchdog.conf
$SUDO rm -f /etc/sysctl.d/99-solaria-resilience.conf
$SUDO rm -f /etc/modules-load.d/solaria-watchdog.conf
echo -e "  ${GREEN}[OK] Resilience drop-in configs cleaned.${NC}"

# Step 4: Remove Solaria application directories & binaries
echo -e "\n${CYAN}[4/5] Removing Application Files and Binaries...${NC}"
INSTALL_DIRS=("/opt/solaria" "/var/log/solaria")
if [ "$KEEP_CONFIG" = false ]; then
    INSTALL_DIRS+=("/etc/solaria")
fi

for DIR in "${INSTALL_DIRS[@]}"; do
    if [ -d "$DIR" ]; then
        echo -e "  Removing directory $DIR..."
        $SUDO rm -rf "$DIR"
    fi
done

# Also clean local bin directory if executed within cloned repository
if [ -d "bin" ] && [ -f "go.mod" ]; then
    echo -e "  Cleaning local build artifacts in $(pwd)/bin..."
    rm -rf bin/solaria-* bin/
fi

echo -e "  ${GREEN}[OK] Application files removed.${NC}"

# Step 5: Summary
echo -e "\n${CYAN}[5/5] Uninstallation Complete!${NC}"
echo -e "${GREEN}======================================================================${NC}"
echo -e "${BOLD}✨ Solaria has been completely removed from this system.${NC}"
echo -e "${GREEN}======================================================================${NC}\n"
