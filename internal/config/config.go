// Package config provides application configuration.
package config

import (
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
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

	// Telemetry
	TelemetryEnabled bool   `yaml:"telemetry_enabled" env:"PUMBAA_TELEMETRY_ENABLED" default:"true"`
	ClientID         string `yaml:"client_id" env:"PUMBAA_CLIENT_ID"`

	// WDL Context configuration
	WDLDirectory string // Directory containing WDL workflows for chat context
	WDLIndexPath string // Path to cached WDL index JSON file
}

// Load loads configuration from file and environment variables.
// Priority: CLI flags > env vars > config file > defaults
func Load() *Config {
	// Load file config first
	fileCfg, _ := LoadFileConfig()

	// Cromwell host: env > file > default
	host := os.Getenv("CROMWELL_HOST")
	if host == "" && fileCfg.CromwellHost != "" {
		host = fileCfg.CromwellHost
	}
	if host == "" {
		host = "http://localhost:8000"
	}

	sessionDBPath := os.Getenv("PUMBAA_SESSION_DB")
	if sessionDBPath == "" {
		home, _ := os.UserHomeDir()
		sessionDBPath = filepath.Join(home, ".pumbaa", "sessions.db")
	}

	// LLM Provider: env > file > default
	llmProvider := os.Getenv("PUMBAA_LLM_PROVIDER")
	if llmProvider == "" && fileCfg.LLMProvider != "" {
		llmProvider = fileCfg.LLMProvider
	}
	if llmProvider == "" {
		llmProvider = "ollama"
	}

	// Ollama config: env > file > default
	ollamaHost := os.Getenv("OLLAMA_HOST")
	if ollamaHost == "" && fileCfg.OllamaHost != "" {
		ollamaHost = fileCfg.OllamaHost
	}
	if ollamaHost == "" {
		ollamaHost = "http://localhost:11434"
	}

	ollamaModel := os.Getenv("OLLAMA_MODEL")
	if ollamaModel == "" && fileCfg.OllamaModel != "" {
		ollamaModel = fileCfg.OllamaModel
	}
	if ollamaModel == "" {
		ollamaModel = "llama3.2:3b"
	}

	// Vertex AI config: env > file > default
	vertexProject := os.Getenv("VERTEX_PROJECT")
	if vertexProject == "" && fileCfg.VertexProject != "" {
		vertexProject = fileCfg.VertexProject
	}

	vertexLocation := os.Getenv("VERTEX_LOCATION")
	if vertexLocation == "" && fileCfg.VertexLocation != "" {
		vertexLocation = fileCfg.VertexLocation
	}
	if vertexLocation == "" {
		vertexLocation = "us-central1"
	}

	vertexModel := os.Getenv("VERTEX_MODEL")
	if vertexModel == "" && fileCfg.VertexModel != "" {
		vertexModel = fileCfg.VertexModel
	}
	if vertexModel == "" {
		vertexModel = "gemini-2.0-flash"
	}

	// WDL Context config: env > file
	wdlDirectory := os.Getenv("PUMBAA_WDL_DIR")
	if wdlDirectory == "" && fileCfg.WDLDirectory != "" {
		wdlDirectory = fileCfg.WDLDirectory
	}

	wdlIndexPath := os.Getenv("PUMBAA_WDL_INDEX")
	if wdlIndexPath == "" {
		home, _ := os.UserHomeDir()
		wdlIndexPath = filepath.Join(home, ".pumbaa", "wdl_index.json")
	}

	// Gemini API config: env > file > default
	geminiAPIKey := os.Getenv("GEMINI_API_KEY")
	if geminiAPIKey == "" && fileCfg.GeminiAPIKey != "" {
		geminiAPIKey = fileCfg.GeminiAPIKey
	}

	geminiModel := os.Getenv("GEMINI_MODEL")
	if geminiModel == "" && fileCfg.GeminiModel != "" {
		geminiModel = fileCfg.GeminiModel
	}
	if geminiModel == "" {
		geminiModel = "gemini-2.0-flash"
	}

	// Telemetry config
	// Telemetry config
	telemetryEnabled := true // default true by default
	if os.Getenv("PUMBAA_TELEMETRY_ENABLED") == "false" {
		telemetryEnabled = false
	} else if fileCfg.TelemetryEnabled != nil && !*fileCfg.TelemetryEnabled {
		// If file says false, respect it
		telemetryEnabled = false
	}

	clientID := os.Getenv("PUMBAA_CLIENT_ID")
	if clientID == "" {
		clientID = fileCfg.ClientID
	}

	// First run / Missing ClientID setup
	if clientID == "" {
		clientID = uuid.New().String()
		// Auto-enable telemetry on first run (create file with enabled=true)
		fileCfg.ClientID = clientID
		enabled := true
		fileCfg.TelemetryEnabled = &enabled // Fix: pass pointer
		_ = SaveFileConfig(fileCfg)         // Persist the new ID
	} else if fileCfg.TelemetryEnabled != nil {
		// If we had a client ID and telemetry setting is present, use it
		telemetryEnabled = *fileCfg.TelemetryEnabled
	}

	// Override via ENV if set
	if envTelemetry := os.Getenv("PUMBAA_TELEMETRY_ENABLED"); envTelemetry != "" {
		telemetryEnabled = (envTelemetry == "true")
	}

	return &Config{
		CromwellHost:     host,
		CromwellTimeout:  30 * time.Second,
		SessionDBPath:    sessionDBPath,
		LLMProvider:      llmProvider,
		OllamaHost:       ollamaHost,
		OllamaModel:      ollamaModel,
		VertexProject:    vertexProject,
		VertexLocation:   vertexLocation,
		VertexModel:      vertexModel,
		GeminiAPIKey:     geminiAPIKey,
		GeminiModel:      geminiModel,
		WDLDirectory:     wdlDirectory,
		WDLIndexPath:     wdlIndexPath,
		TelemetryEnabled: telemetryEnabled,
		ClientID:         clientID,
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
