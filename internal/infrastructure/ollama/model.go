// Package ollama provides an implementation of model.LLM for Ollama,
// allowing local models to be used with the Google Agents ADK.
package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"net/http"
	"strings"
	"time"

	"github.com/lmtani/pumbaa/internal/infrastructure/agent/tools"
	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

// ============================================================================
// Structures for Ollama API with Tools support
// ============================================================================

// OllamaMessage represents a message in the Ollama API format
type OllamaMessage struct {
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	ToolName  string     `json:"tool_name,omitempty"` // Ollama API uses tool_name, not tool_call_id
}

// ToolCall represents a tool call made by the model
type ToolCall struct {
	ID       string       `json:"id,omitempty"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// FunctionCall contains details of the function to be called
type FunctionCall struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// OllamaTool represents a tool in the format expected by Ollama
type OllamaTool struct {
	Type     string         `json:"type"`
	Function OllamaFunction `json:"function"`
}

// OllamaFunction defines the structure of a function/tool
type OllamaFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// OllamaChatRequest is the request body for /api/chat
type OllamaChatRequest struct {
	Model    string          `json:"model"`
	Messages []OllamaMessage `json:"messages"`
	Tools    []OllamaTool    `json:"tools,omitempty"`
	Stream   bool            `json:"stream"`
	Options  *OllamaOptions  `json:"options,omitempty"`
}

// OllamaOptions allows configuring model parameters
type OllamaOptions struct {
	Temperature float64 `json:"temperature,omitempty"`
	TopP        float64 `json:"top_p,omitempty"`
	TopK        int     `json:"top_k,omitempty"`
	NumPredict  int     `json:"num_predict,omitempty"`
}

// OllamaChatResponse is the response from Ollama
type OllamaChatResponse struct {
	Model     string        `json:"model"`
	CreatedAt string        `json:"created_at"`
	Message   OllamaMessage `json:"message"`
	Done      bool          `json:"done"`
}

// ============================================================================
// Implementation of model.LLM for Ollama
// ============================================================================

// Model implements model.LLM for Ollama
type Model struct {
	baseURL string
	model   string
	client  *http.Client
	options *OllamaOptions
}

// Config contains configurations for creating a new Model
type Config struct {
	BaseURL     string
	ModelName   string
	Timeout     time.Duration
	Temperature float64
	TopP        float64
}

// NewModel creates a new instance of the Ollama model
// It is equivalent to gemini.NewModel, but for Ollama.
func NewModel(baseURL, modelName string) model.LLM {
	return NewModelWithConfig(Config{
		BaseURL:   baseURL,
		ModelName: modelName,
	})
}

// NewModelWithConfig creates a model with advanced configurations
func NewModelWithConfig(cfg Config) model.LLM {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "http://localhost:11434"
	}
	if cfg.ModelName == "" {
		cfg.ModelName = "llama3.2:3b"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 120 * time.Second
	}

	var opts *OllamaOptions
	if cfg.Temperature > 0 || cfg.TopP > 0 {
		opts = &OllamaOptions{
			Temperature: cfg.Temperature,
			TopP:        cfg.TopP,
		}
	}

	return &Model{
		baseURL: strings.TrimRight(cfg.BaseURL, "/"),
		model:   cfg.ModelName,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
		options: opts,
	}
}

// Name returns the identifier name of the model
func (m *Model) Name() string {
	return fmt.Sprintf("ollama/%s", m.model)
}

// GenerateContent implements model.LLM.GenerateContent
// Supports:
// - Simple text generation
// - Tool calls (function calling)
// - System instructions
func (m *Model) GenerateContent(
	ctx context.Context,
	req *model.LLMRequest,
	stream bool,
) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		// Convert ADK request to Ollama format
		ollamaReq, err := m.buildRequest(req)
		if err != nil {
			_ = yield(nil, fmt.Errorf("error building request: %w", err))
			return
		}

		// Make HTTP call
		ollamaResp, err := m.doRequest(ctx, ollamaReq)
		if err != nil {
			_ = yield(nil, err)
			return
		}

		// Convert response to ADK format
		adkResp, err := m.buildResponse(ollamaResp)
		if err != nil {
			_ = yield(nil, fmt.Errorf("error building response: %w", err))
			return
		}

		_ = yield(adkResp, nil)
	}
}

// buildRequest converts an ADK LLMRequest to OllamaChatRequest
func (m *Model) buildRequest(req *model.LLMRequest) (*OllamaChatRequest, error) {
	ollamaReq := &OllamaChatRequest{
		Model:   m.model,
		Stream:  false,
		Options: m.options,
	}

	// Add system instruction if exists
	if req.Config != nil && req.Config.SystemInstruction != nil {
		systemText := extractTextFromContent(req.Config.SystemInstruction)
		if systemText != "" {
			ollamaReq.Messages = append(ollamaReq.Messages, OllamaMessage{
				Role:    "system",
				Content: systemText,
			})
		}
	}

	// Convert message history
	for _, content := range req.Contents {
		if content == nil {
			continue
		}
		// Check if this content has FunctionResponse parts - these need to be separate messages
		if content.Role == "tool" {
			for _, part := range content.Parts {
				if part != nil && part.FunctionResponse != nil {
					toolMsg := m.functionResponseToMessage(part.FunctionResponse)
					ollamaReq.Messages = append(ollamaReq.Messages, toolMsg)
				}
			}
			continue
		}
		msg := m.contentToMessage(content)
		if msg.Content != "" || len(msg.ToolCalls) > 0 {
			ollamaReq.Messages = append(ollamaReq.Messages, msg)
		}
	}

	// Convert tools
	if req.Config != nil && len(req.Config.Tools) > 0 {
		ollamaReq.Tools = m.convertTools(req.Config.Tools)
	}

	return ollamaReq, nil
}

// contentToMessage converts genai.Content to OllamaMessage
func (m *Model) contentToMessage(content *genai.Content) OllamaMessage {
	msg := OllamaMessage{
		Role: mapRole(content.Role),
	}

	var textParts []string

	for _, part := range content.Parts {
		if part == nil {
			continue
		}

		// Check each content type in Part
		if part.Text != "" {
			textParts = append(textParts, part.Text)
		}

		if part.FunctionCall != nil {
			args, _ := json.Marshal(part.FunctionCall.Args)
			msg.ToolCalls = append(msg.ToolCalls, ToolCall{
				ID:   part.FunctionCall.Name,
				Type: "function",
				Function: FunctionCall{
					Name:      part.FunctionCall.Name,
					Arguments: args,
				},
			})
		}

		// Note: FunctionResponse parts are now handled separately in buildRequest
		// This block is kept for backwards compatibility but should not be reached
		// when the Content.Role is "tool"
		if part.FunctionResponse != nil {
			return m.functionResponseToMessage(part.FunctionResponse)
		}
	}

	if len(textParts) > 0 {
		msg.Content = strings.Join(textParts, "\n")
	}

	return msg
}

// functionResponseToMessage converts a FunctionResponse to an OllamaMessage
// Each FunctionResponse becomes a separate tool message per Ollama API format
func (m *Model) functionResponseToMessage(fr *genai.FunctionResponse) OllamaMessage {
	msg := OllamaMessage{
		Role:     "tool",
		ToolName: fr.Name, // Ollama uses tool_name, not tool_call_id
	}

	if fr.Response != nil {
		// Convert errors to string before serializing
		// ADK returns errors as map[string]any{"error": error}
		// but error is an interface that serializes as {}
		response := convertErrorsToStrings(fr.Response)

		// If there is an error, format explicitly for the model to understand
		if errMsg, hasError := response["error"]; hasError {
			msg.Content = fmt.Sprintf("ERROR: The tool failed with the following error: %v. Inform the user about this problem.", errMsg)
		} else {
			respJSON, _ := json.Marshal(response)
			msg.Content = string(respJSON)
		}
	}

	return msg
}

// convertTools converts ADK tools to Ollama format
func (m *Model) convertTools(tools []*genai.Tool) []OllamaTool {
	var ollamaTools []OllamaTool

	for _, tool := range tools {
		if tool == nil {
			continue
		}
		for _, fn := range tool.FunctionDeclarations {
			if fn == nil {
				continue
			}

			params := make(map[string]interface{})
			if fn.Parameters != nil && len(fn.Parameters.Properties) > 0 {
				// Use the schema from ADK
				params["type"] = fn.Parameters.Type
				props := make(map[string]interface{})
				for name, prop := range fn.Parameters.Properties {
					propMap := map[string]interface{}{
						"type":        prop.Type,
						"description": prop.Description,
					}
					if len(prop.Enum) > 0 {
						propMap["enum"] = prop.Enum
					}
					props[name] = propMap
				}
				params["properties"] = props
				if len(fn.Parameters.Required) > 0 {
					params["required"] = fn.Parameters.Required
				}
			} else if fn.Name == "pumbaa" {
				// Fallback: ADK functiontool doesn't populate Parameters for Ollama
				// Provide explicit schema for the pumbaa tool
				params = getPumbaaParametersSchema()
			}

			ollamaTools = append(ollamaTools, OllamaTool{
				Type: "function",
				Function: OllamaFunction{
					Name:        fn.Name,
					Description: fn.Description,
					Parameters:  params,
				},
			})
		}
	}

	return ollamaTools
}

// getPumbaaParametersSchema returns the schema from the tools package.
// This ensures a single source of truth for the pumbaa tool parameters.
func getPumbaaParametersSchema() map[string]interface{} {
	return tools.GetParametersSchema()
}

// doRequest executes HTTP call to Ollama
func (m *Model) doRequest(ctx context.Context, req *OllamaChatRequest) (*OllamaChatResponse, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error serializing request: %w", err)
	}

	url := m.baseURL + "/api/chat"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("error creating HTTP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("error calling Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Ollama returned status %d: %s", resp.StatusCode, string(body))
	}

	var ollamaResp OllamaChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return &ollamaResp, nil
}

// buildResponse converts OllamaChatResponse to model.LLMResponse
func (m *Model) buildResponse(resp *OllamaChatResponse) (*model.LLMResponse, error) {
	var parts []*genai.Part

	// Check for tool calls in response
	if len(resp.Message.ToolCalls) > 0 {
		for _, tc := range resp.Message.ToolCalls {
			var args map[string]any
			if err := json.Unmarshal(tc.Function.Arguments, &args); err != nil {
				// If fails, try as string
				args = map[string]any{"raw": string(tc.Function.Arguments)}
			}
			parts = append(parts, genai.NewPartFromFunctionCall(tc.Function.Name, args))
		}
	}

	// Add response text if exists
	if text := strings.TrimSpace(resp.Message.Content); text != "" {
		parts = append(parts, genai.NewPartFromText(text))
		// fmt.Printf("[DEBUG] Response text: %s\n", text[:min(200, len(text))])
	}

	content := &genai.Content{
		Role:  "model",
		Parts: parts,
	}

	turnComplete := len(resp.Message.ToolCalls) == 0
	// fmt.Printf("[DEBUG] TurnComplete=%v, parts=%d\n", turnComplete, len(parts))

	return &model.LLMResponse{
		Content:      content,
		TurnComplete: turnComplete,
	}, nil
}

// ============================================================================
// Helper functions
// ============================================================================

// mapRole converts ADK roles to Ollama roles
func mapRole(role string) string {
	switch role {
	case "model":
		return "assistant"
	case "user":
		return "user"
	case "system":
		return "system"
	case "tool":
		return "tool"
	default:
		return "user"
	}
}

// extractTextFromContent extracts text from a genai.Content
func extractTextFromContent(content *genai.Content) string {
	if content == nil {
		return ""
	}
	var parts []string
	for _, p := range content.Parts {
		if p != nil && p.Text != "" {
			parts = append(parts, p.Text)
		}
	}
	return strings.Join(parts, "\n")
}

// convertErrorsToStrings converts error values to string
// ADK returns tools errors as map[string]any{"error": error}
// but error interface does not serialize well in JSON (results in {})
func convertErrorsToStrings(m map[string]any) map[string]any {
	result := make(map[string]any, len(m))
	for k, v := range m {
		if err, ok := v.(error); ok {
			result[k] = err.Error()
		} else {
			result[k] = v
		}
	}
	return result
}
