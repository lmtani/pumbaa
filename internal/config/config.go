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
	OllamaHost      string
	OllamaModel     string
	SessionDBPath   string
}

// Load loads configuration from environment variables.
func Load() *Config {
	host := os.Getenv("CROMWELL_HOST")
	if host == "" {
		host = "http://localhost:8000"
	}

	ollamaHost := os.Getenv("OLLAMA_HOST")
	if ollamaHost == "" {
		ollamaHost = "http://localhost:11434"
	}

	ollamaModel := os.Getenv("OLLAMA_MODEL")
	if ollamaModel == "" {
		ollamaModel = "llama3.2:3b"
	}

	sessionDBPath := os.Getenv("PUMBAA_SESSION_DB")
	if sessionDBPath == "" {
		home, _ := os.UserHomeDir()
		sessionDBPath = filepath.Join(home, ".pumbaa", "sessions.db")
	}

	return &Config{
		CromwellHost:    host,
		CromwellTimeout: 30 * time.Second,
		OllamaHost:      ollamaHost,
		OllamaModel:     ollamaModel,
		SessionDBPath:   sessionDBPath,
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
