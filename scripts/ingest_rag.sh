#!/usr/bin/env bash
set -euo pipefail

# Ingest a PDF into a Vertex AI RAG Engine corpus.
# Usage: ./scripts/ingest_rag.sh
#
# Required environment variables:
#   GCP_PROJECT_ID   - GCP project ID
#   GCP_REGION       - GCP region (default: us-central1)
#   GCS_PDF_URI      - GCS URI of the PDF (e.g., gs://bucket/icf_slalom_2025.pdf)
#
# Optional:
#   CORPUS_DISPLAY_NAME - Display name for the corpus (default: icf_slalom_2025)
#   CHUNK_SIZE          - Chunk size in tokens (default: 1000)
#   CHUNK_OVERLAP       - Chunk overlap in tokens (default: 100)

: "${GCP_PROJECT_ID:?GCP_PROJECT_ID is required}"
: "${GCS_PDF_URI:?GCS_PDF_URI is required}"

REGION="${GCP_REGION:-us-central1}"
DISPLAY_NAME="${CORPUS_DISPLAY_NAME:-icf_slalom_2025}"
CHUNK_SIZE="${CHUNK_SIZE:-1000}"
CHUNK_OVERLAP="${CHUNK_OVERLAP:-100}"
API_ENDPOINT="${REGION}-aiplatform.googleapis.com"

echo "==> Creating RAG corpus: ${DISPLAY_NAME}..."
CORPUS_RESPONSE=$(curl -s -X POST \
  "https://${API_ENDPOINT}/v1/projects/${GCP_PROJECT_ID}/locations/${REGION}/ragCorpora" \
  -H "Authorization: Bearer $(gcloud auth print-access-token)" \
  -H "Content-Type: application/json" \
  -d "{
    \"display_name\": \"${DISPLAY_NAME}\",
    \"rag_embedding_model_config\": {
      \"vertex_prediction_endpoint\": {
        \"endpoint\": \"projects/${GCP_PROJECT_ID}/locations/${REGION}/publishers/google/models/text-embedding-005\"
      }
    },
    \"rag_vector_db_config\": {
      \"rag_managed_db\": {}
    }
  }")

echo "Corpus creation response:"
echo "${CORPUS_RESPONSE}" | python3 -m json.tool 2>/dev/null || echo "${CORPUS_RESPONSE}"

# Extract operation name for polling.
OP_NAME=$(echo "${CORPUS_RESPONSE}" | python3 -c "import sys,json; print(json.load(sys.stdin).get('name',''))" 2>/dev/null || true)

if [ -n "${OP_NAME}" ]; then
  echo "==> Waiting for corpus creation (${OP_NAME})..."
  while true; do
    STATUS=$(curl -s \
      "https://${API_ENDPOINT}/v1/${OP_NAME}" \
      -H "Authorization: Bearer $(gcloud auth print-access-token)" \
      | python3 -c "import sys,json; print(json.load(sys.stdin).get('done', False))" 2>/dev/null || echo "false")
    if [ "${STATUS}" = "True" ]; then
      break
    fi
    echo "  Still creating..."
    sleep 5
  done
fi

# List corpora to find the one we just created.
echo "==> Listing corpora..."
CORPORA=$(curl -s \
  "https://${API_ENDPOINT}/v1/projects/${GCP_PROJECT_ID}/locations/${REGION}/ragCorpora" \
  -H "Authorization: Bearer $(gcloud auth print-access-token)")

CORPUS_NAME=$(echo "${CORPORA}" | python3 -c "
import sys, json
data = json.load(sys.stdin)
for c in data.get('ragCorpora', []):
    if c.get('displayName') == '${DISPLAY_NAME}':
        print(c['name'])
        break
" 2>/dev/null || true)

if [ -z "${CORPUS_NAME}" ]; then
  echo "ERROR: Could not find corpus with display name ${DISPLAY_NAME}"
  exit 1
fi

echo "==> Corpus: ${CORPUS_NAME}"
echo "==> Importing PDF: ${GCS_PDF_URI}..."

IMPORT_RESPONSE=$(curl -s -X POST \
  "https://${API_ENDPOINT}/v1/${CORPUS_NAME}/ragFiles:import" \
  -H "Authorization: Bearer $(gcloud auth print-access-token)" \
  -H "Content-Type: application/json" \
  -d "{
    \"import_rag_files_config\": {
      \"gcs_source\": {
        \"uris\": [\"${GCS_PDF_URI}\"]
      },
      \"rag_file_chunking_config\": {
        \"chunk_size\": ${CHUNK_SIZE},
        \"chunk_overlap\": ${CHUNK_OVERLAP}
      }
    }
  }")

echo "Import response:"
echo "${IMPORT_RESPONSE}" | python3 -m json.tool 2>/dev/null || echo "${IMPORT_RESPONSE}"

echo ""
echo "==> Done. Set this in your .env:"
echo "RAG_CORPUS_ID=${CORPUS_NAME}"
