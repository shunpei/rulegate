# rulegate — ICF Rules QA Bot（一般公開）設計書 / Claude Code 指示書

> **プロジェクト名**: rulegate（Rule + Gate）
> **Go module**: `github.com/shunpei/rulegate`

## 0. ゴール

英語版の **ICF Canoe Slalom Competition Rules (PDF)** を根拠に、ユーザーが **日本語で質問**すると、**日本語で回答**し、必ず **根拠（条文/セクション/出典URL）** を添えて返す一般公開Webアプリを作る。

* 方式：**RAG（Retrieval Augmented Generation）**
* インフラ：**Google Cloud**

  * API：Cloud Run
  * 検索：Vertex AI RAG Engine
  * 生成：Vertex AI Gemini
  * ストレージ：Cloud Storage
* 重要要件：**ハルシネーション抑止**（根拠が取れない場合は「見当たらない」と返す）

---

## 1. ユースケース

1. ユーザー（日本語）
   「ゲートに触った場合のペナルティは？」→ 日本語回答 + 条文番号/セクション + PDFへのリンク + （短い英語引用 25語以内）

2. ユーザー（日本語、条件付き）
   「再走（リラン）になる条件は？」→ 条件を列挙し、根拠の条文を複数提示

3. 根拠不足
   「Aという特殊ケースは？」→ 「提供されたルール本文の範囲では明記が見当たりません」と返す（推測禁止）

---

## 2. 非機能要件

* 一般公開のため **レート制限**（最低限）を実装
* コスト管理（上限設定・ログで監視）
* 監査ログ：質問、edition、top_k、スコア、応答時間、トークン量相当（取得可能な範囲）
* i18n：入力は日本語、根拠は英語テキスト、回答は日本語
* セキュリティ：プロンプトインジェクション対策（“コンテキスト以外の情報を使うな”強制）

---

## 3. 著作権/利用条件の前提

* ICFルールPDFは著作物。一般公開アプリとしては、

  * **全文表示は避ける**
  * **引用は短く（<= 25 words）**
  * **出典URL（ICF公式PDF）を必ず表示**
* アプリ内にPDF原本を再配布するか、公式URL参照にするかは実装で切替可能にする（MVPでは公式URL参照を基本にする）。

---

## 4. 全体アーキテクチャ（文章図）

* Frontend（Next.js 予定） → Cloud Run API `/ask`
* Cloud Run 内で

  1. クエリ展開（日本語→検索用英語）
  2. Vertex AI RAG Engine で retrieve
  3. Gemini で回答生成（コンテキスト厳密）
  4. JSONで返却
* Cloud Storage：PDF/テキスト（取り込み元）
* Cloud Logging：ログ集約

---

## 5. データ設計（RAGコーパス）

### 5.1 コーパス単位

* `icf_slalom_2025` のように **年度別・競技別** に corpus を分ける
* 受け付けるパラメータ：

  * `discipline` = `canoe_slalom`（将来拡張）
  * `rule_edition` = `2025`（将来拡張）

### 5.2 チャンク

* チャンク単位：章/条/段落
* サイズ目安：300〜800 tokens
* メタデータ：

  * `rule_id`（条文番号）
  * `section_title`
  * `edition`
  * `source_url`（ICF公式PDF URL）
  * `page_hint`（可能なら）

---

## 6. API仕様（Cloud Run）

### 6.1 `POST /ask`

#### Request JSON

```json
{
  "question_ja": "ゲートに触った場合のペナルティは？",
  "discipline": "canoe_slalom",
  "rule_edition": "2025",
  "context": {
    "boat_class": "K1",
    "race_phase": "final",
    "notes": ""
  },
  "options": {
    "top_k": 8,
    "min_confidence": 0.55,
    "return_contexts": false,
    "answer_style": "concise"
  }
}
```

#### Response JSON

```json
{
  "answer_ja": "…",
  "confidence": 0.0,
  "citations": [
    {
      "rule_id": "…",
      "section_title": "…",
      "quote_en": "…(<=25 words)…",
      "source_url": "…",
      "score": 0.0
    }
  ],
  "meta": {
    "rag_corpus": "icf_slalom_2025",
    "top_k": 8,
    "warnings": []
  }
}
```

#### 重要挙動

* retrieve結果の最大スコアが `min_confidence` 未満なら：

  * `answer_ja` は「ルール本文に該当箇所が見当たりません」
  * `citations` は空 or 低スコアの参考（要検討）
* 例外時：

  * 400：不正リクエスト
  * 429：レート制限
  * 500：Vertex障害等

---

## 7. プロンプト設計（2段）

### 7.1 クエリ展開用（日本語→検索用英語）

**目的**：単純翻訳ではなく、競技用語で検索に強くする
**出力**：JSONのみ

**System**

```
You are a query rewriting engine for ICF Canoe Slalom rules (English).
Convert a Japanese question into an English retrieval query optimized for rulebook search.
Return JSON only.
```

**User**

```
Japanese question:
{{question_ja}}

Optional context:
{{context_json}}

Return JSON:
{
  "q_en": "...",
  "keywords_en": ["..."],
  "q_ja": "..."
}
Constraints:
- Prefer official rulebook terms (e.g., missed gate, gate touch, DSQ, DNF, rerun).
- Include likely synonyms (DSQ=disqualification).
```

### 7.2 回答生成用（根拠コンテキスト厳密）

**目的**：ハルシネーション抑止・根拠必須
**出力**：JSONのみ

**System**

```
You answer questions about ICF Canoe Slalom rules.
RULES:
1) Use ONLY the provided contexts as source of truth.
2) Answer in Japanese.
3) Provide citations (rule_id, section_title, source_url) for every claim.
4) If contexts do not contain the answer, say you cannot find it.
5) Quotes must be short (<=25 words). Prefer Japanese paraphrase.
Output JSON only.
```

**User**

```
Question (Japanese):
{{question_ja}}

Retrieved contexts (English excerpts):
{{contexts_json}}

Return JSON:
{
  "answer_ja": "...",
  "citations": [
    {"rule_id":"...","section_title":"...","quote_en":"...","source_url":"...","score":0.0}
  ],
  "confidence": 0.0
}
```

---

## 8. 実装方針（Cloud Run / Go 推奨）

### 8.1 主要モジュール

* `internal/http`：ルーティング、バリデーション、レート制限
* `internal/rag`：RAG Engine 呼び出し
* `internal/llm`：Gemini 呼び出し（query rewrite / answer）
* `internal/domain`：Request/Response DTO、エラー型
* `internal/logging`：構造化ログ

### 8.2 レート制限（MVP）

* IP単位の簡易トークンバケット（メモリ or Redis/MemoryStoreは将来）
* まずは Cloud Run 1インスタンスでの簡易でもOK（ただしスケール時は要再設計）

### 8.3 環境変数（例）

* `GCP_PROJECT_ID`
* `GCP_REGION`
* `RAG_CORPUS_ID_Slalom_2025`（or マッピングJSON）
* `GEMINI_MODEL`（例：`gemini-1.5-pro` 等、実在モデル名は実装時に確認）
* `MIN_CONFIDENCE_DEFAULT`
* `TOP_K_DEFAULT`
* `RATE_LIMIT_RPS`
* `ALLOW_ORIGIN`（CORS）

---

## 9. リポジトリ構成（提案）

```
.
├── cmd/api/                 # Cloud Run entrypoint
├── internal/
│   ├── http/
│   ├── rag/
│   ├── llm/
│   ├── domain/
│   └── logging/
├── docs/
│   ├── designdoc.md
│   ├── api.md
│   ├── prompts.md
│   └── operations.md
├── scripts/
│   ├── ingest_rag.sh        # corpus取り込み（必要なら）
│   └── deploy.sh
├── infra/                   # terraform (optional)
└── README.md
```

---

## 10. 実装タスク（Claude Code にやってほしいこと）

### Phase 1: API最小実装

* [ ] `/healthz`（GET）
* [ ] `/ask`（POST）DTO + バリデーション
* [ ] クエリ展開（Gemini呼び出し）
* [ ] RAG retrieve 呼び出し（top_k、edition/discipline）
* [ ] スコア判定（min_confidence）
* [ ] 回答生成（Gemini呼び出し）
* [ ] citations整形（quote短縮、source_url付与）
* [ ] 構造化ログ出力（request_id、latency、score、token目安）

### Phase 2: 運用最低限

* [ ] レート制限（IP）
* [ ] CORS
* [ ] エラー整形（ユーザー向け/内部向け）
* [ ] Cloud Run デプロイ手順（README）

### Phase 3: 品質向上（余裕があれば）

* [ ] 攻撃的プロンプト/注入対策（“ignore previous” などのフィルタ）
* [ ] 取得コンテキストの重複排除/多様化
* [ ] 回答の「条件分岐テンプレ」（例外・条件の列挙）
* [ ] 回答の信頼度推定（スコア＋文脈数＋一貫性）

---

## 11. 受け入れ基準（Acceptance Criteria）

* 日本語質問 → 日本語回答が返る
* すべての回答に **少なくとも1つ** citation が付く（根拠ありの場合）
* 根拠が取れない場合は「不明/見当たらない」と返し、推測しない
* 引用は **25語以内**（英語）
* 平均応答：数秒以内（目標、環境次第）
* ログに `confidence`、最大スコア、top_k、latency が出る

---

## 12. 補足（実装時の注意）

* RAG Engine / Gemini のAPI仕様は更新されることがあるため、実装時点の公式ドキュメントに合わせること。
* 特に RAG の retrieve 出力（context/score/source）の扱いは “output explained” を参照して正規化すること。
* PDFの取り込みパイプライン（GCS→RAG corpus）は、運用スクリプト化する。

---

## 13. Claude Codeへの作業指示（短文）

* この設計書に従って Go で Cloud Run API を実装すること
* `/ask` の入出力は本設計のJSON schemaに合わせること
* “根拠がない回答をしない” 制約を最優先に実装すること
* プロンプトは `docs/prompts.md` に分離し、アプリ本体はテンプレ変数で読み込める設計にすること

