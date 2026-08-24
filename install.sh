#!/usr/bin/env bash
# ==============================================================================
# Solaria: Autonomous Linux & Raspberry Pi Installer (install.sh)
# ==============================================================================
# Usage:
#   One-Liner:       curl -fsSL https://raw.githubusercontent.com/fkcurrie/solaria/main/install.sh | bash
#   Interactive:     ./install.sh
#   Non-Interactive: ./install.sh -y
#   Dry Run:         ./install.sh --dry-run
#   No Systemd:      ./install.sh --no-service
# ==============================================================================
set -e

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

NON_INTERACTIVE=false
DRY_RUN=false
INSTALL_SERVICE=true
SHOW_HELP=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        -y|--yes|--non-interactive)
            NON_INTERACTIVE=true
            shift
            ;;
        --dry-run)
            DRY_RUN=true
            NON_INTERACTIVE=true
            shift
            ;;
        --no-service)
            INSTALL_SERVICE=false
            shift
            ;;
        -h|--help)
            SHOW_HELP=true
            shift
            ;;
        *)
            shift
            ;;
    esac
done

if [ "$SHOW_HELP" = true ]; then
    cat << 'EOF'
Solaria Autonomous Linux & Raspberry Pi Installer

USAGE:
    ./install.sh [OPTIONS]
    curl -fsSL https://raw.githubusercontent.com/fkcurrie/solaria/main/install.sh | bash

OPTIONS:
    -y, --yes, --non-interactive  Run without prompting, applying defaults
    --dry-run                     Check environment and display install plan without changes
    --no-service                  Compile binaries but skip systemd service installation
    -h, --help                    Show this help message
EOF
    exit 0
fi

echo -e "${CYAN}======================================================================${NC}"
echo -e "${BOLD}Solaria: Autonomous Linux & Raspberry Pi Installer${NC}"
echo -e "${CYAN}======================================================================${NC}"
echo -e "Target Hardware: Renogy Rover 20A MPPT, 170Ah LiFePO4, 400W 2S2P Array\n"

# Step 1: Detect OS & Architecture
echo -e "${CYAN}[1/6] Detecting System Architecture & OS...${NC}"
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
    x86_64)         GO_ARCH="amd64" ;;
    aarch64|arm64)  GO_ARCH="arm64" ;;
    armv7l|armv6l)  GO_ARCH="armv6l" ;;
    *)              GO_ARCH="amd64" ;;
esac

echo -e "  • OS Platform:    ${GREEN}${OS}${NC}"
echo -e "  • CPU Arch:       ${GREEN}${ARCH}${NC} (Go target: ${GO_ARCH})"

# Detect Distro
DISTRO="unknown"
if [ -f /etc/os-release ]; then
    # shellcheck disable=SC1091
    source /etc/os-release
    DISTRO="${ID:-unknown}"
    DISTRO_NAME="${PRETTY_NAME:-$DISTRO}"
    echo -e "  • Linux Distro:   ${GREEN}${DISTRO_NAME}${NC}"
fi

# Step 2: System Package Manager
echo -e "\n${CYAN}[2/6] Verifying System Dependencies...${NC}"
SUDO=""
if [ "$(id -u)" -ne 0 ]; then
    if command -v sudo &>/dev/null; then
        SUDO="sudo"
    fi
fi

install_pkg() {
    local pkgs="$*"
    if [ "$DRY_RUN" = true ]; then
        echo -e "  [DRY-RUN] Would install: ${pkgs}"
        return 0
    fi

    if [ "$DISTRO" = "debian" ] || [ "$DISTRO" = "ubuntu" ] || [ "$DISTRO" = "raspbian" ]; then
        echo -e "  Installing packages on Debian/Ubuntu/Raspberry Pi OS: ${pkgs}..."
        $SUDO apt-get update -qq && $SUDO apt-get install -y -qq $pkgs
    elif [ "$DISTRO" = "fedora" ] || [ "$DISTRO" = "rhel" ] || [ "$DISTRO" = "centos" ]; then
        echo -e "  Installing packages on Fedora/RHEL: ${pkgs}..."
        $SUDO dnf install -y $pkgs
    elif [ "$DISTRO" = "arch" ] || [ "$DISTRO" = "manjaro" ]; then
        echo -e "  Installing packages on Arch Linux: ${pkgs}..."
        $SUDO pacman -Sy --noconfirm $pkgs
    elif [ "$DISTRO" = "alpine" ]; then
        echo -e "  Installing packages on Alpine: ${pkgs}..."
        $SUDO apk add $pkgs
    else
        echo -e "  ${YELLOW}Unknown package manager. Please ensure bluez, curl, git, and libcap2-bin are installed.${NC}"
    fi
}

NEEDED_PKGS=""
command -v curl &>/dev/null || NEEDED_PKGS="$NEEDED_PKGS curl"
command -v git &>/dev/null || NEEDED_PKGS="$NEEDED_PKGS git"
command -v bluetoothctl &>/dev/null || NEEDED_PKGS="$NEEDED_PKGS bluez"
command -v avahi-daemon &>/dev/null || NEEDED_PKGS="$NEEDED_PKGS avahi-daemon"

if [ -n "$NEEDED_PKGS" ]; then
    echo -e "  Installing missing packages:${YELLOW}${NEEDED_PKGS}${NC}"
    install_pkg $NEEDED_PKGS
else
    echo -e "  [OK] System packages verified (bluez, avahi, curl, git)"
fi

# Step 3: Go Toolchain
echo -e "\n${CYAN}[3/6] Verifying Go Toolchain...${NC}"
export PATH="$HOME/.local/go/bin:$HOME/go/bin:/usr/local/go/bin:$PATH"

if command -v go &>/dev/null; then
    echo -e "  [OK] Go is installed: ${GREEN}$(go version)${NC}"
else
    echo -e "  Downloading and installing Go 1.23.6 for ${OS}-${GO_ARCH}..."
    if [ "$DRY_RUN" = false ]; then
        mkdir -p "$HOME/.local"
        curl -fsSL "https://go.dev/dl/go1.23.6.${OS}-${GO_ARCH}.tar.gz" | tar -C "$HOME/.local" -xz
        export PATH="$HOME/.local/go/bin:$PATH"
        echo -e "  [OK] Go installed: ${GREEN}$(go version)${NC}"
    fi
fi

# Step 4: Workspace & Build
echo -e "\n${CYAN}[4/6] Compiling Solaria Binaries...${NC}"
INSTALL_DIR="$(pwd)"
if [ ! -f "go.mod" ]; then
    if [ -d "solaria" ]; then
        cd solaria
        INSTALL_DIR="$(pwd)"
    elif [ -d "solar-testing" ]; then
        cd solar-testing
        INSTALL_DIR="$(pwd)"
    else
        echo -e "  Cloning repository into $(pwd)/solaria..."
        if [ "$DRY_RUN" = false ]; then
            git clone https://github.com/fkcurrie/solaria.git solaria
            cd solaria
            INSTALL_DIR="$(pwd)"
        fi
    fi
fi

if [ "$DRY_RUN" = false ]; then
    mkdir -p bin
    echo -e "  Building bin/solaria-bridge..."
    go build -ldflags="-s -w" -o bin/solaria-bridge ./cmd/bridge
    echo -e "  Building bin/solaria-edge..."
    go build -ldflags="-s -w" -o bin/solaria-edge ./cmd/edge-agent
    echo -e "  Building bin/solaria-cloud..."
    go build -ldflags="-s -w" -o bin/solaria-cloud ./cmd/cloud-server
    echo -e "  [OK] Binaries compiled successfully."

    # Grant Bluetooth & Raw Socket capabilities
    if command -v setcap &>/dev/null && [ -n "$SUDO" ]; then
        echo -e "  Granting Bluetooth raw socket capabilities (CAP_NET_RAW, CAP_NET_ADMIN)..."
        $SUDO setcap 'cap_net_raw,cap_net_admin+eip' bin/solaria-bridge 2>/dev/null || true
        $SUDO setcap 'cap_net_raw,cap_net_admin+eip' bin/solaria-edge 2>/dev/null || true
    fi
fi

# Step 5: Systemd Service
echo -e "\n${CYAN}[5/6] Configuring Background Service...${NC}"
if [ "$INSTALL_SERVICE" = true ] && [ -d "/etc/systemd/system" ] && [ -n "$SUDO" ]; then
    if [ "$DRY_RUN" = false ]; then
        SERVICE_FILE="/etc/systemd/system/solaria-bridge.service"
        CURRENT_USER="$(id -un)"
        cat << SYSTEMD_EOF | $SUDO tee "$SERVICE_FILE" > /dev/null
[Unit]
Description=Solaria Renogy Solar Bridge & Atmospheric Supervisor
After=network.target bluetooth.target avahi-daemon.service
Wants=bluetooth.target

[Service]
Type=simple
User=${CURRENT_USER}
WorkingDirectory=${INSTALL_DIR}
ExecStart=${INSTALL_DIR}/bin/solaria-bridge
Restart=always
RestartSec=5s
Environment=PORT=8080
Environment=STORAGE_MODE=both
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
SYSTEMD_EOF

        $SUDO systemctl daemon-reload
        $SUDO systemctl enable solaria-bridge.service
        $SUDO systemctl restart solaria-bridge.service || true
        echo -e "  [OK] systemd service ${GREEN}solaria-bridge.service${NC} enabled and started."
    else
        echo -e "  [DRY-RUN] Would install and start /etc/systemd/system/solaria-bridge.service"
    fi
else
    echo -e "  Skipping systemd service setup."
fi

# Step 6: Summary & Instructions
echo -e "\n${CYAN}[6/6] Installation Complete!${NC}"
echo -e "${GREEN}======================================================================${NC}"
echo -e "${BOLD}🎉 Solaria is ready!${NC}"
echo -e "${GREEN}======================================================================${NC}"
echo -e "  • Local Dashboard:   ${CYAN}http://localhost:8080${NC}"
echo -e "  • mDNS Network URL:  ${CYAN}http://solaria.local:8080${NC}"
echo -e "  • Service Status:    ${CYAN}systemctl status solaria-bridge${NC}"
echo -e "  • Live Logs:         ${CYAN}journalctl -u solaria-bridge -f${NC}"
echo -e "  • Binaries:          ${INSTALL_DIR}/bin/"
echo -e "${GREEN}======================================================================${NC}\n"
