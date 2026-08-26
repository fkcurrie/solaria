#!/usr/bin/env bash
# ==============================================================================
# Solaria: Setup & Installation Wizard
# ==============================================================================
# Usage:
#   Interactive:     ./setup.sh
#   One-Liner:       curl -fsSL https://raw.githubusercontent.com/fkcurrie/solaria/main/setup.sh | bash
#   Agent Mode:      ./setup.sh --agent-mode
#   Non-Interactive: ./setup.sh --non-interactive
#   Quick Start:     ./setup.sh --start-bridge
#   Deploy Cloud:    ./setup.sh --deploy-cloud
# ==============================================================================
set -e

# ANSI Color Codes (disabled when stdout is not a tty or in json mode)
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

# Parse Flags
AGENT_MODE=false
NON_INTERACTIVE=false
CHECK_ONLY=false
INSTALL_DEPS=false
START_BRIDGE=false
DEPLOY_CLOUD=false
SHOW_HELP=false
STORAGE_MODE_FLAG=""

while [[ $# -gt 0 ]]; do
    case "$1" in
        --agent-mode|--json)
            AGENT_MODE=true
            NON_INTERACTIVE=true
            shift
            ;;
        --non-interactive|-y|--yes)
            NON_INTERACTIVE=true
            shift
            ;;
        --storage-mode)
            STORAGE_MODE_FLAG="$2"
            shift 2
            ;;
        --check-only)
            CHECK_ONLY=true
            shift
            ;;
        --install-deps)
            INSTALL_DEPS=true
            shift
            ;;
        --start-bridge|--bridge)
            START_BRIDGE=true
            NON_INTERACTIVE=true
            shift
            ;;
        --deploy-cloud|--deploy)
            DEPLOY_CLOUD=true
            NON_INTERACTIVE=true
            shift
            ;;
        --help|-h)
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
Solaria Installation & Setup Wizard

USAGE:
    ./setup.sh [OPTIONS]
    curl -fsSL https://raw.githubusercontent.com/fkcurrie/solaria/main/setup.sh | bash

OPTIONS:
    --agent-mode, --json            Run in machine-readable JSON mode for AI agents (Gemini, Claude, Antigravity)
    --non-interactive, -y           Execute configuration using environment variables and sensible defaults
    --storage-mode <mode>           Storage destination: local (CSV on node), bigquery (Cloud), or both (default: both)
    --start-bridge                  Immediately start the Go Edge Bridge and Web Dashboard on http://localhost:8080
    --deploy-cloud                  Deploy the Go cloud ingestion service and dashboard directly to Google Cloud Run
    --install-deps                  Automatically install Go and system dependencies if missing
    --check-only                    Run environment diagnostics and exit
    --help, -h                      Show this help message

ENVIRONMENT VARIABLES (Optional overrides):
    STORAGE_MODE            Storage mode: local, bigquery, or both (default: both)
    GCP_PROJECT             Google Cloud Project ID (default: solaria-solar)
    BIGQUERY_DATASET        BigQuery Dataset (default: solaria)
    BIGQUERY_TABLE          BigQuery Table (default: telemetry)
    SITE_NAME               Installation Site Name (default: 1296 Wren Lake Drive, Dorset, ON)
    SITE_LATITUDE           Site Latitude (default: 45.186)
    SITE_LONGITUDE          Site Longitude (default: -78.863)
    PANEL_RATED_WATTS       Array Peak Capacity in Watts (default: 400.0)
    SOLARIA_API_TOKEN       Bearer token for cloud ingestion (default: solaria_cottage_secret_token_2026)
    SOLARIA_CLOUD_ENDPOINT  Cloud Run Ingestion URL
EOF
    exit 0
fi

# Ensure workspace / repo directory exists
if [ ! -f "go.mod" ]; then
    if [ -d "solaria" ]; then
        cd solaria
    elif [ -d "solar-testing" ]; then
        cd solar-testing
    else
        echo -e "${CYAN}Cloning Solaria repository into ./solaria...${NC}"
        git clone https://github.com/fkcurrie/solaria.git solaria
        cd solaria
    fi
fi

# Detect Environment
OS_TYPE="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH_TYPE="$(uname -m)"
case "$ARCH_TYPE" in
    x86_64)  GO_ARCH="amd64" ;;
    aarch64|arm64) GO_ARCH="arm64" ;;
    armv7l)  GO_ARCH="armv6l" ;;
    *)       GO_ARCH="amd64" ;;
esac

# Check Go in standard paths and user-local paths
export PATH="$HOME/.local/go/bin:$HOME/go/bin:/usr/local/go/bin:$PATH"
HAS_GO=false
GO_VERSION_STR="not found"
if command -v go &>/dev/null; then
    HAS_GO=true
    GO_VERSION_STR=$(go version 2>&1)
fi

# Check gcloud
HAS_GCLOUD=false
GCLOUD_ACCOUNT="not logged in"
if command -v gcloud &>/dev/null; then
    HAS_GCLOUD=true
    GCLOUD_ACCOUNT=$(gcloud config get-value account 2>/dev/null || echo "not logged in")
fi

# Check Bluetooth Tools
HAS_BLUETOOTH=false
if command -v bluetoothctl &>/dev/null; then
    HAS_BLUETOOTH=true
fi

# Load existing .env if present
if [ -f .env ]; then
    # shellcheck disable=SC1091
    source .env 2>/dev/null || true
fi

# Apply storage mode flag if provided
if [ -n "$STORAGE_MODE_FLAG" ]; then
    STORAGE_MODE="$STORAGE_MODE_FLAG"
fi
STORAGE_MODE="${STORAGE_MODE:-both}"

# Agent Mode JSON Output
if [ "$AGENT_MODE" = true ]; then
    cat << JSON_EOF
{
  "status": "ready",
  "environment": {
    "os": "${OS_TYPE}",
    "arch": "${ARCH_TYPE}",
    "go_installed": ${HAS_GO},
    "go_version": "${GO_VERSION_STR}",
    "gcloud_installed": ${HAS_GCLOUD},
    "gcloud_account": "${GCLOUD_ACCOUNT}",
    "bluetooth_available": ${HAS_BLUETOOTH}
  },
  "storage_options": {
    "current_mode": "${STORAGE_MODE}",
    "supported_modes": ["local", "bigquery", "both"],
    "descriptions": {
      "local": "Store telemetry strictly on local node in CSV/JSON logs (no cloud transmission)",
      "bigquery": "Stream telemetry to Google BigQuery via Cloud Run ingestion",
      "both": "Store locally in CSV logs and stream to Google BigQuery"
    }
  },
  "site_defaults": {
    "site_name": "${SITE_NAME:-1296 Wren Lake Drive, Dorset, ON}",
    "latitude": ${SITE_LATITUDE:-45.186},
    "longitude": ${SITE_LONGITUDE:--78.863},
    "rated_watts": ${PANEL_RATED_WATTS:-400.0}
  },
  "cloud_defaults": {
    "gcp_project": "${GCP_PROJECT:-solaria-solar}",
    "dataset": "${BIGQUERY_DATASET:-solaria}",
    "table": "${BIGQUERY_TABLE:-telemetry}",
    "cloud_endpoint": "${SOLARIA_CLOUD_ENDPOINT:-https://solaria.example.com/api/v1/telemetry}"
  },
  "suggested_actions": [
    {"command": "./setup.sh --start-bridge", "description": "Run local Go bridge & dashboard on http://localhost:8080"},
    {"command": "./setup.sh --storage-mode local --start-bridge", "description": "Run in local-only storage mode"},
    {"command": "./setup.sh --deploy-cloud", "description": "Deploy to Google Cloud Run"},
    {"command": "go test ./...", "description": "Run unit and integration test suite"}
  ]
}
JSON_EOF
    exit 0
fi

# Header
echo -e "${CYAN}======================================================================${NC}"
echo -e "${BOLD}Solaria: Renogy Solar & Atmospheric Intelligence Platform${NC}"
echo -e "${CYAN}======================================================================${NC}"
echo -e "Configuring local edge bridge, Bluetooth supervisor, and telemetry storage.\n"

# Step 1: Prerequisites
echo -e "${CYAN}[1/4] Checking Prerequisites...${NC}"
if [ "$HAS_GO" = true ]; then
    echo -e "  [OK] Go: ${GO_VERSION_STR}"
else
    echo -e "  [!] Go 1.21+ not found."
    if [ "$INSTALL_DEPS" = true ] || [ "$NON_INTERACTIVE" = false ]; then
        if [ "$NON_INTERACTIVE" = false ]; then
            read -rp "  Would you like to install Go automatically? [Y/n]: " INSTALL_GO_PROMPT
        else
            INSTALL_GO_PROMPT="Y"
        fi
        if [[ "$INSTALL_GO_PROMPT" =~ ^[Yy]?$ ]]; then
            echo -e "  Downloading official Go binary for ${OS_TYPE}-${GO_ARCH}..."
            mkdir -p "$HOME/.local"
            curl -fsSL "https://go.dev/dl/go1.23.6.${OS_TYPE}-${GO_ARCH}.tar.gz" | tar -C "$HOME/.local" -xz
            export PATH="$HOME/.local/go/bin:$PATH"
            echo -e "  [OK] Go installed: $(go version)"
            HAS_GO=true
        fi
    fi
fi

if [ "$HAS_GCLOUD" = true ]; then
    echo -e "  [OK] Google Cloud SDK (Account: ${GCLOUD_ACCOUNT})"
else
    echo -e "  [!] gcloud SDK not found (optional, required only for Cloud Run deployment)"
fi

if [ "$HAS_BLUETOOTH" = true ]; then
    echo -e "  [OK] Linux BlueZ / Bluetooth Tools detected"
fi

if [ "$CHECK_ONLY" = true ]; then
    echo -e "\nDiagnostic check complete."
    exit 0
fi

# Step 2: Configuration
echo -e "\n${CYAN}[2/4] Storage & Site Configuration...${NC}"

if [ "$NON_INTERACTIVE" = true ]; then
    STORAGE_MODE="${STORAGE_MODE:-both}"
    SITE_NAME="${SITE_NAME:-1296 Wren Lake Drive, Dorset, ON}"
    SITE_LATITUDE="${SITE_LATITUDE:-45.186}"
    SITE_LONGITUDE="${SITE_LONGITUDE:--78.863}"
    PANEL_RATED_WATTS="${PANEL_RATED_WATTS:-400.0}"
    GCP_PROJECT="${GCP_PROJECT:-solaria-solar}"
    BIGQUERY_DATASET="${BIGQUERY_DATASET:-solaria}"
    BIGQUERY_TABLE="${BIGQUERY_TABLE:-telemetry}"
    SOLARIA_API_TOKEN="${SOLARIA_API_TOKEN:-solaria_cottage_secret_token_2026}"
    SOLARIA_CLOUD_ENDPOINT="${SOLARIA_CLOUD_ENDPOINT:-https://solaria.example.com/api/v1/telemetry}"
else
    echo -e "  Storage Destination Options:"
    echo -e "    1) Local node only (CSV logs on Raspberry Pi/Linux, no cloud)"
    echo -e "    2) Google BigQuery only (Streamed via Cloud Run)"
    echo -e "    3) Both (Local CSV logs + Google BigQuery streaming) [Default]"
    read -rp "  Select storage destination [1-3, default: 3]: " STORAGE_CHOICE
    case "${STORAGE_CHOICE:-3}" in
        1) STORAGE_MODE="local" ;;
        2) STORAGE_MODE="bigquery" ;;
        *) STORAGE_MODE="both" ;;
    esac
    echo -e "  Selected storage mode: ${GREEN}${STORAGE_MODE}${NC}"

    DEFAULT_SITE_NAME="${SITE_NAME:-1296 Wren Lake Drive, Dorset, ON}"
    read -rp "  Site Name [$DEFAULT_SITE_NAME]: " INPUT_SITE_NAME
    SITE_NAME="${INPUT_SITE_NAME:-$DEFAULT_SITE_NAME}"

    DEFAULT_LAT="${SITE_LATITUDE:-45.186}"
    read -rp "  Site Latitude [$DEFAULT_LAT]: " INPUT_LAT
    SITE_LATITUDE="${INPUT_LAT:-$DEFAULT_LAT}"

    DEFAULT_LON="${SITE_LONGITUDE:--78.863}"
    read -rp "  Site Longitude [$DEFAULT_LON]: " INPUT_LON
    SITE_LONGITUDE="${INPUT_LON:-$DEFAULT_LON}"

    DEFAULT_WATTS="${PANEL_RATED_WATTS:-400.0}"
    read -rp "  Array Capacity in Watts (e.g., 400 for 4x100W) [$DEFAULT_WATTS]: " INPUT_WATTS
    PANEL_RATED_WATTS="${INPUT_WATTS:-$DEFAULT_WATTS}"

    if [ "$STORAGE_MODE" != "local" ]; then
        DEFAULT_PROJECT="${GCP_PROJECT:-solaria-solar}"
        read -rp "  GCP Project ID [$DEFAULT_PROJECT]: " INPUT_PROJECT
        GCP_PROJECT="${INPUT_PROJECT:-$DEFAULT_PROJECT}"

        DEFAULT_DATASET="${BIGQUERY_DATASET:-solaria}"
        read -rp "  BigQuery Dataset [$DEFAULT_DATASET]: " INPUT_DATASET
        BIGQUERY_DATASET="${INPUT_DATASET:-$DEFAULT_DATASET}"

        DEFAULT_TABLE="${BIGQUERY_TABLE:-telemetry}"
        read -rp "  BigQuery Table [$DEFAULT_TABLE]: " INPUT_TABLE
        BIGQUERY_TABLE="${INPUT_TABLE:-$DEFAULT_TABLE}"

        DEFAULT_TOKEN="${SOLARIA_API_TOKEN:-solaria_cottage_secret_token_2026}"
        read -rp "  API Ingestion Token [$DEFAULT_TOKEN]: " INPUT_TOKEN
        SOLARIA_API_TOKEN="${INPUT_TOKEN:-$DEFAULT_TOKEN}"

        DEFAULT_ENDPOINT="${SOLARIA_CLOUD_ENDPOINT:-https://solaria.example.com/api/v1/telemetry}"
        read -rp "  Cloud Endpoint [$DEFAULT_ENDPOINT]: " INPUT_ENDPOINT
        SOLARIA_CLOUD_ENDPOINT="${INPUT_ENDPOINT:-$DEFAULT_ENDPOINT}"
    else
        GCP_PROJECT="${GCP_PROJECT:-solaria-solar}"
        BIGQUERY_DATASET="${BIGQUERY_DATASET:-solaria}"
        BIGQUERY_TABLE="${BIGQUERY_TABLE:-telemetry}"
        SOLARIA_API_TOKEN="${SOLARIA_API_TOKEN:-solaria_cottage_secret_token_2026}"
        SOLARIA_CLOUD_ENDPOINT="${SOLARIA_CLOUD_ENDPOINT:-https://solaria.example.com/api/v1/telemetry}"
    fi
fi

# Step 3: Write .env
echo -e "\n${CYAN}[3/4] Saving Configuration (.env)...${NC}"
cat << ENV_EOF > .env
# ==============================================================================
# Solaria Configuration File (Auto-generated by setup.sh)
# ==============================================================================
STORAGE_MODE=${STORAGE_MODE}
GCP_PROJECT=${GCP_PROJECT}
BIGQUERY_DATASET=${BIGQUERY_DATASET}
BIGQUERY_TABLE=${BIGQUERY_TABLE}
PORT=8080

SITE_NAME="${SITE_NAME}"
SITE_LATITUDE=${SITE_LATITUDE}
SITE_LONGITUDE=${SITE_LONGITUDE}
PANEL_RATED_WATTS=${PANEL_RATED_WATTS}

SOLARIA_API_TOKEN=${SOLARIA_API_TOKEN}
SOLARIA_CLOUD_ENDPOINT=${SOLARIA_CLOUD_ENDPOINT}
ENV_EOF
echo -e "  [OK] Configuration saved to ${CYAN}.env${NC}"

# Step 4: Execution / Action
echo -e "\n${CYAN}[4/4] Execution...${NC}"

if [ "$START_BRIDGE" = true ]; then
    echo -e "${GREEN}Launching Solaria Go Bridge on http://localhost:8080...${NC}"
    exec go run ./cmd/bridge
fi

if [ "$DEPLOY_CLOUD" = true ]; then
    if [ "$HAS_GCLOUD" = true ]; then
        echo -e "${GREEN}Deploying to Google Cloud Run in project ${GCP_PROJECT}...${NC}"
        gcloud config set project "$GCP_PROJECT"
        exec gcloud run deploy solaria-dashboard \
            --source . \
            --region us-central1 \
            --allow-unauthenticated \
            --set-env-vars "GCP_PROJECT=${GCP_PROJECT},SOLARIA_API_TOKEN=${SOLARIA_API_TOKEN}"
    else
        echo -e "${RED}gcloud CLI is required for Cloud Run deployment.${NC}"
        exit 1
    fi
fi

if [ "$NON_INTERACTIVE" = true ]; then
    echo -e "${GREEN}[OK] Solaria setup completed successfully.${NC}"
    echo -e "  • Storage Mode:    ${CYAN}${STORAGE_MODE}${NC}"
    echo -e "  • Start Bridge:    ${CYAN}go run ./cmd/bridge${NC} (or ${CYAN}./setup.sh --start-bridge${NC})"
    echo -e "  • Deploy Cloud:    ${CYAN}./setup.sh --deploy-cloud${NC}"
    echo -e "  • Open Dashboard:  ${CYAN}http://localhost:8080${NC}"
    exit 0
fi

# Interactive Menu
echo -e "  1) Start Local Go Bridge & Dashboard (${CYAN}http://localhost:8080${NC})"
echo -e "  2) Deploy to Google Cloud Run"
echo -e "  3) Build all binaries (bridge, edge-agent, cloud-server)"
echo -e "  4) Exit"
read -rp "  Select an option [1-4, default: 1]: " OPTION
OPTION="${OPTION:-1}"

case "$OPTION" in
    1)
        echo -e "\n${GREEN}Starting Solaria Go Bridge on http://localhost:8080...${NC}"
        exec go run ./cmd/bridge
        ;;
    2)
        if [ "$HAS_GCLOUD" = true ]; then
            echo -e "\n${GREEN}Deploying to Cloud Run...${NC}"
            gcloud config set project "$GCP_PROJECT"
            gcloud run deploy solaria-dashboard \
                --source . \
                --region us-central1 \
                --allow-unauthenticated \
                --set-env-vars "GCP_PROJECT=${GCP_PROJECT},SOLARIA_API_TOKEN=${SOLARIA_API_TOKEN}"
        else
            echo -e "\n${RED}gcloud CLI is required to deploy to Cloud Run.${NC}"
        fi
        ;;
    3)
        echo -e "\n${GREEN}Building all binaries into bin/...${NC}"
        mkdir -p bin
        go build -o bin/solaria-bridge ./cmd/bridge
        go build -o bin/solaria-edge ./cmd/edge-agent
        go build -o bin/solaria-cloud ./cmd/cloud-server
        echo -e "${GREEN}[OK] Binaries generated in bin/${NC}"
        ;;
    *)
        echo -e "\nSetup complete. Run './setup.sh' anytime."
        ;;
esac
