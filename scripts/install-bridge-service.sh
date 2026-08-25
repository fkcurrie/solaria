#!/usr/bin/env bash
set -euo pipefail

# Solaria Bridge Service Installer
# Installs and enables solaria-bridge systemd service with auto-restart resilience

PROJECT_DIR="/home/fcurrie/Projects/solar-testing"
SERVICE_FILE="${PROJECT_DIR}/deploy/solaria-bridge.service"
SYSTEMD_DEST="/etc/systemd/system/solaria-bridge.service"

echo "=== Solaria Bridge Service Setup ==="

# 1. Compile latest bridge binary
echo "Building bin/solaria-bridge..."
export PATH="$PATH:/home/fcurrie/.local/go/bin"
cd "${PROJECT_DIR}"
go build -o bin/solaria-bridge ./cmd/bridge

# 2. Check if running with sudo / root access for systemd
if command -v systemctl &>/dev/null; then
    if [ "$EUID" -eq 0 ]; then
        cp "${SERVICE_FILE}" "${SYSTEMD_DEST}"
        systemctl daemon-reload
        systemctl enable solaria-bridge
        systemctl restart solaria-bridge
        echo "✅ solaria-bridge systemd service installed and started."
        systemctl status solaria-bridge --no-pager
    else
        echo "ℹ️ Run with sudo to register systemd unit: sudo cp ${SERVICE_FILE} ${SYSTEMD_DEST} && sudo systemctl enable --now solaria-bridge"
    fi
else
    echo "⚠️ systemctl not available on this host."
fi

echo "=== Setup complete ==="
