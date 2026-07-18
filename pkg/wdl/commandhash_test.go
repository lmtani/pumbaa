package wdl

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// The command template hash must reproduce what Cromwell recorded, or the
// forecast cannot detect a command change without the reference run's WDL —
// which metadata never carries for imported files.
func TestCommandTemplateHashMatchesCromwell(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "..", "example", "showcase", "wdl", "vcf_index_stats.wdl"))
	if err != nil {
		t.Skipf("showcase workflow unavailable: %v", err)
	}
	specs, err := TaskSpecs(source)
	if err != nil {
		t.Fatalf("TaskSpecs() error: %v", err)
	}

	fixture := filepath.Join("..", "..", "internal", "infrastructure", "cromwell",
		"testdata", "callcache", "run1_reference.json")
	data, err := os.ReadFile(fixture)
	if err != nil {
		t.Skipf("fixture unavailable: %v", err)
	}
	var meta struct {
		Calls map[string][]struct {
			CallCaching struct {
				Hashes map[string]any `json:"hashes"`
			} `json:"callCaching"`
		} `json:"calls"`
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		t.Fatalf("parsing fixture: %v", err)
	}

	checked := 0
	for key, calls := range meta.Calls {
		task := key[len("VcfIndexAndStats."):]
		spec, ok := specs[task]
		if !ok || len(calls) == 0 {
			continue
		}
		want, _ := calls[0].CallCaching.Hashes["command template"].(string)
		if want == "" {
			continue
		}
		if got := spec.CommandHash(); got != want {
			t.Errorf("%s: CommandHash() = %s, want %s (Cromwell's recorded hash)", task, got, want)
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("no task was actually compared against a recorded hash")
	}
}

func TestCommandTemplateHashIsStableAndSensitive(t *testing.T) {
	base := "  set -e\n  bcftools index ~{input_vcf}\n"
	// Indentation is normalised away; the content is not.
	if CommandTemplateHash(base) != CommandTemplateHash("set -e\nbcftools index ~{input_vcf}") {
		t.Error("hash must ignore leading indentation")
	}
	if CommandTemplateHash(base) == CommandTemplateHash("set -e\nbcftools index --force ~{input_vcf}") {
		t.Error("hash must change when the command does")
	}
}
