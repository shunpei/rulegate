package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/shunpei/rulegate/internal/domain"
)

// --- Mocks ---

type mockRetriever struct {
	contexts []domain.RetrievedContext
	err      error
}

func (m *mockRetriever) RetrieveContexts(_ context.Context, _ string, _ string, _ int) ([]domain.RetrievedContext, error) {
	return m.contexts, m.err
}
func (m *mockRetriever) Close() error { return nil }

type mockLLM struct {
	rewriteResult *domain.RewriteResult
	answerResult  *domain.AnswerResult
	rewriteErr    error
	answerErr     error
}

func (m *mockLLM) RewriteQuery(_ context.Context, _ string, _ *domain.QueryContext) (*domain.RewriteResult, error) {
	return m.rewriteResult, m.rewriteErr
}
func (m *mockLLM) GenerateAnswer(_ context.Context, _ string, _ []domain.RetrievedContext, _ string) (*domain.AnswerResult, error) {
	return m.answerResult, m.answerErr
}
func (m *mockLLM) Close() error { return nil }

func defaultMockLLM() *mockLLM {
	return &mockLLM{
		rewriteResult: &domain.RewriteResult{
			QueryEN:    "penalty for gate touch",
			KeywordsEN: []string{"gate touch", "penalty"},
			QueryJA:    "ゲートに触った場合のペナルティは？",
		},
		answerResult: &domain.AnswerResult{
			AnswerJA:   "ゲートに触った場合、2秒のペナルティが課されます。",
			Confidence: 0.85,
			Citations: []domain.Citation{
				{
					RuleID:       "29.4",
					SectionTitle: "Penalties",
					QuoteEN:      "A 2-second penalty is applied for each gate touch.",
					SourceURL:    "https://www.canoeicf.com/rules",
					Score:        0.88,
				},
			},
		},
	}
}

func defaultConfig() Config {
	return Config{
		DefaultTopK:    8,
		DefaultMinConf: 0.55,
		SourceURL:      "https://www.canoeicf.com/rules",
		RAGCorpusID:    "projects/test/locations/us-central1/ragCorpora/test",
	}
}

// --- Tests ---

func TestAsk_MissingQuestionJA_Returns400(t *testing.T) {
	h := NewHandler(&mockRetriever{}, defaultMockLLM(), defaultConfig())

	body := `{"discipline":"canoe_slalom"}`
	req := httptest.NewRequest(http.MethodPost, "/ask", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Ask(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}

	var resp domain.ErrorResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Code != string(domain.ErrCatValidation) {
		t.Errorf("expected validation error code, got %q", resp.Code)
	}
}

func TestAsk_InvalidJSON_Returns400(t *testing.T) {
	h := NewHandler(&mockRetriever{}, defaultMockLLM(), defaultConfig())

	req := httptest.NewRequest(http.MethodPost, "/ask", strings.NewReader("not json"))
	rec := httptest.NewRecorder()

	h.Ask(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestAsk_LowScore_ReturnsNotFound(t *testing.T) {
	retriever := &mockRetriever{
		contexts: []domain.RetrievedContext{
			{Text: "some text", Score: 0.3},
			{Text: "other text", Score: 0.2},
		},
	}
	h := NewHandler(retriever, defaultMockLLM(), defaultConfig())

	body := `{"question_ja":"存在しない質問"}`
	req := httptest.NewRequest(http.MethodPost, "/ask", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.Ask(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var resp domain.AskResponse
	json.NewDecoder(rec.Body).Decode(&resp)

	if resp.AnswerJA != "ルール本文に該当箇所が見当たりません" {
		t.Errorf("expected not-found message, got %q", resp.AnswerJA)
	}
	if len(resp.Citations) != 0 {
		t.Errorf("expected no citations, got %d", len(resp.Citations))
	}
	if resp.Confidence != 0.0 {
		t.Errorf("expected confidence 0, got %f", resp.Confidence)
	}
}

func TestAsk_HighScore_ReturnsAnswer(t *testing.T) {
	retriever := &mockRetriever{
		contexts: []domain.RetrievedContext{
			{Text: "A 2-second penalty is applied for each gate touch.", Score: 0.88},
		},
	}
	h := NewHandler(retriever, defaultMockLLM(), defaultConfig())

	body := `{"question_ja":"ゲートに触った場合のペナルティは？"}`
	req := httptest.NewRequest(http.MethodPost, "/ask", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.Ask(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	// Read raw bytes first, then decode from them.
	raw := rec.Body.Bytes()

	var resp domain.AskResponse
	json.Unmarshal(raw, &resp)

	if resp.AnswerJA == "" {
		t.Error("expected non-empty answer")
	}
	if resp.Confidence == 0 {
		t.Error("expected non-zero confidence")
	}
	if len(resp.Citations) == 0 {
		t.Error("expected at least one citation")
	}

	// Verify JSON schema fields exist.
	var rawMap map[string]any
	json.Unmarshal(raw, &rawMap)

	for _, field := range []string{"answer_ja", "confidence", "citations", "meta"} {
		if _, ok := rawMap[field]; !ok {
			t.Errorf("missing required field %q in response", field)
		}
	}

	// Verify meta fields.
	meta, ok := rawMap["meta"].(map[string]any)
	if !ok {
		t.Fatal("meta is not an object")
	}
	for _, field := range []string{"rag_corpus", "top_k", "warnings"} {
		if _, ok := meta[field]; !ok {
			t.Errorf("missing required meta field %q", field)
		}
	}
}

func TestAsk_CitationQuoteEnforced25Words(t *testing.T) {
	longQuote := "this is a very long quote that has way more than twenty five words in it and should be truncated by the enforcement function to meet the citation policy requirements set forth in the design document"
	llmClient := &mockLLM{
		rewriteResult: &domain.RewriteResult{
			QueryEN: "test query",
		},
		answerResult: &domain.AnswerResult{
			AnswerJA:   "テスト回答",
			Confidence: 0.9,
			Citations: []domain.Citation{
				{
					RuleID:       "1.1",
					SectionTitle: "Test",
					QuoteEN:      longQuote,
					SourceURL:    "https://example.com",
					Score:        0.9,
				},
			},
		},
	}

	retriever := &mockRetriever{
		contexts: []domain.RetrievedContext{
			{Text: "test context", Score: 0.9},
		},
	}

	h := NewHandler(retriever, llmClient, defaultConfig())

	body := `{"question_ja":"テスト質問"}`
	req := httptest.NewRequest(http.MethodPost, "/ask", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.Ask(rec, req)

	var resp domain.AskResponse
	json.NewDecoder(rec.Body).Decode(&resp)

	for _, c := range resp.Citations {
		words := strings.Fields(c.QuoteEN)
		if len(words) > 26 { // 25 + "..." counted as part of last element
			t.Errorf("quote_en exceeds 25 words: %d words", len(words))
		}
	}
}

func TestAsk_DefaultsApplied(t *testing.T) {
	retriever := &mockRetriever{
		contexts: []domain.RetrievedContext{
			{Text: "test", Score: 0.9},
		},
	}
	h := NewHandler(retriever, defaultMockLLM(), defaultConfig())

	// No discipline or rule_edition specified — should use defaults.
	body := `{"question_ja":"テスト"}`
	req := httptest.NewRequest(http.MethodPost, "/ask", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.Ask(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestHealthz(t *testing.T) {
	h := NewHandler(&mockRetriever{}, defaultMockLLM(), defaultConfig())

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	h.Healthz(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var resp map[string]string
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Errorf("expected status ok, got %q", resp["status"])
	}
}

func TestRateLimiter(t *testing.T) {
	limiter := NewIPRateLimiter(1, 1) // 1 RPS, burst 1

	h := NewHandler(&mockRetriever{
		contexts: []domain.RetrievedContext{{Text: "test", Score: 0.9}},
	}, defaultMockLLM(), defaultConfig())

	mux := NewServeMux(h, limiter, "*")

	// First request should succeed.
	body := `{"question_ja":"テスト"}`
	req := httptest.NewRequest(http.MethodPost, "/ask", bytes.NewReader([]byte(body)))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("first request: expected 200, got %d", rec.Code)
	}

	// Second immediate request should be rate limited.
	req = httptest.NewRequest(http.MethodPost, "/ask", bytes.NewReader([]byte(body)))
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("second request: expected 429, got %d", rec.Code)
	}
}
