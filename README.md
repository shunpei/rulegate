# rulegate

Japanese Q&A API for ICF Canoe Slalom Competition Rules, powered by RAG on Google Cloud.

## Architecture

- **Runtime**: Cloud Run (Go)
- **Retrieval**: Vertex AI RAG Engine
- **Generation**: Vertex AI Gemini (2.5 Flash)
- **Storage**: Cloud Storage (PDF source)

## Quick Start

### Prerequisites

- Go 1.24+
- Google Cloud project with Vertex AI API enabled
- `gcloud` CLI authenticated

### Local Development

```bash
cp .env.example .env
# Edit .env with your GCP project ID and RAG corpus ID

# Load environment
export $(cat .env | xargs)

# Run
go run ./cmd/api
```

### Test

```bash
go test ./...
```

### API Usage

```bash
# Health check
curl http://localhost:8080/healthz

# Ask a question
curl -X POST http://localhost:8080/ask \
  -H 'Content-Type: application/json' \
  -d '{"question_ja": "ゲートに触った場合のペナルティは？"}'
```

## RAG Corpus Setup

1. Upload the ICF Canoe Slalom rules PDF to GCS:
   ```bash
   gsutil cp icf_canoe_slalom_2025.pdf gs://YOUR_BUCKET/
   ```

2. Create corpus and ingest PDF:
   ```bash
   export GCP_PROJECT_ID=your-project
   export GCS_PDF_URI=gs://YOUR_BUCKET/icf_canoe_slalom_2025.pdf
   ./scripts/ingest_rag.sh
   ```

3. Set the output `RAG_CORPUS_ID` in your `.env`.

## Deploy to Cloud Run

```bash
export GCP_PROJECT_ID=your-project
export RAG_CORPUS_ID=projects/.../ragCorpora/...
./scripts/deploy.sh
```

## API Reference

### `POST /ask`

**Request:**
```json
{
  "question_ja": "ゲートに触った場合のペナルティは？",
  "discipline": "canoe_slalom",
  "rule_edition": "2025",
  "options": {
    "top_k": 8,
    "min_confidence": 0.55
  }
}
```

**Response:**
```json
{
  "answer_ja": "ゲートに触った場合、2秒のペナルティが課されます。",
  "confidence": 0.85,
  "citations": [
    {
      "rule_id": "29.4",
      "section_title": "Penalties",
      "quote_en": "A 2-second penalty for each gate touch.",
      "source_url": "https://www.canoeicf.com/rules",
      "score": 0.88
    }
  ],
  "meta": {
    "rag_corpus": "icf_slalom_2025",
    "top_k": 8,
    "warnings": []
  }
}
```

### `GET /healthz`

Returns `{"status": "ok"}`.
