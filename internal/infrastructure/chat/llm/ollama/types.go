// Package ollama provides an implementation of model.LLM for Ollama,
// allowing local models to be used with the Google Agents ADK.
//
// NOTE: This package exists because Ollama does not have an official Go SDK
// with full support for the features we need (like Tool Calling/Function Calling).
// Therefore, we implement the communication with the Ollama API manually here.
package ollama

import (
	"encoding/json"
)

// Message represents a message in the Ollama API format
type Message struct {
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

// Tool represents a tool in the format expected by Ollama
type Tool struct {
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

// Function defines the structure of a function/tool
type Function struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ChatRequest is the request body for /api/chat
type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Tools    []Tool    `json:"tools,omitempty"`
	Stream   bool      `json:"stream"`
	Options  *Options  `json:"options,omitempty"`
}

// Options allows configuring model parameters
type Options struct {
	Temperature float64 `json:"temperature,omitempty"`
	TopP        float64 `json:"top_p,omitempty"`
	TopK        int     `json:"top_k,omitempty"`
	NumPredict  int     `json:"num_predict,omitempty"`
}

// ChatResponse is the response from Ollama
type ChatResponse struct {
	Model     string  `json:"model"`
	CreatedAt string  `json:"created_at"`
	Message   Message `json:"message"`
	Done      bool    `json:"done"`
}
