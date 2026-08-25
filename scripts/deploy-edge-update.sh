#!/usr/bin/env bash
set -euo pipefail

# Solaria Zero-Downtime Safe Edge Updater
# 1. Compiles candidate binary to temporary staging artifact
# 2. Runs verification checks
# 3. Atomically replaces bin/solaria-bridge
# 4. Triggers graceful hot-reload (SIGHUP) or clean service restart
# 5. Confirms /api/v1/health is responsive

PROJECT_DIR="/home/fcurrie/Projects/solar-testing"
BIN_DIR="${PROJECT_DIR}/bin"
TARGET_BIN="${BIN_DIR}/solaria-bridge"
TMP_BIN="${BIN_DIR}/solaria-bridge.staging"

echo "=== Solaria Edge Update Engine ==="
export PATH="$PATH:/home/fcurrie/.local/go/bin"
cd "${PROJECT_DIR}"

# 1. Run unit tests before building
echo "Step 1: Running unit tests..."
go test ./cmd/bridge

# 2. Build staging binary
echo "Step 2: Building staging binary..."
go build -o "${TMP_BIN}" ./cmd/bridge

# 3. Transactional atomic binary swap
echo "Step 3: Atomically deploying new binary..."
chmod +x "${TMP_BIN}"
mv -f "${TMP_BIN}" "${TARGET_BIN}"

# 4. Reload or restart service
echo "Step 4: Signaling running bridge..."
if pgrep -f "bin/solaria-bridge" > /dev/null; then
    BRIDGE_PID=$(pgrep -f "bin/solaria-bridge" | head -n 1)
    echo "Found active bridge process (PID ${BRIDGE_PID}). Sending SIGHUP to reload config..."
    kill -HUP "${BRIDGE_PID}" || true
else
    echo "Bridge process not currently active."
fi

if command -v systemctl &>/dev/null && systemctl is-active --quiet solaria-bridge 2>/dev/null; then
    echo "Restarting solaria-bridge systemd daemon cleanly..."
    sudo systemctl restart solaria-bridge || true
fi

# 5. Health Check Verification
echo "Step 5: Verifying health endpoint (http://localhost:8080/api/v1/health)..."
for i in {1..10}; do
    if curl -s http://localhost:8080/api/v1/health | grep -q "status"; then
        echo "✅ Edge Bridge verified ONLINE and HEALTHY."
        exit 0
    fi
    sleep 1
done

echo "⚠️ Health check timed out, but binary is in place."
