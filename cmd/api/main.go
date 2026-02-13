package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	apphttp "github.com/shunpei/rulegate/internal/http"
	"github.com/shunpei/rulegate/internal/llm"
	"github.com/shunpei/rulegate/internal/logging"
	"github.com/shunpei/rulegate/internal/rag"
)

func main() {
	logging.Init()

	if err := run(); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Load configuration from environment.
	projectID := envOrDefault("GCP_PROJECT_ID", "")
	region := envOrDefault("GCP_REGION", "us-central1")
	ragCorpusID := envOrDefault("RAG_CORPUS_ID", "")
	model := envOrDefault("GEMINI_MODEL", "gemini-2.5-flash")
	rewriteModel := envOrDefault("GEMINI_REWRITE_MODEL", model)
	sourceURL := envOrDefault("SOURCE_URL", "https://www.canoeicf.com/rules")
	allowOrigin := envOrDefault("ALLOW_ORIGIN", "*")
	port := envOrDefault("PORT", "8080")
	promptsPath := envOrDefault("PROMPTS_PATH", "docs/prompts.md")

	defaultTopK := envOrDefaultInt("TOP_K_DEFAULT", 8)
	defaultMinConf := envOrDefaultFloat("MIN_CONFIDENCE_DEFAULT", 0.55)
	rateLimitRPS := envOrDefaultFloat("RATE_LIMIT_RPS", 10.0)
	rateLimitBurst := envOrDefaultInt("RATE_LIMIT_BURST", 20)

	if projectID == "" {
		return fmt.Errorf("GCP_PROJECT_ID is required")
	}
	if ragCorpusID == "" {
		return fmt.Errorf("RAG_CORPUS_ID is required")
	}

	// Load prompt templates.
	prompts, err := llm.LoadPrompts(promptsPath)
	if err != nil {
		return fmt.Errorf("load prompts: %w", err)
	}
	slog.Info("prompts loaded", "path", promptsPath)

	// Initialize RAG client.
	ragClient, err := rag.NewVertexRAGClient(ctx, projectID, region)
	if err != nil {
		return fmt.Errorf("init rag client: %w", err)
	}
	defer ragClient.Close()

	// Initialize LLM client.
	llmClient, err := llm.NewGeminiClient(ctx, projectID, region, model, rewriteModel, prompts)
	if err != nil {
		return fmt.Errorf("init llm client: %w", err)
	}
	defer llmClient.Close()

	// Build handler and server.
	handler := apphttp.NewHandler(ragClient, llmClient, apphttp.Config{
		DefaultTopK:    defaultTopK,
		DefaultMinConf: defaultMinConf,
		SourceURL:      sourceURL,
		RAGCorpusID:    ragCorpusID,
	})

	rateLimiter := apphttp.NewIPRateLimiter(rateLimitRPS, rateLimitBurst)
	mux := apphttp.NewServeMux(handler, rateLimiter, allowOrigin)

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown.
	errCh := make(chan error, 1)
	go func() {
		slog.Info("server starting", "port", port)
		errCh <- srv.ListenAndServe()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		slog.Info("shutdown signal received", "signal", sig.String())
	case err := <-errCh:
		if err != http.ErrServerClosed {
			return fmt.Errorf("server error: %w", err)
		}
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}

	slog.Info("server stopped gracefully")
	return nil
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envOrDefaultInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func envOrDefaultFloat(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}
