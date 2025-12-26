package ollama

import (
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/adk/model"
	"google.golang.org/genai"

	"github.com/lmtani/pumbaa/internal/infrastructure/chat/agent/tools"
)

// buildRequest converts an ADK LLMRequest to ChatRequest
func (m *Model) buildRequest(req *model.LLMRequest) (*ChatRequest, error) {
	ollamaReq := &ChatRequest{
		Model:   m.model,
		Stream:  false,
		Options: m.options,
	}

	// Add system instruction if exists
	if req.Config != nil && req.Config.SystemInstruction != nil {
		systemText := extractTextFromContent(req.Config.SystemInstruction)
		if systemText != "" {
			ollamaReq.Messages = append(ollamaReq.Messages, Message{
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

// contentToMessage converts genai.Content to Message
func (m *Model) contentToMessage(content *genai.Content) Message {
	msg := Message{
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

// functionResponseToMessage converts a FunctionResponse to an Message
// Each FunctionResponse becomes a separate tool message per Ollama API format
func (m *Model) functionResponseToMessage(fr *genai.FunctionResponse) Message {
	msg := Message{
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
func (m *Model) convertTools(tools []*genai.Tool) []Tool {
	var ollamaTools []Tool

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

			ollamaTools = append(ollamaTools, Tool{
				Type: "function",
				Function: Function{
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

// buildResponse converts ChatResponse to model.LLMResponse
func (m *Model) buildResponse(resp *ChatResponse) (*model.LLMResponse, error) {
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
	}

	content := &genai.Content{
		Role:  "model",
		Parts: parts,
	}

	turnComplete := len(resp.Message.ToolCalls) == 0

	return &model.LLMResponse{
		Content:      content,
		TurnComplete: turnComplete,
	}, nil
}

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
