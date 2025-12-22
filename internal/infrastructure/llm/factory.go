// Package llm provides a factory for creating LLM instances based on configuration.
package llm

import (
	"fmt"

	"github.com/lmtani/pumbaa/internal/config"
	"github.com/lmtani/pumbaa/internal/infrastructure/ollama"
	"google.golang.org/adk/model"
)

// Provider constants
const (
	ProviderOllama = "ollama"
	ProviderVertex = "vertex"
)

// NewLLM creates a new LLM instance based on the configuration.
func NewLLM(cfg *config.Config) (model.LLM, error) {
	switch cfg.LLMProvider {
	case ProviderOllama:
		return ollama.NewModel(cfg.OllamaHost, cfg.OllamaModel), nil
	case ProviderVertex:
		return NewVertexModel(cfg.VertexProject, cfg.VertexLocation, cfg.VertexModel)
	default:
		return nil, fmt.Errorf("unknown LLM provider: %s (supported: ollama, vertex)", cfg.LLMProvider)
	}
}
