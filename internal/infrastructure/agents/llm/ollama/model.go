package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"net/http"
	"strings"
	"time"

	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

// Model implements model.LLM for Ollama
type Model struct {
	baseURL string
	model   string
	client  *http.Client
	options *Options
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

	var opts *Options
	if cfg.Temperature > 0 || cfg.TopP > 0 {
		opts = &Options{
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
// - Simple text generation (with optional streaming)
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
		ollamaReq.Stream = stream

		if stream {
			m.streamRequest(ctx, ollamaReq, yield)
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

// streamRequest executes a streaming call to Ollama (NDJSON lines), yielding
// each text delta as a Partial response and finishing with one aggregated
// final response carrying the full message, tool calls and token counts.
func (m *Model) streamRequest(ctx context.Context, req *ChatRequest, yield func(*model.LLMResponse, error) bool) {
	data, err := json.Marshal(req)
	if err != nil {
		_ = yield(nil, fmt.Errorf("error serializing request: %w", err))
		return
	}

	url := m.baseURL + "/api/chat"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		_ = yield(nil, fmt.Errorf("error creating HTTP request: %w", err))
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(httpReq)
	if err != nil {
		_ = yield(nil, fmt.Errorf("error calling Ollama: %w", err))
		return
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		_ = yield(nil, fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(body)))
		return
	}

	var (
		text      strings.Builder
		toolCalls []ToolCall
		final     ChatResponse
	)

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}

		var chunk ChatResponse
		if err := json.Unmarshal(line, &chunk); err != nil {
			_ = yield(nil, fmt.Errorf("error decoding stream chunk: %w", err))
			return
		}

		if chunk.Message.Content != "" {
			text.WriteString(chunk.Message.Content)
			partial := &model.LLMResponse{
				Content: &genai.Content{
					Role:  "model",
					Parts: []*genai.Part{genai.NewPartFromText(chunk.Message.Content)},
				},
				Partial: true,
			}
			if !yield(partial, nil) {
				return
			}
		}
		toolCalls = append(toolCalls, chunk.Message.ToolCalls...)

		if chunk.Done {
			final = chunk
		}
	}
	if err := scanner.Err(); err != nil {
		_ = yield(nil, fmt.Errorf("error reading stream: %w", err))
		return
	}

	final.Message.Role = "assistant"
	final.Message.Content = text.String()
	final.Message.ToolCalls = toolCalls

	adkResp, err := m.buildResponse(&final)
	if err != nil {
		_ = yield(nil, fmt.Errorf("error building response: %w", err))
		return
	}
	_ = yield(adkResp, nil)
}

// doRequest executes HTTP call to Ollama
func (m *Model) doRequest(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
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
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(body))
	}

	var ollamaResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return &ollamaResp, nil
}

// Close closes the client connection.
func (m *Model) Close() error {
	m.client.CloseIdleConnections()
	return nil
}
