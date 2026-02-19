package http

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/shunpei/rulegate/internal/domain"
	"github.com/shunpei/rulegate/internal/llm"
	"github.com/shunpei/rulegate/internal/logging"
	"github.com/shunpei/rulegate/internal/rag"
)

// Config holds handler configuration from environment.
type Config struct {
	DefaultTopK    int
	DefaultMinConf float64
	SourceURL      string
	RAGCorpusID    string
}

// Handler implements the /api/ask and /healthz endpoints.
type Handler struct {
	retriever rag.Retriever
	llm       llm.LLM
	cfg       Config
}

func NewHandler(retriever rag.Retriever, llmClient llm.LLM, cfg Config) *Handler {
	return &Handler{
		retriever: retriever,
		llm:       llmClient,
		cfg:       cfg,
	}
}

func (h *Handler) Healthz(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) Ask(c echo.Context) error {
	ctx := c.Request().Context()
	reqID := logging.RequestID(ctx)
	totalStart := time.Now()

	// Parse request body.
	var req domain.AskRequest
	if err := c.Bind(&req); err != nil {
		return respondAppError(c, domain.NewValidationError("invalid JSON body"))
	}

	// Validate.
	if err := req.Validate(); err != nil {
		return respondAppError(c, err)
	}

	topK := req.EffectiveTopK(h.cfg.DefaultTopK)
	minConf := req.EffectiveMinConfidence(h.cfg.DefaultMinConf)
	corpusID := h.cfg.RAGCorpusID

	logFields := []any{
		"request_id", reqID,
		"discipline", req.Discipline,
		"rule_edition", req.RuleEdition,
		"top_k", topK,
		"min_confidence", minConf,
	}

	// Step 1: Query rewrite (JA → EN).
	rewriteStart := time.Now()
	rewritten, err := h.llm.RewriteQuery(ctx, req.QuestionJA, req.Context)
	rewriteLatency := time.Since(rewriteStart)
	if err != nil {
		slog.ErrorContext(ctx, "rewrite failed", append(logFields, "error", err)...)
		return respondAppError(c, domain.NewVertexError("query rewrite failed", err))
	}

	slog.InfoContext(ctx, "query rewritten",
		append(logFields, "q_en", rewritten.QueryEN, "rewrite_ms", rewriteLatency.Milliseconds())...,
	)

	// Step 2: RAG retrieval.
	retrieveStart := time.Now()
	contexts, err := h.retriever.RetrieveContexts(ctx, rewritten.QueryEN, corpusID, topK)
	retrieveLatency := time.Since(retrieveStart)
	if err != nil {
		slog.ErrorContext(ctx, "retrieval failed", append(logFields, "error", err)...)
		return respondAppError(c, domain.NewVertexError("context retrieval failed", err))
	}

	// Step 3: Score gating.
	maxScore := 0.0
	for _, ctx := range contexts {
		if ctx.Score > maxScore {
			maxScore = ctx.Score
		}
	}

	slog.InfoContext(ctx, "retrieval done",
		append(logFields,
			"max_score", maxScore,
			"num_contexts", len(contexts),
			"retrieve_ms", retrieveLatency.Milliseconds(),
		)...,
	)

	corpus := rag.CorpusName(corpusID, req.Discipline, req.RuleEdition)

	if maxScore < minConf {
		slog.InfoContext(ctx, "below confidence threshold",
			append(logFields, "max_score", maxScore, "threshold", minConf)...,
		)
		resp := domain.NotFoundResponse(corpus, topK)
		return c.JSON(http.StatusOK, resp)
	}

	// Step 4: Answer generation.
	genStart := time.Now()
	answer, err := h.llm.GenerateAnswer(ctx, req.QuestionJA, contexts, h.cfg.SourceURL)
	genLatency := time.Since(genStart)
	if err != nil {
		slog.ErrorContext(ctx, "generation failed", append(logFields, "error", err)...)
		return respondAppError(c, domain.NewVertexError("answer generation failed", err))
	}

	totalLatency := time.Since(totalStart)

	slog.InfoContext(ctx, "answer generated",
		append(logFields,
			"confidence", answer.Confidence,
			"num_citations", len(answer.Citations),
			"rewrite_ms", rewriteLatency.Milliseconds(),
			"retrieve_ms", retrieveLatency.Milliseconds(),
			"generate_ms", genLatency.Milliseconds(),
			"total_ms", totalLatency.Milliseconds(),
		)...,
	)

	// Enforce citation constraints at handler level (defense in depth).
	citations := answer.Citations
	for i := range citations {
		citations[i].QuoteEN = enforceWordLimit(citations[i].QuoteEN, 25)
		if citations[i].SourceURL == "" {
			citations[i].SourceURL = h.cfg.SourceURL
		}
	}
	if citations == nil {
		citations = []domain.Citation{}
	}

	resp := &domain.AskResponse{
		AnswerJA:   answer.AnswerJA,
		Confidence: answer.Confidence,
		Citations:  citations,
		Meta: domain.Meta{
			RAGCorpus: corpus,
			TopK:      topK,
			Warnings:  []string{},
		},
	}

	return c.JSON(http.StatusOK, resp)
}

// enforceWordLimit truncates text to maxWords and appends "..." if truncated.
func enforceWordLimit(text string, maxWords int) string {
	words := strings.Fields(text)
	if len(words) <= maxWords {
		return text
	}
	return strings.Join(words[:maxWords], " ") + "..."
}

func respondAppError(c echo.Context, err error) error {
	var appErr *domain.AppError
	if errors.As(err, &appErr) {
		return c.JSON(appErr.StatusCode, domain.ErrorResponse{
			Error: appErr.Message,
			Code:  string(appErr.Category),
		})
	}
	return c.JSON(http.StatusInternalServerError, domain.ErrorResponse{
		Error: "internal server error",
		Code:  string(domain.ErrCatUnknown),
	})
}
