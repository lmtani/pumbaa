// Package llm provides LLM implementations.
package llm

import (
	"context"
	"fmt"
	"iter"

	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

// VertexModel implements model.LLM for Google Vertex AI using the genai SDK.
type VertexModel struct {
	client    *genai.Client
	modelName string
}

// NewVertexModel creates a new Vertex AI model.
// Requires GOOGLE_APPLICATION_CREDENTIALS or running on GCP with appropriate permissions.
func NewVertexModel(project, location, modelName string) (*VertexModel, error) {
	if project == "" {
		return nil, fmt.Errorf("vertex project is required (set VERTEX_PROJECT or --vertex-project)")
	}
	if location == "" {
		location = "us-central1"
	}
	if modelName == "" {
		modelName = "gemini-2.0-flash"
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Project:  project,
		Location: location,
		Backend:  genai.BackendVertexAI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Vertex AI client: %w", err)
	}

	return &VertexModel{
		client:    client,
		modelName: modelName,
	}, nil
}

// Name returns the model identifier.
func (m *VertexModel) Name() string {
	return fmt.Sprintf("vertex/%s", m.modelName)
}

// GenerateContent implements model.LLM.GenerateContent.
func (m *VertexModel) GenerateContent(
	ctx context.Context,
	req *model.LLMRequest,
	stream bool,
) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		// Build the genai request
		genaiReq := &genai.GenerateContentConfig{}

		if req.Config != nil {
			genaiReq = req.Config
		}

		// Use the genai client to generate content
		resp, err := m.client.Models.GenerateContent(ctx, m.modelName, req.Contents, genaiReq)
		if err != nil {
			_ = yield(nil, fmt.Errorf("vertex AI generation failed: %w", err))
			return
		}

		// Convert response
		if len(resp.Candidates) == 0 {
			_ = yield(nil, fmt.Errorf("no candidates in response"))
			return
		}

		candidate := resp.Candidates[0]
		adkResp := &model.LLMResponse{
			Content:       candidate.Content,
			UsageMetadata: resp.UsageMetadata,
		}

		_ = yield(adkResp, nil)
	}
}

// Close closes the client connection.
func (m *VertexModel) Close() error {
	// genai.Client doesn't have a Close method in the current SDK
	return nil
}
