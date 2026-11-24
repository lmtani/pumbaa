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
		"PUMBAA_TELEMETRY_ENDPOINT",
		"PUMBAA_TELEMETRY_KEY",
		"PUMBAA_CLIENT_ID",
		"PUMBAA_HISTORY_KEEP_FILES",
	}

	oldValues := make(map[string]string)
	for _, v := range envVars {
		oldValues[v] = os.Getenv(v)
		_ = os.Unsetenv(v)
	}

	// Override HOME to temp dir to avoid reading user's config file
	oldHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	_ = os.Setenv("HOME", tmpDir)

	return func() {
		_ = os.Setenv("HOME", oldHome)
		for k, v := range oldValues {
			if v != "" {
				_ = os.Setenv(k, v)
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
	if cfg.VertexModel != "gemini-2.5-flash" {
		t.Errorf("expected default VertexModel=gemini-2.5-flash, got %s", cfg.VertexModel)
	}
	if cfg.GeminiModel != "gemini-2.5-flash" {
		t.Errorf("expected default GeminiModel=gemini-2.5-flash, got %s", cfg.GeminiModel)
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	cleanup := clearEnvVars(t)
	defer cleanup()

	// Set env vars
	_ = os.Setenv("CROMWELL_HOST", "http://env-cromwell:8000")
	_ = os.Setenv("PUMBAA_LLM_PROVIDER", "vertex")
	_ = os.Setenv("OLLAMA_HOST", "http://env-ollama:11434")
	_ = os.Setenv("VERTEX_PROJECT", "env-project")
	_ = os.Setenv("VERTEX_LOCATION", "europe-west1")
	_ = os.Setenv("GEMINI_API_KEY", "test-api-key")

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
	_ = os.Setenv("PUMBAA_TELEMETRY_ENABLED", "false")

	cfg := Load()

	if cfg.TelemetryEnabled != false {
		t.Errorf("expected TelemetryEnabled=false when env is 'false', got %v", cfg.TelemetryEnabled)
	}
}

// The documented precedence is env > file > default, in both directions:
// the env var must be able to turn keeping back on as well as off.
func TestLoad_HistoryKeepFilesPrecedence(t *testing.T) {
	for _, tc := range []struct {
		name string
		file *bool
		env  string
		want bool
	}{
		{name: "default", want: true},
		{name: "file off", file: boolPtr(false), want: false},
		{name: "env off", env: "false", want: false},
		{name: "env off, other spelling", env: "0", want: false},
		{name: "env on overrides file off", file: boolPtr(false), env: "true", want: true},
		{name: "env off overrides file on", file: boolPtr(true), env: "FALSE", want: false},
		{name: "unreadable env falls back to file", file: boolPtr(false), env: "nope", want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := clearEnvVars(t)
			defer cleanup()
			if tc.file != nil {
				if err := SaveFileConfig(&FileConfig{ClientID: "test", HistoryKeepFiles: tc.file}); err != nil {
					t.Fatalf("SaveFileConfig: %v", err)
				}
			}
			if tc.env != "" {
				_ = os.Setenv("PUMBAA_HISTORY_KEEP_FILES", tc.env)
			}

			if got := Load().HistoryKeepFiles; got != tc.want {
				t.Errorf("HistoryKeepFiles = %v, want %v", got, tc.want)
			}
		})
	}
}

func boolPtr(b bool) *bool { return &b }

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
