#!/usr/bin/env bash
set -euo pipefail

# Deploy the rulegate Web frontend to Cloud Run.
# Usage: ./scripts/deploy-web.sh
#
# Required environment variables:
#   GCP_PROJECT_ID   - GCP project ID
#   API_URL          - Internal URL of the rulegate-api Cloud Run service

: "${GCP_PROJECT_ID:?GCP_PROJECT_ID is required}"
: "${API_URL:?API_URL is required (e.g. https://rulegate-api-xxxxx.run.app)}"

REGION="${GCP_REGION:-us-central1}"
SERVICE_NAME="rulegate-web"
IMAGE="${REGION}-docker.pkg.dev/${GCP_PROJECT_ID}/cloud-run/${SERVICE_NAME}"

echo "==> Building container image..."
gcloud builds submit \
  --tag "${IMAGE}" \
  --project "${GCP_PROJECT_ID}" \
  --region "${REGION}" \
  ./frontend

echo "==> Deploying to Cloud Run..."
gcloud run deploy "${SERVICE_NAME}" \
  --image "${IMAGE}" \
  --region "${REGION}" \
  --project "${GCP_PROJECT_ID}" \
  --platform managed \
  --allow-unauthenticated \
  --set-env-vars "API_URL=${API_URL}" \
  --memory 256Mi \
  --cpu 1 \
  --timeout 30 \
  --max-instances 3 \
  --port 3000

echo "==> Done. Service URL:"
gcloud run services describe "${SERVICE_NAME}" \
  --region "${REGION}" \
  --project "${GCP_PROJECT_ID}" \
  --format 'value(status.url)'
