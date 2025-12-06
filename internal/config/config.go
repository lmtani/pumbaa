// Package config provides application configuration.
package config

import (
	"os"
	"time"
)

// Config holds the application configuration.
type Config struct {
	CromwellHost    string
	CromwellTimeout time.Duration
}

// Load loads configuration from environment variables.
func Load() *Config {
	host := os.Getenv("CROMWELL_HOST")
	if host == "" {
		host = "http://localhost:8000"
	}

	return &Config{
		CromwellHost:    host,
		CromwellTimeout: 30 * time.Second,
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
