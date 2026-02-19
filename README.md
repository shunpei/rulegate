# rulegate

ICF カヌースラローム競技規則（英語PDF）に対する **日本語 Q&A Webサービス** です。
RAG（Retrieval Augmented Generation）を用いて、ルールブックの根拠に基づいた回答を返します。

## 特徴

- 日本語で質問すると、ルールブックの該当箇所を検索し、日本語で回答
- 回答には必ず根拠（条文番号・セクション・出典URL・英語引用）を付与
- 根拠が見つからない場合は「見当たりません」と返し、推測しない（ハルシネーション抑止）
- レート制限・CORS・プロンプトインジェクション対策を実装

## 技術スタック

| 要素 | 技術 |
|---|---|
| バックエンド | Go 1.24+ / Echo v4 |
| フロントエンド | Next.js 15 / TypeScript / Tailwind CSS v4 / shadcn/ui |
| ランタイム | Cloud Run（API + Web の2サービス） |
| 検索（Retrieval） | Vertex AI RAG Engine |
| 生成（Generation） | Vertex AI Gemini (gemini-2.5-flash) |
| PDF保管 | Cloud Storage |
| ログ | Cloud Logging（slog 構造化ログ） |
| コンテナ | Docker（API: distroless / Web: node:22-alpine） |
| CI/CD | GitHub Actions + Workload Identity Federation |

## プロジェクト構成

```
.
├── cmd/api/              # API エントリーポイント
├── internal/
│   ├── domain/           # DTO、エラー型
│   ├── http/             # Echo ハンドラー・ミドルウェア
│   ├── llm/              # Gemini クライアント
│   ├── logging/          # 構造化ログ
│   └── rag/              # Vertex AI RAG Engine クライアント
├── frontend/             # Next.js 15 フロントエンド
│   ├── src/
│   │   ├── app/          # App Router ページ
│   │   ├── components/   # UI コンポーネント
│   │   └── lib/          # API クライアント・型定義
│   ├── Dockerfile        # 本番用
│   └── Dockerfile.dev    # 開発用
├── docs/
│   ├── designdoc.md      # 設計書
│   └── prompts.md        # プロンプトテンプレート
├── scripts/
│   ├── deploy-api.sh     # API Cloud Run デプロイ
│   ├── deploy-web.sh     # Web Cloud Run デプロイ
│   └── ingest_rag.sh     # RAG コーパス作成 + PDF取り込み
├── .github/workflows/    # CI/CD
├── compose.yaml          # Docker Compose ローカル開発
├── Dockerfile            # API 本番用
├── Dockerfile.dev        # API 開発用（air ホットリロード）
├── Makefile
└── README.md
```

## 処理フロー

1. **クエリ展開** — 日本語の質問を Gemini で検索用の英語クエリに変換
2. **検索（Retrieve）** — Vertex AI RAG Engine でルールブックから関連コンテキストを取得
3. **スコア判定** — 最大スコアが `min_confidence` 未満なら「見当たりません」を返す
4. **回答生成** — 取得したコンテキストのみを使い、Gemini で日本語回答を生成
5. **レスポンス** — 回答 + 根拠引用（citations）を JSON で返却

## ローカル開発（Docker Compose）

### 前提条件

- Docker / Docker Compose
- Google Cloud 認証済み（`gcloud auth application-default login`）

### 起動

```bash
cp .env.example .env
# .env を編集して GCP_PROJECT_ID, RAG_CORPUS_ID を設定

make up
```

- フロントエンド: http://localhost:3000
- API: http://localhost:8080
- ヘルスチェック: http://localhost:8080/healthz

### その他コマンド

```bash
make down     # 停止
make logs     # ログ表示
make test     # Go テスト実行
```

## ローカル開発（Docker なし）

### 前提条件

- Go 1.24 以上
- Node.js 22 以上
- Google Cloud プロジェクト（Vertex AI API 有効化済み）
- `gcloud` CLI（認証済み）

### API 起動

```bash
export $(cat .env | xargs)
go run ./cmd/api
```

### フロントエンド起動

```bash
cd frontend
npm install
npm run dev
```

### テスト

```bash
go test ./...
```

## 環境変数

| 変数名 | 説明 | デフォルト |
|---|---|---|
| `GCP_PROJECT_ID` | GCP プロジェクト ID | — |
| `GCP_REGION` | リージョン | `us-central1` |
| `RAG_CORPUS_ID` | RAG コーパスのリソース名 | — |
| `GEMINI_MODEL` | 回答生成モデル | `gemini-2.5-flash` |
| `GEMINI_REWRITE_MODEL` | クエリ展開モデル | `gemini-2.5-flash` |
| `MIN_CONFIDENCE_DEFAULT` | 最低信頼度スコア | `0.55` |
| `TOP_K_DEFAULT` | 検索時の取得件数 | `8` |
| `RATE_LIMIT_RPS` | レート制限（リクエスト/秒） | `10` |
| `RATE_LIMIT_BURST` | レート制限バースト | `20` |
| `ALLOW_ORIGIN` | CORS 許可オリジン | `*` |
| `PORT` | API リッスンポート | `8080` |
| `SOURCE_URL` | ルールPDFの出典URL | `https://www.canoeicf.com/rules` |
| `API_URL` | フロントエンド → API のURL（Web サービス用） | `http://localhost:8080` |

## RAG コーパスのセットアップ

1. ICF カヌースラロームルール PDF を Cloud Storage にアップロード:
   ```bash
   gsutil cp icf_canoe_slalom_2025.pdf gs://YOUR_BUCKET/
   ```

2. コーパスを作成し PDF を取り込み:
   ```bash
   export GCP_PROJECT_ID=your-project
   export GCS_PDF_URI=gs://YOUR_BUCKET/icf_canoe_slalom_2025.pdf
   ./scripts/ingest_rag.sh
   ```

3. 出力された `RAG_CORPUS_ID` を `.env` に設定

## API リファレンス

### `POST /api/ask`

日本語で質問し、ルールブックに基づいた回答を取得します。

**リクエスト:**

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

| フィールド | 必須 | 説明 |
|---|---|---|
| `question_ja` | はい | 日本語の質問文 |
| `discipline` | いいえ | 競技種別（デフォルト: `canoe_slalom`） |
| `rule_edition` | いいえ | ルール版（デフォルト: `2025`） |
| `options.top_k` | いいえ | 検索取得件数 |
| `options.min_confidence` | いいえ | 最低信頼度スコア |

**レスポンス（根拠あり）:**

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

**レスポンス（根拠なし）:**

```json
{
  "answer_ja": "ルール本文に該当箇所が見当たりません",
  "confidence": 0.0,
  "citations": [],
  "meta": {
    "rag_corpus": "icf_slalom_2025",
    "top_k": 8,
    "warnings": []
  }
}
```

**エラーコード:**

| ステータス | 説明 |
|---|---|
| `400` | 不正なリクエスト（`question_ja` が未指定など） |
| `429` | レート制限超過 |
| `502` | Vertex AI 障害 |

### `GET /healthz`

ヘルスチェックエンドポイント。

```json
{"status": "ok"}
```

## Cloud Run へのデプロイ

### API デプロイ

```bash
export GCP_PROJECT_ID=your-project
export RAG_CORPUS_ID=projects/.../ragCorpora/...
./scripts/deploy-api.sh
```

### Web デプロイ

```bash
export GCP_PROJECT_ID=your-project
export API_URL=https://rulegate-api-xxxxx.run.app
./scripts/deploy-web.sh
```

## ライセンス

本ソフトウェアのライセンスは未定です。
ICF 競技規則の著作権は ICF に帰属します。本APIは規則の全文を提供するものではなく、短い引用（25語以内）と出典URLを通じて参照を提供します。
