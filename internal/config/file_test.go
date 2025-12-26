package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFileConfigFrom_NonExistent(t *testing.T) {
	cfg, err := LoadFileConfigFrom("/nonexistent/path/config.yaml")
	if err != nil {
		t.Fatalf("expected no error for non-existent file, got: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected empty config, got nil")
	}
}

func TestLoadFileConfigFrom_ValidYAML(t *testing.T) {
	// Create temp file with valid YAML
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")

	content := `
llm_provider: vertex
cromwell_host: http://test:8000
ollama_host: http://ollama:11434
ollama_model: llama3
vertex_project: my-project
vertex_location: us-east1
telemetry_enabled: false
`
	if err := os.WriteFile(cfgPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadFileConfigFrom(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.LLMProvider != "vertex" {
		t.Errorf("expected llm_provider=vertex, got %s", cfg.LLMProvider)
	}
	if cfg.CromwellHost != "http://test:8000" {
		t.Errorf("expected cromwell_host=http://test:8000, got %s", cfg.CromwellHost)
	}
	if cfg.VertexProject != "my-project" {
		t.Errorf("expected vertex_project=my-project, got %s", cfg.VertexProject)
	}
	if cfg.TelemetryEnabled == nil || *cfg.TelemetryEnabled != false {
		t.Error("expected telemetry_enabled=false")
	}
}

func TestLoadFileConfigFrom_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")

	// Write invalid YAML
	if err := os.WriteFile(cfgPath, []byte("invalid: yaml: content: ["), 0600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	_, err := LoadFileConfigFrom(cfgPath)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func TestSaveFileConfigTo(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "subdir", "config.yaml")

	cfg := &FileConfig{
		LLMProvider:  "ollama",
		CromwellHost: "http://localhost:8000",
		OllamaModel:  "llama3.2",
	}

	if err := SaveFileConfigTo(cfg, cfgPath); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		t.Fatal("config file was not created")
	}

	// Load it back and verify
	loaded, err := LoadFileConfigFrom(cfgPath)
	if err != nil {
		t.Fatalf("failed to load saved config: %v", err)
	}

	if loaded.LLMProvider != cfg.LLMProvider {
		t.Errorf("expected llm_provider=%s, got %s", cfg.LLMProvider, loaded.LLMProvider)
	}
	if loaded.OllamaModel != cfg.OllamaModel {
		t.Errorf("expected ollama_model=%s, got %s", cfg.OllamaModel, loaded.OllamaModel)
	}
}

func TestFileConfig_GetValue(t *testing.T) {
	enabled := true
	cfg := &FileConfig{
		LLMProvider:      "gemini",
		CromwellHost:     "http://cromwell:8000",
		TelemetryEnabled: &enabled,
		ClientID:         "test-client-id",
	}

	tests := []struct {
		key     string
		wantVal string
		wantOk  bool
	}{
		{"llm_provider", "gemini", true},
		{"cromwell_host", "http://cromwell:8000", true},
		{"telemetry_enabled", "true", true},
		{"client_id", "test-client-id", true},
		{"ollama_host", "", false}, // Not set
		{"unknown_key", "", false}, // Unknown key
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			val, ok := cfg.GetValue(tt.key)
			if ok != tt.wantOk {
				t.Errorf("GetValue(%q) ok = %v, want %v", tt.key, ok, tt.wantOk)
			}
			if val != tt.wantVal {
				t.Errorf("GetValue(%q) = %q, want %q", tt.key, val, tt.wantVal)
			}
		})
	}
}

func TestFileConfig_SetValue(t *testing.T) {
	cfg := &FileConfig{}

	tests := []struct {
		key     string
		value   string
		wantErr bool
	}{
		{"llm_provider", "vertex", false},
		{"llm_provider", "ollama", false},
		{"llm_provider", "gemini", false},
		{"llm_provider", "invalid", true}, // Invalid provider
		{"cromwell_host", "http://test:8000", false},
		{"ollama_host", "http://ollama:11434", false},
		{"telemetry_enabled", "true", false},
		{"unknown_key", "value", true}, // Unknown key
	}

	for _, tt := range tests {
		t.Run(tt.key+"_"+tt.value, func(t *testing.T) {
			err := cfg.SetValue(tt.key, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetValue(%q, %q) error = %v, wantErr %v", tt.key, tt.value, err, tt.wantErr)
			}
		})
	}

	// Verify final state
	if cfg.CromwellHost != "http://test:8000" {
		t.Errorf("expected cromwell_host=http://test:8000, got %s", cfg.CromwellHost)
	}
}

func TestAllKeys(t *testing.T) {
	keys := AllKeys()
	if len(keys) == 0 {
		t.Fatal("AllKeys() returned empty list")
	}

	// Check some expected keys are present
	expectedKeys := []string{"llm_provider", "cromwell_host", "telemetry_enabled"}
	for _, expected := range expectedKeys {
		found := false
		for _, k := range keys {
			if k == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected key %q not found in AllKeys()", expected)
		}
	}
}
