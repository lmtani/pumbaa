// Package llm provides LLM implementations.
package llm

import (
	"context"
	"fmt"
	"iter"
	"os"

	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

// GeminiModel implements model.LLM for Google Gemini API using an API key.
// This is for users who don't have Vertex AI access.
type GeminiModel struct {
	client    *genai.Client
	modelName string
}

// NewGeminiModel creates a new Gemini API model using an API key.
// Requires GEMINI_API_KEY environment variable or explicit apiKey parameter.
func NewGeminiModel(apiKey, modelName string) (*GeminiModel, error) {
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("Gemini API key is required (set GEMINI_API_KEY or --gemini-api-key)")
	}
	if modelName == "" {
		modelName = "gemini-2.0-flash"
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini API client: %w", err)
	}

	return &GeminiModel{
		client:    client,
		modelName: modelName,
	}, nil
}

// Name returns the model identifier.
func (m *GeminiModel) Name() string {
	return fmt.Sprintf("gemini/%s", m.modelName)
}

// GenerateContent implements model.LLM.GenerateContent.
func (m *GeminiModel) GenerateContent(
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
			_ = yield(nil, fmt.Errorf("Gemini API generation failed: %w", err))
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
func (m *GeminiModel) Close() error {
	// genai.Client doesn't have a Close method in the current SDK
	return nil
}
