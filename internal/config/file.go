// Package config provides application configuration.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// FileConfig represents the structure of the config file.
type FileConfig struct {
	LLMProvider string `yaml:"llm_provider,omitempty"`

	// Cromwell
	CromwellHost string `yaml:"cromwell_host,omitempty"`

	// Ollama
	OllamaHost  string `yaml:"ollama_host,omitempty"`
	OllamaModel string `yaml:"ollama_model,omitempty"`

	// Vertex AI
	VertexProject  string `yaml:"vertex_project,omitempty"`
	VertexLocation string `yaml:"vertex_location,omitempty"`
	VertexModel    string `yaml:"vertex_model,omitempty"`

	// Gemini API
	GeminiAPIKey string `yaml:"gemini_api_key,omitempty"`
	GeminiModel  string `yaml:"gemini_model,omitempty"`

	// WDL Context
	WDLDirectory string `yaml:"wdl_directory,omitempty"`
}

// DefaultConfigPath returns the default path for the config file.
func DefaultConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".pumbaa", "config.yaml")
}

// LoadFileConfig loads configuration from the YAML config file.
// Returns an empty FileConfig if the file doesn't exist.
func LoadFileConfig() (*FileConfig, error) {
	path := DefaultConfigPath()
	return LoadFileConfigFrom(path)
}

// LoadFileConfigFrom loads configuration from a specific path.
func LoadFileConfigFrom(path string) (*FileConfig, error) {
	cfg := &FileConfig{}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil // Return empty config if file doesn't exist
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return cfg, nil
}

// SaveFileConfig saves configuration to the YAML config file.
func SaveFileConfig(cfg *FileConfig) error {
	path := DefaultConfigPath()
	return SaveFileConfigTo(cfg, path)
}

// SaveFileConfigTo saves configuration to a specific path.
func SaveFileConfigTo(cfg *FileConfig, path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}

	// Write with restricted permissions (only owner can read/write)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetConfigValue returns a specific config value by key.
func (c *FileConfig) GetValue(key string) (string, bool) {
	switch key {
	case "llm_provider":
		return c.LLMProvider, c.LLMProvider != ""
	case "cromwell_host":
		return c.CromwellHost, c.CromwellHost != ""
	case "ollama_host":
		return c.OllamaHost, c.OllamaHost != ""
	case "ollama_model":
		return c.OllamaModel, c.OllamaModel != ""
	case "vertex_project":
		return c.VertexProject, c.VertexProject != ""
	case "vertex_location":
		return c.VertexLocation, c.VertexLocation != ""
	case "vertex_model":
		return c.VertexModel, c.VertexModel != ""
	case "gemini_api_key":
		return c.GeminiAPIKey, c.GeminiAPIKey != ""
	case "gemini_model":
		return c.GeminiModel, c.GeminiModel != ""
	case "wdl_directory":
		return c.WDLDirectory, c.WDLDirectory != ""
	default:
		return "", false
	}
}

// SetValue sets a config value by key.
func (c *FileConfig) SetValue(key, value string) error {
	switch key {
	case "llm_provider":
		if value != "ollama" && value != "vertex" && value != "gemini" {
			return fmt.Errorf("invalid provider: %s (must be ollama, vertex, or gemini)", value)
		}
		c.LLMProvider = value
	case "cromwell_host":
		c.CromwellHost = value
	case "ollama_host":
		c.OllamaHost = value
	case "ollama_model":
		c.OllamaModel = value
	case "vertex_project":
		c.VertexProject = value
	case "vertex_location":
		c.VertexLocation = value
	case "vertex_model":
		c.VertexModel = value
	case "gemini_api_key":
		c.GeminiAPIKey = value
	case "gemini_model":
		c.GeminiModel = value
	case "wdl_directory":
		c.WDLDirectory = value
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}
	return nil
}

// AllKeys returns all valid config keys.
func AllKeys() []string {
	return []string{
		"llm_provider",
		"cromwell_host",
		"ollama_host",
		"ollama_model",
		"vertex_project",
		"vertex_location",
		"vertex_model",
		"gemini_api_key",
		"gemini_model",
		"wdl_directory",
	}
}
