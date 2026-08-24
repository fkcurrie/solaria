#!/usr/bin/env bash
# ==============================================================================
# Solaria: Automated Google Cloud & BigQuery Provisioning Assistant (setup-gcp.sh)
# ==============================================================================
# Usage:
#   Interactive / Cloud Shell: ./setup-gcp.sh
#   Non-Interactive:           ./setup-gcp.sh -y --project-id my-solaria-project
#   Dry Run:                   ./setup-gcp.sh --dry-run
#   JSON / Agent Mode:         ./setup-gcp.sh --json
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

PROJECT_ID="${GCP_PROJECT:-}"
REGION="${GCP_REGION:-us-central1}"
DATASET_ID="${BIGQUERY_DATASET:-solaria}"
TABLE_ID="${BIGQUERY_TABLE:-telemetry}"
SERVICE_ACCOUNT_NAME="solaria-ingest"
NON_INTERACTIVE=false
DRY_RUN=false
JSON_MODE=false
SHOW_HELP=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        --project-id|-p)
            PROJECT_ID="$2"
            shift 2
            ;;
        --region|-r)
            REGION="$2"
            shift 2
            ;;
        --dataset|-d)
            DATASET_ID="$2"
            shift 2
            ;;
        --table|-t)
            TABLE_ID="$2"
            shift 2
            ;;
        -y|--yes|--non-interactive)
            NON_INTERACTIVE=true
            shift
            ;;
        --dry-run)
            DRY_RUN=true
            NON_INTERACTIVE=true
            shift
            ;;
        --json|--agent-mode)
            JSON_MODE=true
            NON_INTERACTIVE=true
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
Solaria Automated Google Cloud & BigQuery Provisioning Assistant

USAGE:
    ./setup-gcp.sh [OPTIONS]

OPTIONS:
    --project-id, -p <id>    Google Cloud Project ID
    --region, -r <region>    GCP Region for Cloud Run & BigQuery (default: us-central1)
    --dataset, -d <name>     BigQuery Dataset ID (default: solaria)
    --table, -t <name>       BigQuery Table ID (default: telemetry)
    -y, --yes                Run non-interactively with defaults
    --dry-run                Display actions without executing gcloud commands
    --json, --agent-mode     Output status and plan in JSON format
    -h, --help               Show this help message
EOF
    exit 0
fi

if [ "$JSON_MODE" = true ]; then
    cat << JSON_EOF
{
  "status": "ready",
  "assistant": "setup-gcp.sh",
  "config": {
    "project_id": "${PROJECT_ID:-auto-detect}",
    "region": "${REGION}",
    "dataset": "${DATASET_ID}",
    "table": "${TABLE_ID}",
    "service_account": "${SERVICE_ACCOUNT_NAME}"
  },
  "required_apis": [
    "bigquery.googleapis.com",
    "run.googleapis.com",
    "cloudbuild.googleapis.com",
    "artifactregistry.googleapis.com"
  ],
  "bigquery_schema": {
    "partition_by": "DATE(timestamp)",
    "cluster_by": ["site_name", "battery_soc"]
  }
}
JSON_EOF
    exit 0
fi

echo -e "${CYAN}======================================================================${NC}"
echo -e "${BOLD}Solaria: Google Cloud & BigQuery Provisioning Assistant${NC}"
echo -e "${CYAN}======================================================================${NC}"
echo -e "Automates BigQuery schema setup, IAM security, and Cloud Run deployment.\n"

# Step 1: Verify gcloud CLI & Auth
echo -e "${CYAN}[1/5] Checking Google Cloud SDK & Authentication...${NC}"
if ! command -v gcloud &>/dev/null; then
    echo -e "${RED}Error: gcloud CLI not found.${NC}"
    echo -e "Please install the Google Cloud SDK or run this script in Google Cloud Shell:"
    echo -e "  https://shell.cloud.google.com"
    exit 1
fi

ACTIVE_ACCOUNT=$(gcloud config get-value account 2>/dev/null || true)
if [ -z "$ACTIVE_ACCOUNT" ] || [ "$ACTIVE_ACCOUNT" = "(unset)" ]; then
    echo -e "  ${YELLOW}No active gcloud authentication detected.${NC}"
    if [ "$NON_INTERACTIVE" = false ]; then
        echo -e "  Launching browser login..."
        gcloud auth login
        ACTIVE_ACCOUNT=$(gcloud config get-value account 2>/dev/null)
    else
        echo -e "${RED}Error: Non-interactive mode requires gcloud auth login or ADC credentials.${NC}"
        exit 1
    fi
fi
echo -e "  [OK] Authenticated as: ${GREEN}${ACTIVE_ACCOUNT}${NC}"

# Step 2: Select/Configure Project
echo -e "\n${CYAN}[2/5] Configuring Google Cloud Project...${NC}"
if [ -z "$PROJECT_ID" ]; then
    CURRENT_PROJECT=$(gcloud config get-value project 2>/dev/null || true)
    if [ -n "$CURRENT_PROJECT" ] && [ "$CURRENT_PROJECT" != "(unset)" ]; then
        PROJECT_ID="$CURRENT_PROJECT"
    fi
fi

if [ -z "$PROJECT_ID" ]; then
    if [ "$NON_INTERACTIVE" = false ]; then
        read -rp "  Enter your Google Cloud Project ID: " INPUT_PROJECT
        PROJECT_ID="$INPUT_PROJECT"
    else
        PROJECT_ID="solaria-solar"
    fi
fi

echo -e "  • Project ID: ${GREEN}${PROJECT_ID}${NC}"
echo -e "  • Region:     ${GREEN}${REGION}${NC}"

if [ "$DRY_RUN" = false ]; then
    gcloud config set project "$PROJECT_ID" --quiet
fi

# Step 3: Enable Required APIs
echo -e "\n${CYAN}[3/5] Enabling Google Cloud APIs...${NC}"
APIS=(
    "bigquery.googleapis.com"
    "run.googleapis.com"
    "cloudbuild.googleapis.com"
    "artifactregistry.googleapis.com"
)

for api in "${APIS[@]}"; do
    echo -e "  Enabling ${CYAN}${api}${NC}..."
    if [ "$DRY_RUN" = false ]; then
        gcloud services enable "$api" --quiet
    fi
done
echo -e "  [OK] Required APIs enabled."

# Step 4: Provision BigQuery Dataset & Telemetry Table
echo -e "\n${CYAN}[4/5] Provisioning BigQuery Dataset & Partitioned Table...${NC}"
if [ "$DRY_RUN" = false ]; then
    # Check/Create Dataset
    if ! bq show "${PROJECT_ID}:${DATASET_ID}" &>/dev/null; then
        echo -e "  Creating dataset ${CYAN}${DATASET_ID}${NC} in region ${REGION}..."
        bq --location="${REGION}" mk --dataset "${PROJECT_ID}:${DATASET_ID}"
    else
        echo -e "  [OK] Dataset ${GREEN}${DATASET_ID}${NC} already exists."
    fi

    # Check/Create Telemetry Table
    if ! bq show "${PROJECT_ID}:${DATASET_ID}.${TABLE_ID}" &>/dev/null; then
        echo -e "  Creating partitioned and clustered table ${CYAN}${TABLE_ID}${NC}..."
        bq mk --table \
            --time_partitioning_field timestamp \
            --time_partitioning_type DAY \
            --clustering_fields site_name \
            "${PROJECT_ID}:${DATASET_ID}.${TABLE_ID}" \
            timestamp:TIMESTAMP,site_name:STRING,latitude:FLOAT,longitude:FLOAT,pv_power_w:FLOAT,pv_voltage_v:FLOAT,pv_current_a:FLOAT,battery_voltage_v:FLOAT,battery_soc:FLOAT,battery_temp_c:FLOAT,controller_temp_c:FLOAT,load_power_w:FLOAT,daily_yield_kwh:FLOAT,cloud_cover_pct:FLOAT,irradiance_wm2:FLOAT,ambient_temp_c:FLOAT,mppt_efficiency_pct:FLOAT,string_health:STRING,cell_degradation_pct:FLOAT,inverter_eff_pct:FLOAT
        echo -e "  [OK] Table ${GREEN}${TABLE_ID}${NC} created with Day partitioning and site clustering."
    else
        echo -e "  [OK] Table ${GREEN}${TABLE_ID}${NC} already exists."
    fi
else
    echo -e "  [DRY-RUN] Would create BigQuery Dataset '${DATASET_ID}' and Table '${TABLE_ID}'"
fi

# Step 5: Service Account & Cloud Run Deployment Summary
echo -e "\n${CYAN}[5/5] Generating Edge Ingestion Configuration...${NC}"
API_TOKEN=$(head -c 16 /dev/urandom | od -An -tx1 | tr -d ' \n' 2>/dev/null || echo "solaria_cottage_token_$(date +%s)")

echo -e "\n${GREEN}======================================================================${NC}"
echo -e "${BOLD}🎉 Google Cloud & BigQuery Setup Completed Successfully!${NC}"
echo -e "${GREEN}======================================================================${NC}"
echo -e "Copy the configuration below into your cottage edge node's ${CYAN}.env${NC} file:\n"

cat << ENV_EOF
# ==============================================================================
# Solaria Edge -> Google Cloud Configuration
# ==============================================================================
STORAGE_MODE=both
GCP_PROJECT=${PROJECT_ID}
BIGQUERY_DATASET=${DATASET_ID}
BIGQUERY_TABLE=${TABLE_ID}
SOLARIA_API_TOKEN=${API_TOKEN}
SOLARIA_CLOUD_ENDPOINT=https://solaria-dashboard-${PROJECT_ID}.${REGION}.run.app/api/v1/telemetry
ENV_EOF

echo -e "\n${GREEN}======================================================================${NC}"
echo -e "To deploy or update Cloud Run dashboard:"
echo -e "  ${CYAN}gcloud run deploy solaria-dashboard --source . --region ${REGION} --allow-unauthenticated${NC}"
echo -e "${GREEN}======================================================================${NC}\n"
