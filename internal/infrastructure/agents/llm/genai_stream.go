package llm

import (
	"context"
	"fmt"
	"iter"
	"strings"

	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

// generateGenaiContent adapts a genai client call to the ADK iterator
// contract, shared by the Gemini and Vertex models.
//
// When stream is false it yields the single complete response. When stream
// is true it yields each text chunk as a Partial response and finishes with
// one aggregated non-partial response carrying the full content, collected
// function calls and usage metadata — matching the ADK convention that only
// the final non-partial event is fully processed downstream.
func generateGenaiContent(
	ctx context.Context,
	client *genai.Client,
	modelName string,
	req *model.LLMRequest,
	stream bool,
	wrapErr func(error) error,
) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		config := &genai.GenerateContentConfig{}
		if req.Config != nil {
			config = req.Config
		}

		if !stream {
			resp, err := client.Models.GenerateContent(ctx, modelName, req.Contents, config)
			if err != nil {
				_ = yield(nil, wrapErr(err))
				return
			}
			if len(resp.Candidates) == 0 {
				_ = yield(nil, wrapErr(fmt.Errorf("no candidates in response")))
				return
			}
			_ = yield(&model.LLMResponse{
				Content:       resp.Candidates[0].Content,
				UsageMetadata: resp.UsageMetadata,
			}, nil)
			return
		}

		var (
			text    strings.Builder
			fnParts []*genai.Part
			usage   *genai.GenerateContentResponseUsageMetadata
		)

		for resp, err := range client.Models.GenerateContentStream(ctx, modelName, req.Contents, config) {
			if err != nil {
				_ = yield(nil, wrapErr(err))
				return
			}
			if resp.UsageMetadata != nil {
				usage = resp.UsageMetadata
			}
			if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
				continue
			}

			content := resp.Candidates[0].Content
			hasText := false
			for _, part := range content.Parts {
				if part == nil {
					continue
				}
				if part.Text != "" {
					text.WriteString(part.Text)
					hasText = true
				}
				if part.FunctionCall != nil {
					fnParts = append(fnParts, part)
				}
			}

			if hasText {
				if !yield(&model.LLMResponse{Content: content, Partial: true}, nil) {
					return
				}
			}
		}

		parts := make([]*genai.Part, 0, len(fnParts)+1)
		if text.Len() > 0 {
			parts = append(parts, genai.NewPartFromText(text.String()))
		}
		parts = append(parts, fnParts...)
		if len(parts) == 0 {
			_ = yield(nil, wrapErr(fmt.Errorf("no candidates in response")))
			return
		}

		_ = yield(&model.LLMResponse{
			Content:       &genai.Content{Role: "model", Parts: parts},
			UsageMetadata: usage,
			TurnComplete:  len(fnParts) == 0,
		}, nil)
	}
}
