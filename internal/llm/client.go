package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/shunpei/rulegate/internal/domain"
	"google.golang.org/genai"
)

// LLM abstracts generative AI operations for testability.
type LLM interface {
	RewriteQuery(ctx context.Context, questionJA string, queryCtx *domain.QueryContext) (*domain.RewriteResult, error)
	GenerateAnswer(ctx context.Context, questionJA string, contexts []domain.RetrievedContext, sourceURL string) (*domain.AnswerResult, error)
	Close() error
}

// GeminiClient implements LLM using the google.golang.org/genai SDK.
type GeminiClient struct {
	client       *genai.Client
	model        string
	rewriteModel string
	prompts      *PromptTemplates
}

// NewGeminiClient creates a new Gemini client via Vertex AI backend.
func NewGeminiClient(ctx context.Context, projectID, region, model, rewriteModel string, prompts *PromptTemplates) (*GeminiClient, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Project:  projectID,
		Location: region,
		Backend:  genai.BackendVertexAI,
	})
	if err != nil {
		return nil, fmt.Errorf("create genai client: %w", err)
	}
	return &GeminiClient{
		client:       client,
		model:        model,
		rewriteModel: rewriteModel,
		prompts:      prompts,
	}, nil
}

func (c *GeminiClient) RewriteQuery(ctx context.Context, questionJA string, queryCtx *domain.QueryContext) (*domain.RewriteResult, error) {
	contextJSON := "{}"
	if queryCtx != nil {
		b, _ := json.Marshal(queryCtx)
		contextJSON = string(b)
	}

	userPrompt := RenderTemplate(c.prompts.RewriteUser, map[string]string{
		"question_ja":  questionJA,
		"context_json": contextJSON,
	})

	resp, err := c.client.Models.GenerateContent(ctx,
		c.rewriteModel,
		[]*genai.Content{
			{Parts: []*genai.Part{{Text: userPrompt}}, Role: "user"},
		},
		&genai.GenerateContentConfig{
			SystemInstruction: &genai.Content{
				Parts: []*genai.Part{{Text: c.prompts.RewriteSystem}},
			},
			ResponseMIMEType: "application/json",
			Temperature:      genai.Ptr[float32](0.2),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("rewrite query: %w", err)
	}

	text := resp.Text()
	var result domain.RewriteResult
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, fmt.Errorf("parse rewrite response: %w (raw: %s)", err, truncate(text, 200))
	}

	return &result, nil
}

func (c *GeminiClient) GenerateAnswer(ctx context.Context, questionJA string, contexts []domain.RetrievedContext, sourceURL string) (*domain.AnswerResult, error) {
	contextsJSON, _ := json.Marshal(contexts)

	userPrompt := RenderTemplate(c.prompts.AnswerUser, map[string]string{
		"question_ja":   questionJA,
		"contexts_json": string(contextsJSON),
	})

	resp, err := c.client.Models.GenerateContent(ctx,
		c.model,
		[]*genai.Content{
			{Parts: []*genai.Part{{Text: userPrompt}}, Role: "user"},
		},
		&genai.GenerateContentConfig{
			SystemInstruction: &genai.Content{
				Parts: []*genai.Part{{Text: c.prompts.AnswerSystem}},
			},
			ResponseMIMEType: "application/json",
			Temperature:      genai.Ptr[float32](0.3),
			MaxOutputTokens:  16384,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("generate answer: %w", err)
	}

	text := resp.Text()
	var result domain.AnswerResult
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, fmt.Errorf("parse answer response: %w (raw: %s)", err, truncate(text, 200))
	}

	// Enforce citation constraints.
	for i := range result.Citations {
		result.Citations[i].QuoteEN = enforceWordLimit(result.Citations[i].QuoteEN, 25)
		if result.Citations[i].SourceURL == "" {
			result.Citations[i].SourceURL = sourceURL
		}
	}

	return &result, nil
}

func (c *GeminiClient) Close() error {
	// The genai client doesn't have a Close method that returns error.
	return nil
}

// enforceWordLimit truncates text to maxWords and appends "..." if truncated.
func enforceWordLimit(text string, maxWords int) string {
	words := strings.Fields(text)
	if len(words) <= maxWords {
		return text
	}
	return strings.Join(words[:maxWords], " ") + "..."
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
