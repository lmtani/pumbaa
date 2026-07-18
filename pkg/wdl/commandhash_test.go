package wdl

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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
		got, canonical := spec.CommandHash()
		if !canonical {
			t.Errorf("%s: command should be canonically renderable", task)
			continue
		}
		if got != want {
			t.Errorf("%s: CommandHash() = %s, want %s (Cromwell's recorded hash)", task, got, want)
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("no task was actually compared against a recorded hash")
	}
}

func TestCommandTemplateHashIsStableAndSensitive(t *testing.T) {
	hash := func(s string) string { h, _ := CommandTemplateHash(s); return h }

	base := "  set -e\n  bcftools index ~{input_vcf}\n"
	// The common indent is removed; the content is not.
	if hash(base) != hash("set -e\nbcftools index ~{input_vcf}") {
		t.Error("hash must ignore the common leading indentation")
	}
	if hash(base) == hash("set -e\nbcftools index --force ~{input_vcf}") {
		t.Error("hash must change when the command does")
	}
	// Relative indentation is part of the command and must survive dedenting.
	nested := "if true; then\n  echo hi\nfi"
	flat := "if true; then\necho hi\nfi"
	if hash(nested) == hash(flat) {
		t.Error("relative indentation must be preserved")
	}
}

// A command whose placeholders are bare references can be rendered exactly;
// one that interpolates an expression cannot, and must say so rather than
// produce a hash that would read as a change.
func TestCommandTemplateHashReportsNonCanonicalPlaceholders(t *testing.T) {
	tests := []struct {
		command string
		want    bool
	}{
		{"echo ~{sample}", true},
		{"echo ~{ sample }", true},
		{"echo ~{obj.field}", true},
		{"echo ~{sep=\" \" bams}", true},
		{"echo ~{default=\"none\" tag}", true},
		{"echo ~{if defined(x) then \"-f \" + x else \"\"}", false},
		{"echo ~{basename(path)}", false},
		{"echo ~{a + b}", false},
	}
	for _, tt := range tests {
		if _, got := CommandTemplateHash(tt.command); got != tt.want {
			t.Errorf("CommandTemplateHash(%q) canonical = %v, want %v", tt.command, got, tt.want)
		}
	}
}

// The parser must hand back the command exactly as written. Rebuilding it from
// tokens drops whitespace on the hidden channel — `~{sep="," xs}` came back as
// `~{sep=","xs}` — which silently broke every command-template hash for tasks
// using placeholder options.
func TestParserPreservesCommandWhitespaceVerbatim(t *testing.T) {
	const src = `version 1.0

workflow W { call T }

task T {
  input { Array[File] bams }
  command <<<
    tool \
      -input ~{sep="," bams} \
      -flag
  >>>
  output { File out = "o" }
}
`
	specs, err := TaskSpecs([]byte(src))
	if err != nil {
		t.Fatalf("TaskSpecs() error: %v", err)
	}
	got := specs["T"].Command
	if !strings.Contains(got, `~{sep="," bams}`) {
		t.Errorf("command lost source whitespace inside the placeholder:\n%q", got)
	}
	// Relative indentation of the continuation lines must survive too.
	if !strings.Contains(got, "\n      -flag") {
		t.Errorf("command lost relative indentation:\n%q", got)
	}
}

func TestCommandTemplateHashHandlesCurlyBraceCommands(t *testing.T) {
	const src = `version 1.0

workflow W { call T }

task T {
  input { String sample }
  command {
    echo ${sample}
  }
  output { File out = "o" }
}
`
	specs, err := TaskSpecs([]byte(src))
	if err != nil {
		t.Fatalf("TaskSpecs() error: %v", err)
	}
	if got := strings.TrimSpace(specs["T"].Command); got != "echo ${sample}" {
		t.Errorf("curly-brace command body = %q, want the inner text", got)
	}
	if _, ok := specs["T"].CommandHash(); !ok {
		t.Error("a bare ${ref} placeholder should be canonically renderable")
	}
}
