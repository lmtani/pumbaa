package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lmtani/pumbaa/internal/application/ports"
	domain "github.com/lmtani/pumbaa/internal/domain/workflow"
)

// The chain that the live Cromwell experiment used: StatsVcf consumes
// IndexVcf's outputs, so a change to IndexVcf must cascade.
const forecastWDL = `version 1.0

workflow VcfIndexAndStats {
  input {
    File input_vcf
    String output_basename
  }

  call IndexVcf {
    input:
      input_vcf = input_vcf,
      output_basename = output_basename
  }

  call StatsVcf {
    input:
      input_vcf = IndexVcf.out_vcf,
      output_basename = output_basename
  }
}

task IndexVcf {
  input {
    File input_vcf
    String output_basename
    String docker = "bcftools:1.11"
  }
  command <<< bcftools index ~{input_vcf} >>>
  runtime { docker: docker }
  output { File out_vcf = "~{output_basename}.vcf.gz" }
}

task StatsVcf {
  input {
    File input_vcf
    String output_basename
    String docker = "bcftools:1.11"
  }
  command <<< bcftools stats ~{input_vcf} >>>
  runtime { docker: docker }
  output { File report = "~{output_basename}.stats.txt" }
}
`

const (
	refVcfHash   = "41a44e64f3c014c39dfc5b7b09fbf75c"
	otherVcfHash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
)

// forecastFixture builds a reference run matching forecastWDL, with the file
// hash and backend the caller wants to exercise.
func forecastFixture(wdlSource, backend string) *domain.Workflow {
	fp := func(inputHash string) domain.CallFingerprint {
		return domain.CallFingerprint{
			"command template":              "TEMPLATE",
			"runtime attribute: docker":     "DOCKERHASH",
			"input: File input_vcf":         inputHash,
			"input: String output_basename": "BASENAME",
			"input: String docker":          "DOCKERHASH",
		}
	}
	return &domain.Workflow{
		ID:                "ref-run-id",
		Name:              "VcfIndexAndStats",
		SubmittedWorkflow: wdlSource,
		Calls: map[string][]domain.Call{
			"VcfIndexAndStats.IndexVcf": {{
				Name: "VcfIndexAndStats.IndexVcf", Backend: backend, Attempt: 1,
				Inputs:      map[string]any{"input_vcf": "/data/in.vcf.gz", "output_basename": "sample", "docker": "bcftools:1.11"},
				Fingerprint: fp(refVcfHash),
			}},
			"VcfIndexAndStats.StatsVcf": {{
				Name: "VcfIndexAndStats.StatsVcf", Backend: backend, Attempt: 1,
				Inputs:      map[string]any{"input_vcf": "/exec/out.vcf.gz", "output_basename": "sample", "docker": "bcftools:1.11"},
				Fingerprint: fp("PRODUCED_UPSTREAM"),
			}},
		},
	}
}

// forecastEnv wires the use case over a temp dir holding the WDL and inputs.
type forecastEnv struct {
	uc        *CacheForecastUseCase
	wdlPath   string
	inputPath string
}

func newForecastEnv(t *testing.T, wdlSource string, inputs map[string]any, reference *domain.Workflow, hashes map[string]string) forecastEnv {
	t.Helper()
	dir := t.TempDir()

	wdlPath := filepath.Join(dir, "wf.wdl")
	if err := os.WriteFile(wdlPath, []byte(wdlSource), 0o600); err != nil {
		t.Fatalf("writing wdl: %v", err)
	}
	inputsJSON, err := json.Marshal(inputs)
	if err != nil {
		t.Fatalf("marshalling inputs: %v", err)
	}
	inputPath := filepath.Join(dir, "inputs.json")
	if err := os.WriteFile(inputPath, inputsJSON, 0o600); err != nil {
		t.Fatalf("writing inputs: %v", err)
	}

	fp := &mockFileProvider{
		readBytesFunc: func(_ context.Context, path string) ([]byte, error) {
			return os.ReadFile(path)
		},
		getContentHashFunc: func(_ context.Context, path string) (string, error) {
			if h, ok := hashes[path]; ok {
				if h == "" {
					return "", fmt.Errorf("%w: %s", ports.ErrHashUnavailable, path)
				}
				return h, nil
			}
			return "", fmt.Errorf("%w: %s", ports.ErrFileNotFound, path)
		},
	}
	reader := &stubMetadataReader{workflow: reference}

	return forecastEnv{
		uc:        NewCacheForecastUseCase(reader, nil, fp),
		wdlPath:   wdlPath,
		inputPath: inputPath,
	}
}

type stubMetadataReader struct{ workflow *domain.Workflow }

func (s *stubMetadataReader) GetMetadata(context.Context, string) (*domain.Workflow, error) {
	return s.workflow, nil
}

func run(t *testing.T, env forecastEnv) *domain.CacheForecast {
	t.Helper()
	got, err := env.uc.Execute(context.Background(), CacheForecastInput{
		WorkflowFile: env.wdlPath,
		InputsFile:   env.inputPath,
		ReferenceID:  "ref-run-id",
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
	return got
}

func predictionFor(t *testing.T, f *domain.CacheForecast, call string) domain.CallPrediction {
	t.Helper()
	for _, p := range f.Calls {
		if p.Call == call {
			return p
		}
	}
	t.Fatalf("call %q missing from forecast %+v", call, f.Calls)
	return domain.CallPrediction{}
}

// Resubmitting the identical thing must predict full reuse — the answer that
// tells a user their run is free.
func TestForecastPredictsFullReuseForIdenticalSubmission(t *testing.T) {
	inputs := map[string]any{
		"VcfIndexAndStats.input_vcf":       "/data/in.vcf.gz",
		"VcfIndexAndStats.output_basename": "sample",
	}
	env := newForecastEnv(t, forecastWDL, inputs,
		forecastFixture(forecastWDL, "Local"),
		map[string]string{"/data/in.vcf.gz": refVcfHash})

	got := run(t, env)

	if got.Backend != domain.BackendLocal {
		t.Errorf("Backend = %v, want local", got.Backend)
	}
	for _, p := range got.Calls {
		if p.Fate != domain.FateReuse {
			t.Errorf("%s: got %v (%v), want reuse", p.Call, p.Fate, p.Reasons)
		}
	}
}

// Changing the docker of the upstream task reproduces the live experiment:
// IndexVcf is the root cause and StatsVcf cascades.
func TestForecastDetectsDockerChangeAndCascades(t *testing.T) {
	changedWDL := strings.Replace(forecastWDL,
		`String docker = "bcftools:1.11"
  }
  command <<< bcftools index`,
		`String docker = "bcftools:1.12"
  }
  command <<< bcftools index`, 1)
	if changedWDL == forecastWDL {
		t.Fatal("test setup: docker substitution did not apply")
	}

	inputs := map[string]any{
		"VcfIndexAndStats.input_vcf":       "/data/in.vcf.gz",
		"VcfIndexAndStats.output_basename": "sample",
	}
	env := newForecastEnv(t, changedWDL, inputs,
		forecastFixture(forecastWDL, "Local"),
		map[string]string{"/data/in.vcf.gz": refVcfHash})

	got := run(t, env)

	index := predictionFor(t, got, "IndexVcf")
	if index.Fate != domain.FateRerun {
		t.Errorf("IndexVcf: got %v, want rerun", index.Fate)
	}
	if len(index.Reasons) == 0 || !strings.Contains(index.Reasons[0], "docker") {
		t.Errorf("IndexVcf reasons = %v, want a docker change", index.Reasons)
	}

	stats := predictionFor(t, got, "StatsVcf")
	if stats.Fate != domain.FateRerunDownstream {
		t.Errorf("StatsVcf: got %v, want rerun (downstream)", stats.Fate)
	}
	if stats.Cause != "IndexVcf" {
		t.Errorf("StatsVcf cause = %q, want IndexVcf", stats.Cause)
	}
}

// A changed input file is detected by content, not by path: this is the case
// where the user points at a genuinely different file.
func TestForecastDetectsChangedInputFileByContent(t *testing.T) {
	inputs := map[string]any{
		"VcfIndexAndStats.input_vcf":       "/data/in.vcf.gz",
		"VcfIndexAndStats.output_basename": "sample",
	}
	env := newForecastEnv(t, forecastWDL, inputs,
		forecastFixture(forecastWDL, "Local"),
		map[string]string{"/data/in.vcf.gz": otherVcfHash})

	got := run(t, env)

	index := predictionFor(t, got, "IndexVcf")
	if index.Fate != domain.FateRerun {
		t.Fatalf("IndexVcf: got %v, want rerun", index.Fate)
	}
	if len(index.Reasons) == 0 || !strings.Contains(index.Reasons[0], "input_vcf") {
		t.Errorf("IndexVcf reasons = %v, want the input file named", index.Reasons)
	}
	if p := predictionFor(t, got, "StatsVcf"); p.Fate != domain.FateRerunDownstream {
		t.Errorf("StatsVcf: got %v, want rerun (downstream)", p.Fate)
	}
}

// Content hashing means a file moved to a new path with identical bytes still
// hits the cache — predicting a rerun there would be wrong.
func TestForecastTreatsMovedIdenticalFileAsReuse(t *testing.T) {
	inputs := map[string]any{
		"VcfIndexAndStats.input_vcf":       "/somewhere/else/in.vcf.gz",
		"VcfIndexAndStats.output_basename": "sample",
	}
	env := newForecastEnv(t, forecastWDL, inputs,
		forecastFixture(forecastWDL, "Local"),
		map[string]string{"/somewhere/else/in.vcf.gz": refVcfHash})

	got := run(t, env)

	if p := predictionFor(t, got, "IndexVcf"); p.Fate != domain.FateReuse {
		t.Errorf("IndexVcf: got %v (%v), want reuse — same content at a new path", p.Fate, p.Reasons)
	}
}

// A changed scalar input is a root cause too.
func TestForecastDetectsChangedScalarInput(t *testing.T) {
	inputs := map[string]any{
		"VcfIndexAndStats.input_vcf":       "/data/in.vcf.gz",
		"VcfIndexAndStats.output_basename": "different",
	}
	env := newForecastEnv(t, forecastWDL, inputs,
		forecastFixture(forecastWDL, "Local"),
		map[string]string{"/data/in.vcf.gz": refVcfHash})

	got := run(t, env)

	index := predictionFor(t, got, "IndexVcf")
	if index.Fate != domain.FateRerun {
		t.Errorf("IndexVcf: got %v, want rerun", index.Fate)
	}
}

// Transparency requirement: an unsupported backend must withhold the forecast
// and say so, not guess.
func TestForecastWithholdsVerdictOnUnsupportedBackend(t *testing.T) {
	inputs := map[string]any{
		"VcfIndexAndStats.input_vcf":       "/data/in.vcf.gz",
		"VcfIndexAndStats.output_basename": "sample",
	}
	env := newForecastEnv(t, forecastWDL, inputs,
		forecastFixture(forecastWDL, "SLURM"),
		map[string]string{"/data/in.vcf.gz": refVcfHash})

	got := run(t, env)

	if got.Backend.Supported() {
		t.Errorf("Backend = %v, want unsupported", got.Backend)
	}
	if len(got.Warnings) == 0 {
		t.Fatal("expected a warning naming the unsupported backend")
	}
	if !strings.Contains(got.Warnings[0], "SLURM") {
		t.Errorf("warning = %q, want it to name SLURM", got.Warnings[0])
	}
	for _, p := range got.Calls {
		if p.Fate != domain.FateUnknown {
			t.Errorf("%s: got %v, want unknown on an unsupported backend", p.Call, p.Fate)
		}
	}
}

// A GCP reference must be supported, since gs:// is the main real-world case.
func TestForecastSupportsGCPBackend(t *testing.T) {
	inputs := map[string]any{
		"VcfIndexAndStats.input_vcf":       "gs://bucket/in.vcf.gz",
		"VcfIndexAndStats.output_basename": "sample",
	}
	ref := forecastFixture(forecastWDL, "PAPIv2")
	env := newForecastEnv(t, forecastWDL, inputs, ref,
		map[string]string{"gs://bucket/in.vcf.gz": refVcfHash})

	got := run(t, env)

	if got.Backend != domain.BackendGCP {
		t.Fatalf("Backend = %v, want gcp", got.Backend)
	}
	if p := predictionFor(t, got, "IndexVcf"); p.Fate != domain.FateReuse {
		t.Errorf("IndexVcf: got %v (%v), want reuse", p.Fate, p.Reasons)
	}
}

// A file whose hash cannot be read is unknowable — never silently "reuse".
func TestForecastMarksUnknownWhenHashUnavailable(t *testing.T) {
	inputs := map[string]any{
		"VcfIndexAndStats.input_vcf":       "gs://bucket/composite.vcf.gz",
		"VcfIndexAndStats.output_basename": "sample",
	}
	env := newForecastEnv(t, forecastWDL, inputs,
		forecastFixture(forecastWDL, "PAPIv2"),
		map[string]string{"gs://bucket/composite.vcf.gz": ""})

	got := run(t, env)

	index := predictionFor(t, got, "IndexVcf")
	if index.Fate != domain.FateUnknown {
		t.Errorf("IndexVcf: got %v, want unknown when the hash cannot be read", index.Fate)
	}
	if p := predictionFor(t, got, "StatsVcf"); p.Fate != domain.FateUnknown {
		t.Errorf("StatsVcf: got %v, want unknown downstream of an unknown", p.Fate)
	}
}

// Without hashes in the reference there is nothing to compare against.
func TestForecastMarksUnknownWhenReferenceHasNoHashes(t *testing.T) {
	ref := forecastFixture(forecastWDL, "Local")
	for key := range ref.Calls {
		calls := ref.Calls[key]
		calls[0].Fingerprint = nil
		ref.Calls[key] = calls
	}

	inputs := map[string]any{
		"VcfIndexAndStats.input_vcf":       "/data/in.vcf.gz",
		"VcfIndexAndStats.output_basename": "sample",
	}
	env := newForecastEnv(t, forecastWDL, inputs, ref,
		map[string]string{"/data/in.vcf.gz": refVcfHash})

	got := run(t, env)

	for _, p := range got.Calls {
		if p.Fate != domain.FateUnknown {
			t.Errorf("%s: got %v, want unknown without reference hashes", p.Call, p.Fate)
		}
	}
}

func TestForecastCountsSummariseTheRun(t *testing.T) {
	changedWDL := strings.Replace(forecastWDL, "bcftools index", "bcftools index --force", 1)
	inputs := map[string]any{
		"VcfIndexAndStats.input_vcf":       "/data/in.vcf.gz",
		"VcfIndexAndStats.output_basename": "sample",
	}
	env := newForecastEnv(t, changedWDL, inputs,
		forecastFixture(forecastWDL, "Local"),
		map[string]string{"/data/in.vcf.gz": refVcfHash})

	got := run(t, env)

	counts := got.Counts()
	if counts[domain.FateRerun] != 1 || counts[domain.FateRerunDownstream] != 1 {
		t.Errorf("Counts() = %v, want 1 rerun and 1 downstream", counts)
	}
	roots := got.RootCauses()
	if len(roots) != 1 || roots[0].Call != "IndexVcf" {
		t.Errorf("RootCauses() = %+v, want only IndexVcf", roots)
	}
	if !strings.Contains(strings.Join(roots[0].Reasons, ","), "command template") {
		t.Errorf("reasons = %v, want the command template change", roots[0].Reasons)
	}
}

// Cromwell accepts call-scoped overrides in the inputs JSON that never appear
// as call bindings in the WDL. Missing those would predict reuse for a
// submission that changed a task's image — the exact way the live experiment
// forced a cache miss.
func TestForecastDetectsCallScopedInputOverride(t *testing.T) {
	inputs := map[string]any{
		"VcfIndexAndStats.input_vcf":       "/data/in.vcf.gz",
		"VcfIndexAndStats.output_basename": "sample",
		"VcfIndexAndStats.IndexVcf.docker": "bcftools:1.12",
	}
	env := newForecastEnv(t, forecastWDL, inputs,
		forecastFixture(forecastWDL, "Local"),
		map[string]string{"/data/in.vcf.gz": refVcfHash})

	got := run(t, env)

	index := predictionFor(t, got, "IndexVcf")
	if index.Fate != domain.FateRerun {
		t.Fatalf("IndexVcf: got %v (%v), want rerun from the docker override",
			index.Fate, index.Reasons)
	}
	if !strings.Contains(strings.Join(index.Reasons, ","), "docker") {
		t.Errorf("IndexVcf reasons = %v, want the docker input named", index.Reasons)
	}
	if p := predictionFor(t, got, "StatsVcf"); p.Fate != domain.FateRerunDownstream {
		t.Errorf("StatsVcf: got %v, want rerun (downstream)", p.Fate)
	}
}
