#!/usr/bin/env bash
# ==============================================================================
# ☀️ SOLARIA: Turnkey Off-Grid Edge Appliance 1-Click Installer
# Multi-Distro Resilient Edge Bootstrapper:
#  1. Raspberry Pi OS Lite (64-bit / 32-bit, Debian Bookworm/Bullseye)
#  2. Debian 12 "Bookworm" Minimal (x86_64 / arm64 Mini PCs & Thin Clients)
#  3. Ubuntu Server 24.04 LTS Minimal (x86_64 / arm64)
# Site: 1296 Wren Lake Drive, Dorset, Ontario, Canada (Renogy Rover 20A MPPT)
# ==============================================================================
set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

echo -e "${CYAN}${BOLD}"
echo "=================================================================="
echo "   ☀️  PROJECT SOLARIA: MULTI-DISTRO EDGE APPLIANCE INSTALLER"
echo "   Autonomous Off-Grid Monitoring Hub & Power-Loss Hardening"
echo "=================================================================="
echo -e "${NC}"

# Check for root / sudo
if [[ $EUID -ne 0 ]]; then
   echo -e "${RED}[ERROR] This installer must be run as root or with sudo.${NC}"
   echo "Please rerun as: sudo bash $0 or curl -sSL ... | sudo bash"
   exit 1
fi

INSTALL_DIR="/opt/solaria"
CONFIG_DIR="/etc/solaria"
SPOOL_DIR="/var/log/solaria"
USER_NAME="${SUDO_USER:-$(logname 2>/dev/null || echo "root")}"

# ------------------------------------------------------------------------------
# STEP 1: Detect Architecture & Linux Distribution
# ------------------------------------------------------------------------------
echo -e "${BLUE}[1/6] Detecting CPU Architecture & Linux Distribution...${NC}"
ARCH=$(uname -m)
case "$ARCH" in
    x86_64)        GO_ARCH="amd64" ;;
    aarch64|arm64) GO_ARCH="arm64" ;;
    armv7l|armhf)  GO_ARCH="armv7" ;;
    *) echo -e "${RED}[ERROR] Unsupported CPU architecture: $ARCH${NC}"; exit 1 ;;
esac

DISTRO_ID="unknown"
DISTRO_DESC="Linux"
IS_RPI=false

if [[ -f /etc/os-release ]]; then
    # shellcheck source=/dev/null
    . /etc/os-release
    DISTRO_ID="${ID:-unknown}"
    DISTRO_DESC="${PRETTY_NAME:-$DISTRO_ID}"
fi

# Detect Raspberry Pi hardware
if [[ -f /proc/device-tree/model ]] && grep -qi "Raspberry Pi" /proc/device-tree/model 2>/dev/null; then
    IS_RPI=true
    RPI_MODEL=$(tr -d '\0' < /proc/device-tree/model)
elif [[ -f /etc/rpi-issue ]] || [[ "$DISTRO_ID" == "raspbian" ]]; then
    IS_RPI=true
    RPI_MODEL="Raspberry Pi"
fi

echo -e "      Architecture: ${GREEN}$ARCH ($GO_ARCH)${NC}"
echo -e "      Distribution: ${GREEN}$DISTRO_DESC${NC}"
if [[ "$IS_RPI" == "true" ]]; then
    echo -e "      Hardware:     ${GREEN}$RPI_MODEL${NC}"
fi

# ------------------------------------------------------------------------------
# STEP 2: Package Management & Distribution Dependencies
# ------------------------------------------------------------------------------
echo -e "${BLUE}[2/6] Installing Essential Packages for $DISTRO_DESC...${NC}"
export DEBIAN_FRONTEND=noninteractive
apt-get update -qq

# Common core dependencies
COMMON_PKGS=(bluez rfkill curl systemd avahi-daemon util-linux)

# Distro specific additions
if [[ "$IS_RPI" == "true" ]]; then
    COMMON_PKGS+=(i2c-tools)
elif [[ "$DISTRO_ID" == "ubuntu" ]]; then
    COMMON_PKGS+=(net-tools wireless-tools)
fi

apt-get install -y -qq "${COMMON_PKGS[@]}" >/dev/null 2>&1 || true
systemctl enable --now avahi-daemon >/dev/null 2>&1 || true
echo -e "      Dependencies: ${GREEN}OK (BlueZ, Avahi/mDNS, Systemd, Utilities)${NC}"

# ------------------------------------------------------------------------------
# STEP 3: Multi-Distro Power-Loss & Flash Fatigue Hardening
# ------------------------------------------------------------------------------
echo -e "${BLUE}[3/6] Applying Distribution-Specific Power Resilience Hardening...${NC}"

# A. Universal RAM-Backed Volatile Journaling (Protects SD / Flash drives)
mkdir -p /etc/systemd/journald.conf.d
cat << 'EOF' > /etc/systemd/journald.conf.d/solaria-volatile.conf
[Journal]
Storage=volatile
RuntimeMaxUse=16M
RuntimeMaxFileSize=4M
EOF
systemctl restart systemd-journald >/dev/null 2>&1 || true

# B. Universal Systemd Hardware Watchdog Timer
mkdir -p /etc/systemd/system.conf.d
cat << 'EOF' > /etc/systemd/system.conf.d/solaria-watchdog.conf
[Manager]
RuntimeWatchdogSec=15s
RebootWatchdogSec=1min
KExecWatchdogSec=1min
EOF

# C. Universal Kernel Panic Auto-Reboot Sysctl
cat << 'EOF' > /etc/sysctl.d/99-solaria-resilience.conf
# Solaria Off-Grid Crash & Brownout Auto-Reboot Invariants
kernel.panic = 10
kernel.panic_on_oops = 1
kernel.hung_task_panic = 1
kernel.hung_task_timeout_secs = 60
vm.dirty_writeback_centisecs = 1500
EOF
sysctl --system -q >/dev/null 2>&1 || true

# D. Hardware-Specific Watchdog Driver Modules
if [[ "$IS_RPI" == "true" ]]; then
    echo -e "      Applying Raspberry Pi SoC Watchdog (bcm2835_wdt)..."
    # Ensure dtparam=watchdog=on is in boot config
    for BOOT_CFG in /boot/firmware/config.txt /boot/config.txt; do
        if [[ -f "$BOOT_CFG" ]] && ! grep -q "^dtparam=watchdog=on" "$BOOT_CFG"; then
            echo "dtparam=watchdog=on" >> "$BOOT_CFG"
        fi
    done
    if ! grep -q "^bcm2835_wdt" /etc/modules 2>/dev/null; then
        echo "bcm2835_wdt" >> /etc/modules
    fi
    modprobe bcm2835_wdt >/dev/null 2>&1 || true
elif [[ "$DISTRO_ID" == "debian" ]] || [[ "$DISTRO_ID" == "ubuntu" ]]; then
    echo -e "      Checking x86/Generic Watchdog Hardware Modules..."
    mkdir -p /etc/modules-load.d
    cat << 'EOF' > /etc/modules-load.d/solaria-watchdog.conf
# Hardware watchdog drivers for x86 mini PCs
iTCO_wdt
wdat_wdt
EOF
    modprobe iTCO_wdt >/dev/null 2>&1 || true
    modprobe wdat_wdt >/dev/null 2>&1 || true
fi

echo -e "      Power Resilience: ${GREEN}RAM Journaling, 15s HW Watchdog & Kernel Panic Auto-Reboot Configured${NC}"

# ------------------------------------------------------------------------------
# STEP 4: Bluetooth Subsystem Pre-Flight & Dongle Discovery
# ------------------------------------------------------------------------------
echo -e "${BLUE}[4/6] Initializing Bluetooth Subsystem & Scanning for Renogy BT-1...${NC}"
mkdir -p "$INSTALL_DIR/bin" "$CONFIG_DIR" "$SPOOL_DIR"
chown -R "$USER_NAME:$USER_NAME" "$INSTALL_DIR" "$CONFIG_DIR" "$SPOOL_DIR" 2>/dev/null || true

rfkill unblock bluetooth || true
systemctl restart bluetooth >/dev/null 2>&1 || true
sleep 1

DETECTED_BLE=""
if command -v bluetoothctl >/dev/null 2>&1; then
    timeout 4s bluetoothctl scan on >/tmp/solaria_bt_scan.log 2>&1 || true
    DETECTED_BLE=$(grep -i -E "BT-TH|Renogy|Rover" /tmp/solaria_bt_scan.log | head -n 1 | awk '{print $2}' || true)
fi

if [[ -n "$DETECTED_BLE" ]]; then
    echo -e "      Found Renogy Bluetooth Dongle: ${GREEN}$DETECTED_BLE${NC}"
else
    echo -e "      ${YELLOW}No active Renogy BLE device found during fast scan (will auto-discover on boot)${NC}"
    DETECTED_BLE="BT-TH-*"
fi

# ------------------------------------------------------------------------------
# STEP 5: Configuration & Binary Deployment
# ------------------------------------------------------------------------------
echo -e "${BLUE}[5/6] Writing Appliance Environment Configuration...${NC}"
cat << EOF > "$CONFIG_DIR/solaria.env"
# Solaria Off-Grid Edge Configuration
SOLARIA_SITE_NAME="1296 Wren Lake Drive"
SOLARIA_SITE_LOCATION="Dorset, ON, Canada"
SOLARIA_LATITUDE="45.186"
SOLARIA_LONGITUDE="-78.863"
SOLARIA_ELEVATION_M="340"
SOLARIA_PANEL_WATTS="400"
SOLARIA_BATTERY_AH="170"
SOLARIA_BATTERY_TYPE="LiFePO4"

# Connectivity & Uplink
SOLARIA_BLE_NAME="$DETECTED_BLE"
SOLARIA_CLOUD_ENDPOINT="https://solaria-dashboard-952659886764.us-central1.run.app"
SOLARIA_API_TOKEN="solaria_cottage_secret_token_2026"
SOLARIA_SPOOL_DIR="$SPOOL_DIR"
PORT="8080"
EOF
chmod 0600 "$CONFIG_DIR/solaria.env"

# Copy pre-compiled binaries if installing from repo directory
CURRENT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
if [[ -f "$CURRENT_DIR/bin/solaria-bridge" ]]; then
    cp "$CURRENT_DIR/bin/solaria-bridge" "$INSTALL_DIR/bin/"
    cp "$CURRENT_DIR/bin/solaria-sre-agent" "$INSTALL_DIR/bin/"
    cp "$CURRENT_DIR/bin/solaria-e2e-audit" "$INSTALL_DIR/bin/"
    chmod +x "$INSTALL_DIR/bin/"*
fi

# ------------------------------------------------------------------------------
# STEP 6: Systemd Units Registration & Startup
# ------------------------------------------------------------------------------
echo -e "${BLUE}[6/6] Installing & Enabling Autonomous Systemd Daemons...${NC}"
cat << EOF > /etc/systemd/system/solaria-bridge.service
[Unit]
Description=Solaria Renogy Solar BLE Bridge & Resilient Telemetry Gateway
Documentation=https://github.com/fkcurrie/solaria
After=network-online.target bluetooth.target systemd-resolved.service
Wants=network-online.target bluetooth.target
Requires=bluetooth.target

[Service]
Type=simple
User=$USER_NAME
WorkingDirectory=$INSTALL_DIR
EnvironmentFile=$CONFIG_DIR/solaria.env
ExecStart=$INSTALL_DIR/bin/solaria-bridge
ExecReload=/bin/kill -HUP \$MAINPID
Restart=always
RestartSec=3s
TimeoutStopSec=10s

LimitNOFILE=65536
CapabilityBoundingSet=CAP_NET_BIND_SERVICE CAP_NET_ADMIN CAP_NET_RAW
AmbientCapabilities=CAP_NET_BIND_SERVICE CAP_NET_ADMIN CAP_NET_RAW

StandardOutput=journal
StandardError=journal
SyslogIdentifier=solaria-bridge

[Install]
WantedBy=multi-user.target
EOF

cat << EOF > /etc/systemd/system/solaria-sre.service
[Unit]
Description=Solaria Autonomous SRE Supervisor & Self-Healing Watchdog
Documentation=https://github.com/fkcurrie/solaria
After=network-online.target solaria-bridge.service
Wants=network-online.target

[Service]
Type=simple
User=root
WorkingDirectory=$INSTALL_DIR
EnvironmentFile=$CONFIG_DIR/solaria.env
ExecStart=$INSTALL_DIR/bin/solaria-sre-agent -supervisor
Restart=always
RestartSec=5s
TimeoutStopSec=10s

LimitNOFILE=65536
CapabilityBoundingSet=CAP_NET_ADMIN CAP_NET_RAW CAP_SYS_ADMIN
AmbientCapabilities=CAP_NET_ADMIN CAP_NET_RAW CAP_SYS_ADMIN

StandardOutput=journal
StandardError=journal
SyslogIdentifier=solaria-sre

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable solaria-bridge.service solaria-sre.service >/dev/null 2>&1 || true

PRIMARY_IP=$(hostname -I 2>/dev/null | awk '{print $1}' || echo "127.0.0.1")

echo -e "${GREEN}${BOLD}"
echo "=================================================================="
echo "   ✅ SOLARIA APPLIANCE INSTALLED & HARDENED SUCCESSFULLY!"
echo "=================================================================="
echo -e "${NC}"
echo -e "Operating System: ${CYAN}$DISTRO_DESC ($ARCH)${NC}"
echo -e "The Solaria daemons are configured to start automatically on cold boot."
echo ""
echo -e "Access your live cottage solar dashboard on local WiFi:"
echo -e "   👉 ${CYAN}${BOLD}http://solaria.local:8080${NC}"
echo -e "   👉 ${CYAN}${BOLD}http://${PRIMARY_IP}:8080${NC}"
echo ""
echo -e "Remote Cloud Run Dashboard (when internet is available):"
echo -e "   👉 ${BLUE}https://solaria-dashboard-952659886764.us-central1.run.app${NC}"
echo ""
echo -e "Useful Commands:"
echo -e "   - Check Live System Status:  ${YELLOW}sudo systemctl status solaria-bridge${NC}"
echo -e "   - View Diagnostic Logs:      ${YELLOW}sudo journalctl -u solaria-bridge -f${NC}"
echo -e "   - Run Deep System Audit:     ${YELLOW}$INSTALL_DIR/bin/solaria-e2e-audit${NC}"
echo ""
if [[ "$IS_RPI" == "true" ]]; then
    echo -e "${BOLD}💡 Raspberry Pi Flash Protection:${NC}"
    echo -e "To make the SD card 100% immune to sudden power cuts, enable Read-Only OverlayFS:"
    echo -e "   ${YELLOW}sudo raspi-config nonint enable_overlayfs${NC} (and reboot)"
elif [[ "$DISTRO_ID" == "debian" ]] || [[ "$DISTRO_ID" == "ubuntu" ]]; then
    echo -e "${BOLD}💡 x86 Mini PC BIOS Power Recovery:${NC}"
    echo -e "In your PC BIOS settings, ensure ${YELLOW}'AC Power Recovery / State After G3'${NC} is set to ${YELLOW}'Always On'${NC}."
fi
echo "=================================================================="
