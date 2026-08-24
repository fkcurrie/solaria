# Deployment Guide

This guide covers deploying Solaria components to local edge hosts, Raspberry Pi systems, and Google Cloud Run.

## Environment Variables

Solaria reads runtime settings from `.env` or system environment variables:

```bash
# Google Cloud Platform
GCP_PROJECT=solaria-solar
BIGQUERY_DATASET=solaria
BIGQUERY_TABLE=telemetry
PORT=8080

# Site Configuration
SITE_NAME="1296 Wren Lake Drive, Dorset, ON"
SITE_LATITUDE=45.186
SITE_LONGITUDE=-78.863
PANEL_RATED_WATTS=400.0

# Ingestion Security
SOLARIA_API_TOKEN=solaria_cottage_secret_token_2026
SOLARIA_CLOUD_ENDPOINT=https://solaria-dashboard-952659886764.us-central1.run.app/api/v1/telemetry
```

## Local Edge Bridge

Start the local bridge and UI server on `http://localhost:8080`:

```bash
go run ./cmd/bridge
```

## Cloud Run Deployment

To deploy the cloud ingestion endpoint and historical analytics dashboard:

```bash
# 1. Select GCP Project
gcloud config set project solaria-solar

# 2. Deploy Cloud Run service from source
gcloud run deploy solaria-dashboard \
  --source . \
  --region us-central1 \
  --allow-unauthenticated \
  --set-env-vars "GCP_PROJECT=solaria-solar,SOLARIA_API_TOKEN=solaria_cottage_secret_token_2026"
```

## Raspberry Pi Headless Daemon (systemd)

For unattended operation on Linux/Raspberry Pi without a graphical desktop:

### 1. Build the Binary

```bash
go build -o /usr/local/bin/solaria-edge ./cmd/edge-agent
```

### 2. Install Systemd Service Unit

```bash
sudo cp edge/renogy_edge.service /etc/systemd/system/renogy_edge.service
sudo systemctl daemon-reload
sudo systemctl enable --now renogy_edge.service
```

### 3. View Logs

```bash
journalctl -u renogy_edge.service -f
```
