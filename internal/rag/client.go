package rag

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	aiplatform "cloud.google.com/go/aiplatform/apiv1"
	aiplatformpb "cloud.google.com/go/aiplatform/apiv1/aiplatformpb"
	"github.com/shunpei/rulegate/internal/domain"
	"google.golang.org/api/option"
)

// Retriever abstracts RAG context retrieval for testability.
type Retriever interface {
	RetrieveContexts(ctx context.Context, query string, corpusID string, topK int) ([]domain.RetrievedContext, error)
	Close() error
}

// VertexRAGClient implements Retriever using Vertex AI RAG Engine.
type VertexRAGClient struct {
	client    *aiplatform.VertexRagClient
	projectID string
	region    string
}

// NewVertexRAGClient creates a new Vertex RAG Engine client.
func NewVertexRAGClient(ctx context.Context, projectID, region string) (*VertexRAGClient, error) {
	endpoint := fmt.Sprintf("%s-aiplatform.googleapis.com:443", region)
	client, err := aiplatform.NewVertexRagClient(ctx, option.WithEndpoint(endpoint))
	if err != nil {
		return nil, fmt.Errorf("create vertex rag client: %w", err)
	}
	return &VertexRAGClient{
		client:    client,
		projectID: projectID,
		region:    region,
	}, nil
}

func (c *VertexRAGClient) RetrieveContexts(ctx context.Context, query string, corpusID string, topK int) ([]domain.RetrievedContext, error) {
	parent := fmt.Sprintf("projects/%s/locations/%s", c.projectID, c.region)

	req := &aiplatformpb.RetrieveContextsRequest{
		Parent: parent,
		DataSource: &aiplatformpb.RetrieveContextsRequest_VertexRagStore_{
			VertexRagStore: &aiplatformpb.RetrieveContextsRequest_VertexRagStore{
				RagResources: []*aiplatformpb.RetrieveContextsRequest_VertexRagStore_RagResource{
					{RagCorpus: corpusID},
				},
			},
		},
		Query: &aiplatformpb.RagQuery{
			Query: &aiplatformpb.RagQuery_Text{Text: query},
			RagRetrievalConfig: &aiplatformpb.RagRetrievalConfig{
				TopK: int32(topK),
			},
		},
	}

	resp, err := c.client.RetrieveContexts(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("retrieve contexts: %w", err)
	}

	contexts := resp.GetContexts().GetContexts()
	results := make([]domain.RetrievedContext, 0, len(contexts))

	seen := make(map[string]bool)
	for _, c := range contexts {
		text := c.GetText()

		// De-duplicate by text content.
		if seen[text] {
			slog.DebugContext(ctx, "skipping duplicate context")
			continue
		}
		seen[text] = true

		var score float64
		if c.Score != nil {
			score = *c.Score
		}

		results = append(results, domain.RetrievedContext{
			Text:      text,
			Score:     score,
			SourceURI: c.GetSourceUri(),
		})
	}

	return results, nil
}

func (c *VertexRAGClient) Close() error {
	return c.client.Close()
}

// CorpusName builds the corpus resource name from discipline and rule edition.
// If corpusID is already a full resource name, it returns it as-is.
func CorpusName(corpusID, discipline, ruleEdition string) string {
	// If it's already a full resource name, return as-is.
	if strings.HasPrefix(corpusID, "projects") {
		return corpusID
	}
	return fmt.Sprintf("icf_%s_%s", discipline, ruleEdition)
}
