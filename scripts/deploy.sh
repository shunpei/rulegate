#!/usr/bin/env bash
set -euo pipefail

# Deploy the rulegate API to Cloud Run.
# Usage: ./scripts/deploy.sh
#
# Required environment variables:
#   GCP_PROJECT_ID   - GCP project ID
#   GCP_REGION       - GCP region (default: us-central1)
#   RAG_CORPUS_ID    - Full RAG corpus resource name

: "${GCP_PROJECT_ID:?GCP_PROJECT_ID is required}"
: "${RAG_CORPUS_ID:?RAG_CORPUS_ID is required}"

REGION="${GCP_REGION:-us-central1}"
SERVICE_NAME="rulegate"
IMAGE="gcr.io/${GCP_PROJECT_ID}/${SERVICE_NAME}"

echo "==> Building container image..."
gcloud builds submit --tag "${IMAGE}" --project "${GCP_PROJECT_ID}"

echo "==> Deploying to Cloud Run..."
gcloud run deploy "${SERVICE_NAME}" \
  --image "${IMAGE}" \
  --region "${REGION}" \
  --project "${GCP_PROJECT_ID}" \
  --platform managed \
  --allow-unauthenticated \
  --set-env-vars "\
GCP_PROJECT_ID=${GCP_PROJECT_ID},\
GCP_REGION=${REGION},\
RAG_CORPUS_ID=${RAG_CORPUS_ID},\
GEMINI_MODEL=${GEMINI_MODEL:-gemini-2.5-flash},\
MIN_CONFIDENCE_DEFAULT=${MIN_CONFIDENCE_DEFAULT:-0.55},\
TOP_K_DEFAULT=${TOP_K_DEFAULT:-8},\
RATE_LIMIT_RPS=${RATE_LIMIT_RPS:-10},\
RATE_LIMIT_BURST=${RATE_LIMIT_BURST:-20},\
ALLOW_ORIGIN=${ALLOW_ORIGIN:-*},\
SOURCE_URL=${SOURCE_URL:-https://www.canoeicf.com/rules},\
PROMPTS_PATH=/docs/prompts.md" \
  --memory 512Mi \
  --cpu 1 \
  --timeout 60 \
  --max-instances 3

echo "==> Done. Service URL:"
gcloud run services describe "${SERVICE_NAME}" \
  --region "${REGION}" \
  --project "${GCP_PROJECT_ID}" \
  --format 'value(status.url)'
