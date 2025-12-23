// Package config provides application configuration.
package config

import (
	"os"
	"path/filepath"
	"time"
)

// Config holds the application configuration.
type Config struct {
	CromwellHost    string
	CromwellTimeout time.Duration
	SessionDBPath   string

	// LLM Provider configuration
	LLMProvider string // "ollama" or "vertex"

	// Ollama-specific
	OllamaHost  string
	OllamaModel string

	// Vertex AI-specific
	VertexProject  string
	VertexLocation string
	VertexModel    string

	// Gemini API-specific (Google AI Studio)
	GeminiAPIKey string
	GeminiModel  string

	// WDL Context configuration
	WDLDirectory string // Directory containing WDL workflows for chat context
	WDLIndexPath string // Path to cached WDL index JSON file
}

// Load loads configuration from environment variables.
func Load() *Config {
	host := os.Getenv("CROMWELL_HOST")
	if host == "" {
		host = "http://localhost:8000"
	}

	sessionDBPath := os.Getenv("PUMBAA_SESSION_DB")
	if sessionDBPath == "" {
		home, _ := os.UserHomeDir()
		sessionDBPath = filepath.Join(home, ".pumbaa", "sessions.db")
	}

	// LLM Provider (default: ollama)
	llmProvider := os.Getenv("PUMBAA_LLM_PROVIDER")
	if llmProvider == "" {
		llmProvider = "ollama"
	}

	// Ollama config
	ollamaHost := os.Getenv("OLLAMA_HOST")
	if ollamaHost == "" {
		ollamaHost = "http://localhost:11434"
	}

	ollamaModel := os.Getenv("OLLAMA_MODEL")
	if ollamaModel == "" {
		ollamaModel = "llama3.2:3b"
	}

	// Vertex AI config
	vertexProject := os.Getenv("VERTEX_PROJECT")
	vertexLocation := os.Getenv("VERTEX_LOCATION")
	if vertexLocation == "" {
		vertexLocation = "us-central1"
	}
	vertexModel := os.Getenv("VERTEX_MODEL")
	if vertexModel == "" {
		vertexModel = "gemini-2.0-flash"
	}

	// WDL Context config
	wdlDirectory := os.Getenv("PUMBAA_WDL_DIR")
	wdlIndexPath := os.Getenv("PUMBAA_WDL_INDEX")
	if wdlIndexPath == "" {
		home, _ := os.UserHomeDir()
		wdlIndexPath = filepath.Join(home, ".pumbaa", "wdl_index.json")
	}

	// Gemini API config
	geminiAPIKey := os.Getenv("GEMINI_API_KEY")
	geminiModel := os.Getenv("GEMINI_MODEL")
	if geminiModel == "" {
		geminiModel = "gemini-2.0-flash"
	}

	return &Config{
		CromwellHost:    host,
		CromwellTimeout: 30 * time.Second,
		SessionDBPath:   sessionDBPath,
		LLMProvider:     llmProvider,
		OllamaHost:      ollamaHost,
		OllamaModel:     ollamaModel,
		VertexProject:   vertexProject,
		VertexLocation:  vertexLocation,
		VertexModel:     vertexModel,
		GeminiAPIKey:    geminiAPIKey,
		GeminiModel:     geminiModel,
		WDLDirectory:    wdlDirectory,
		WDLIndexPath:    wdlIndexPath,
	}
}

// FromFlags creates a config from CLI flags, with env vars as fallback.
func FromFlags(host string) *Config {
	cfg := Load()

	if host != "" {
		cfg.CromwellHost = host
	}

	return cfg
}
