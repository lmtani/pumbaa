package config

import (
	"os"
	"testing"
)

// clearEnvVars clears all Pumbaa-related env vars for clean test state
// It also sets HOME to a temp dir to avoid reading user's config file
func clearEnvVars(t *testing.T) func() {
	t.Helper()
	envVars := []string{
		"CROMWELL_HOST",
		"PUMBAA_SESSION_DB",
		"PUMBAA_LLM_PROVIDER",
		"OLLAMA_HOST",
		"OLLAMA_MODEL",
		"VERTEX_PROJECT",
		"VERTEX_LOCATION",
		"VERTEX_MODEL",
		"GEMINI_API_KEY",
		"GEMINI_MODEL",
		"PUMBAA_WDL_DIR",
		"PUMBAA_WDL_INDEX",
		"PUMBAA_TELEMETRY_ENABLED",
		"PUMBAA_CLIENT_ID",
	}

	oldValues := make(map[string]string)
	for _, v := range envVars {
		oldValues[v] = os.Getenv(v)
		os.Unsetenv(v)
	}

	// Override HOME to temp dir to avoid reading user's config file
	oldHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)

	return func() {
		os.Setenv("HOME", oldHome)
		for k, v := range oldValues {
			if v != "" {
				os.Setenv(k, v)
			}
		}
	}
}

func TestLoad_Defaults(t *testing.T) {
	cleanup := clearEnvVars(t)
	defer cleanup()

	cfg := Load()

	// Check defaults
	if cfg.CromwellHost != "http://localhost:8000" {
		t.Errorf("expected default CromwellHost=http://localhost:8000, got %s", cfg.CromwellHost)
	}
	if cfg.LLMProvider != "ollama" {
		t.Errorf("expected default LLMProvider=ollama, got %s", cfg.LLMProvider)
	}
	if cfg.OllamaHost != "http://localhost:11434" {
		t.Errorf("expected default OllamaHost=http://localhost:11434, got %s", cfg.OllamaHost)
	}
	if cfg.OllamaModel != "llama3.2:3b" {
		t.Errorf("expected default OllamaModel=llama3.2:3b, got %s", cfg.OllamaModel)
	}
	if cfg.VertexLocation != "us-central1" {
		t.Errorf("expected default VertexLocation=us-central1, got %s", cfg.VertexLocation)
	}
	if cfg.VertexModel != "gemini-2.0-flash" {
		t.Errorf("expected default VertexModel=gemini-2.0-flash, got %s", cfg.VertexModel)
	}
	if cfg.GeminiModel != "gemini-2.0-flash" {
		t.Errorf("expected default GeminiModel=gemini-2.0-flash, got %s", cfg.GeminiModel)
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	cleanup := clearEnvVars(t)
	defer cleanup()

	// Set env vars
	os.Setenv("CROMWELL_HOST", "http://env-cromwell:8000")
	os.Setenv("PUMBAA_LLM_PROVIDER", "vertex")
	os.Setenv("OLLAMA_HOST", "http://env-ollama:11434")
	os.Setenv("VERTEX_PROJECT", "env-project")
	os.Setenv("VERTEX_LOCATION", "europe-west1")
	os.Setenv("GEMINI_API_KEY", "test-api-key")

	cfg := Load()

	if cfg.CromwellHost != "http://env-cromwell:8000" {
		t.Errorf("expected CromwellHost from env, got %s", cfg.CromwellHost)
	}
	if cfg.LLMProvider != "vertex" {
		t.Errorf("expected LLMProvider from env, got %s", cfg.LLMProvider)
	}
	if cfg.OllamaHost != "http://env-ollama:11434" {
		t.Errorf("expected OllamaHost from env, got %s", cfg.OllamaHost)
	}
	if cfg.VertexProject != "env-project" {
		t.Errorf("expected VertexProject from env, got %s", cfg.VertexProject)
	}
	if cfg.VertexLocation != "europe-west1" {
		t.Errorf("expected VertexLocation from env, got %s", cfg.VertexLocation)
	}
	if cfg.GeminiAPIKey != "test-api-key" {
		t.Errorf("expected GeminiAPIKey from env, got %s", cfg.GeminiAPIKey)
	}
}

func TestLoad_TelemetryEnvOverride(t *testing.T) {
	cleanup := clearEnvVars(t)
	defer cleanup()

	// Test that env var overrides default telemetry
	os.Setenv("PUMBAA_TELEMETRY_ENABLED", "false")

	cfg := Load()

	if cfg.TelemetryEnabled != false {
		t.Errorf("expected TelemetryEnabled=false when env is 'false', got %v", cfg.TelemetryEnabled)
	}
}

func TestFromFlags_HostOverride(t *testing.T) {
	cleanup := clearEnvVars(t)
	defer cleanup()

	cfg := FromFlags("http://custom-host:9000")

	if cfg.CromwellHost != "http://custom-host:9000" {
		t.Errorf("expected CromwellHost from flags, got %s", cfg.CromwellHost)
	}
}

func TestFromFlags_EmptyHost(t *testing.T) {
	cleanup := clearEnvVars(t)
	defer cleanup()

	// When host is empty, should use default
	cfg := FromFlags("")

	if cfg.CromwellHost != "http://localhost:8000" {
		t.Errorf("expected default CromwellHost, got %s", cfg.CromwellHost)
	}
}

func TestLoad_ClientIDGeneration(t *testing.T) {
	cleanup := clearEnvVars(t)
	defer cleanup()

	cfg := Load()

	// ClientID should be generated if not set
	if cfg.ClientID == "" {
		t.Error("expected ClientID to be generated, got empty string")
	}

	// Should be a valid UUID format (36 chars with dashes)
	if len(cfg.ClientID) != 36 {
		t.Errorf("expected ClientID to be UUID format (36 chars), got %d chars", len(cfg.ClientID))
	}
}
