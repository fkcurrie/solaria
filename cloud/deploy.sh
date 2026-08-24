#!/usr/bin/env bash
set -e

PROJECT_ID="${1:-solaria-solar}"
REGION="${2:-us-central1}"
SERVICE_NAME="solaria-dashboard"

echo "=========================================================="
echo "☀️  Deploying Solaria Solar Dashboard to Cloud Run"
echo "    Project: $PROJECT_ID"
echo "    Region:  $REGION"
echo "    Service: $SERVICE_NAME"
echo "=========================================================="

gcloud config set project "$PROJECT_ID"

# Enable Cloud Run & Firestore services
gcloud services enable run.googleapis.com cloudbuild.googleapis.com firestore.googleapis.com

# Build & Deploy to Cloud Run
gcloud run deploy "$SERVICE_NAME" \
  --source . \
  --platform managed \
  --region "$REGION" \
  --allow-unauthenticated \
  --set-env-vars "GCP_PROJECT=$PROJECT_ID,SOLARIA_API_TOKEN=solaria_cottage_secret_token_2026"

echo ""
echo "✅ Solaria Cloud Run Service Deployed Successfully!"
gcloud run services describe "$SERVICE_NAME" --region "$REGION" --format="value(status.url)"
