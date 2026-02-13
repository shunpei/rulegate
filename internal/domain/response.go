package domain

// AskResponse is the JSON response for POST /ask.
type AskResponse struct {
	AnswerJA   string     `json:"answer_ja"`
	Confidence float64    `json:"confidence"`
	Citations  []Citation `json:"citations"`
	Meta       Meta       `json:"meta"`
}

type Citation struct {
	RuleID       string  `json:"rule_id"`
	SectionTitle string  `json:"section_title"`
	QuoteEN      string  `json:"quote_en"`
	SourceURL    string  `json:"source_url"`
	Score        float64 `json:"score"`
}

type Meta struct {
	RAGCorpus string   `json:"rag_corpus"`
	TopK      int      `json:"top_k"`
	Warnings  []string `json:"warnings"`
}

// NotFoundResponse returns the standard "not found in rules" response.
func NotFoundResponse(corpus string, topK int) *AskResponse {
	return &AskResponse{
		AnswerJA:   "ルール本文に該当箇所が見当たりません",
		Confidence: 0.0,
		Citations:  []Citation{},
		Meta: Meta{
			RAGCorpus: corpus,
			TopK:      topK,
			Warnings:  []string{},
		},
	}
}

// ErrorResponse is used for non-200 error responses.
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// RetrievedContext represents a single context from RAG retrieval.
type RetrievedContext struct {
	Text         string  `json:"text"`
	Score        float64 `json:"score"`
	SourceURI    string  `json:"source_uri,omitempty"`
	RuleID       string  `json:"rule_id,omitempty"`
	SectionTitle string  `json:"section_title,omitempty"`
}

// RewriteResult is the output of query rewriting.
type RewriteResult struct {
	QueryEN    string   `json:"q_en"`
	KeywordsEN []string `json:"keywords_en"`
	QueryJA    string   `json:"q_ja"`
}

// AnswerResult is the output of answer generation.
type AnswerResult struct {
	AnswerJA   string     `json:"answer_ja"`
	Citations  []Citation `json:"citations"`
	Confidence float64    `json:"confidence"`
}
