# 🚀 Deployment Guide: Edge Daemon & Cloud Run

This guide covers deploying the Solaria components to edge devices (Linux laptop, Raspberry Pi) and Google Cloud Platform.

---

## 1. Environment Variables Reference

Create a `.env` file in the project root with the following parameters:

```bash
# Google Cloud Platform
GCP_PROJECT=solaria-solar
BIGQUERY_DATASET=solaria
BIGQUERY_TABLE=telemetry
PORT=8080

# Physical Solar Site
SITE_NAME="1296 Wren Lake Drive, Dorset, ON"
SITE_LATITUDE=45.186
SITE_LONGITUDE=-78.863
PANEL_RATED_WATTS=400.0

# Ingestion Security & Cloud Ingestion URL
SOLARIA_API_TOKEN=solaria_cottage_secret_token_2026
SOLARIA_CLOUD_ENDPOINT=https://solaria-dashboard-952659886764.us-central1.run.app/api/v1/telemetry
```

---

## 2. Deploying to Google Cloud Run

To deploy the cloud dashboard and BigQuery ingestion endpoint:

```bash
# 1. Authenticate with Google Cloud SDK
gcloud auth login
gcloud config set project solaria-solar

# 2. Deploy Cloud Run service from source
gcloud run deploy solaria-dashboard \
  --source . \
  --region us-central1 \
  --allow-unauthenticated \
  --set-env-vars "GCP_PROJECT=solaria-solar,SOLARIA_API_TOKEN=solaria_cottage_secret_token_2026"
```

The service will be live at:
`https://solaria-dashboard-952659886764.us-central1.run.app`

---

## 3. Local Gateway & Edge Setup

### Option A: Interactive Web Gateway (`cmd/bridge`)
Runs the local dashboard on `http://localhost:8080` with Web Bluetooth auto-connection and resilience watchdog:

```bash
go run ./cmd/bridge
```

### Option B: Headless Systemd Service on Raspberry Pi / Linux Gateway
To run Solaria as an unattended background service on a Raspberry Pi or Linux gateway:

1. Build the edge agent:
```bash
go build -o /usr/local/bin/solaria-edge ./cmd/edge-agent
```

2. Install systemd service unit:
```bash
sudo cp edge/renogy_edge.service /etc/systemd/system/renogy_edge.service
sudo systemctl daemon-reload
sudo systemctl enable --now renogy_edge.service
```

3. Check service status & logs:
```bash
sudo systemctl status renogy_edge.service
journalctl -u renogy_edge.service -f
```
