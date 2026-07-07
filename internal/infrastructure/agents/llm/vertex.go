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
		modelName = "gemini-2.5-flash"
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

// GenerateContent implements model.LLM.GenerateContent. Streaming yields
// Partial responses per chunk followed by one aggregated final response.
func (m *VertexModel) GenerateContent(
	ctx context.Context,
	req *model.LLMRequest,
	stream bool,
) iter.Seq2[*model.LLMResponse, error] {
	return generateGenaiContent(ctx, m.client, m.modelName, req, stream, func(err error) error {
		return fmt.Errorf("vertex AI generation failed: %w", err)
	})
}

// Close closes the client connection.
func (m *VertexModel) Close() error {
	// genai.Client doesn't have a Close method in the current SDK
	return nil
}
