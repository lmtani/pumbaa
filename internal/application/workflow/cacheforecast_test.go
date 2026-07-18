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
	"github.com/lmtani/pumbaa/pkg/wdl"
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
	// Command-template hashes must be the ones Cromwell would really record,
	// because that is what the forecast compares against — and because the
	// forecast refuses to report command changes until it has reproduced a
	// known hash for this reference.
	specs, err := wdl.TaskSpecs([]byte(wdlSource))
	if err != nil {
		panic("forecast fixture: unparseable WDL: " + err.Error())
	}
	fp := func(task, inputHash string) domain.CallFingerprint {
		return domain.CallFingerprint{
			"command template":              commandHashOf(specs[task]),
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
				Fingerprint: fp("IndexVcf", refVcfHash),
			}},
			"VcfIndexAndStats.StatsVcf": {{
				Name: "VcfIndexAndStats.StatsVcf", Backend: backend, Attempt: 1,
				Inputs:      map[string]any{"input_vcf": "/exec/out.vcf.gz", "output_basename": "sample", "docker": "bcftools:1.11"},
				Fingerprint: fp("StatsVcf", "PRODUCED_UPSTREAM"),
			}},
		},
	}
}

// commandHashOf is the hash a reference run would have recorded for a task.
func commandHashOf(spec wdl.TaskSpec) string {
	h, _ := spec.CommandHash()
	return h
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
		getDigestsFunc: func(_ context.Context, path string) (ports.FileDigests, error) {
			if h, ok := hashes[path]; ok {
				if h == "" {
					return ports.FileDigests{}, fmt.Errorf("%w: %s", ports.ErrHashUnavailable, path)
				}
				return ports.FileDigests{MD5: h}, nil
			}
			return ports.FileDigests{}, fmt.Errorf("%w: %s", ports.ErrFileNotFound, path)
		},
	}
	reader := &stubMetadataReader{workflow: reference}

	return forecastEnv{
		uc:        NewCacheForecastUseCase(reader, nil, nil, fp),
		wdlPath:   wdlPath,
		inputPath: inputPath,
	}
}

// newForecastEnvWithDigests is newForecastEnv with full control over the
// digests a backend reports, for exercising the GCS crc32c path.
func newForecastEnvWithDigests(t *testing.T, wdlSource string, inputs map[string]any, reference *domain.Workflow, digests map[string]ports.FileDigests) forecastEnv {
	t.Helper()
	env := newForecastEnv(t, wdlSource, inputs, reference, nil)
	fp := &mockFileProvider{
		readBytesFunc: func(_ context.Context, path string) ([]byte, error) { return os.ReadFile(path) },
		getDigestsFunc: func(_ context.Context, path string) (ports.FileDigests, error) {
			if d, ok := digests[path]; ok {
				return d, nil
			}
			return ports.FileDigests{}, fmt.Errorf("%w: %s", ports.ErrFileNotFound, path)
		},
	}
	env.uc = NewCacheForecastUseCase(&stubMetadataReader{workflow: reference}, nil, nil, fp)
	return env
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

// --- Subworkflow coverage -------------------------------------------------

const subWDL = `version 1.0

workflow AlignSample {
  input {
    File reads
    String sample
  }
  call Align { input: reads = reads, sample = sample }
  call Sort { input: bam = Align.out, sample = sample }
  output {
    File sorted = Sort.out
  }
}

task Align {
  input { File reads String sample String docker = "bwa:0.7.17" }
  command <<< bwa mem ~{reads} >>>
  runtime { docker: docker }
  output { File out = "~{sample}.bam" }
}

task Sort {
  input { File bam String sample String docker = "samtools:1.21" }
  command <<< samtools sort ~{bam} >>>
  runtime { docker: docker }
  output { File out = "~{sample}.sorted.bam" }
}
`

const parentWDL = `version 1.0

import "sub.wdl" as sub

workflow Cohort {
  input {
    File reads
    String sample
  }
  call sub.AlignSample { input: reads = reads, sample = sample }
  call Report { input: bam = AlignSample.sorted, sample = sample }
}

task Report {
  input { File bam String sample String docker = "report:1.0" }
  command <<< echo ~{bam} >>>
  runtime { docker: docker }
  output { File out = "~{sample}.txt" }
}
`

// subReference models what Cromwell returns with expandSubWorkflows=true: the
// subworkflow call carries its own metadata, and the leaves live inside it.
func subReference(readsHash string) *domain.Workflow {
	subSpecs, err := wdl.TaskSpecs([]byte(subWDL))
	if err != nil {
		panic("sub fixture: unparseable subworkflow WDL: " + err.Error())
	}
	parentSpecs, err := wdl.TaskSpecs([]byte(parentWDL))
	if err != nil {
		panic("sub fixture: unparseable parent WDL: " + err.Error())
	}
	fp := func(task string, entries map[string]string) domain.CallFingerprint {
		spec, ok := subSpecs[task]
		if !ok {
			spec = parentSpecs[task]
		}
		out := domain.CallFingerprint{
			"command template":          commandHashOf(spec),
			"runtime attribute: docker": "D",
		}
		for k, v := range entries {
			out[k] = v
		}
		return out
	}
	align := domain.Call{
		Name: "AlignSample.Align", Backend: "Local", Attempt: 1,
		Inputs: map[string]any{"reads": "/data/r.fq", "sample": "S1", "docker": "bwa:0.7.17"},
		Fingerprint: fp("Align", map[string]string{
			"input: File reads":    readsHash,
			"input: String sample": "SAMPLE",
			"input: String docker": "DOCK_BWA",
		}),
	}
	sortCall := domain.Call{
		Name: "AlignSample.Sort", Backend: "Local", Attempt: 1,
		Inputs: map[string]any{"bam": "/exec/S1.bam", "sample": "S1", "docker": "samtools:1.21"},
		Fingerprint: fp("Sort", map[string]string{
			"input: File bam":      "UPSTREAM",
			"input: String sample": "SAMPLE",
			"input: String docker": "DOCK_SAM",
		}),
	}
	report := domain.Call{
		Name: "Cohort.Report", Backend: "Local", Attempt: 1,
		Inputs: map[string]any{"bam": "/exec/S1.sorted.bam", "sample": "S1", "docker": "report:1.0"},
		Fingerprint: fp("Report", map[string]string{
			"input: File bam":      "UPSTREAM2",
			"input: String sample": "SAMPLE",
			"input: String docker": "DOCK_REP",
		}),
	}
	return &domain.Workflow{
		ID:                "ref-sub",
		Name:              "Cohort",
		SubmittedWorkflow: parentWDL,
		Calls: map[string][]domain.Call{
			"Cohort.AlignSample": {{
				Name: "Cohort.AlignSample", Backend: "Local", Attempt: 1,
				SubWorkflowMetadata: &domain.Workflow{
					Name: "AlignSample",
					Calls: map[string][]domain.Call{
						"AlignSample.Align": {align},
						"AlignSample.Sort":  {sortCall},
					},
				},
			}},
			"Cohort.Report": {report},
		},
	}
}

// newSubEnv writes the parent and the subworkflow side by side, the way a
// checkout looks, so imports resolve without a zip.
func newSubEnv(t *testing.T, parentSource, subSource string, inputs map[string]any, ref *domain.Workflow, hashes map[string]string) forecastEnv {
	t.Helper()
	dir := t.TempDir()
	wdlPath := filepath.Join(dir, "cohort.wdl")
	if err := os.WriteFile(wdlPath, []byte(parentSource), 0o600); err != nil {
		t.Fatalf("writing parent: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sub.wdl"), []byte(subSource), 0o600); err != nil {
		t.Fatalf("writing sub: %v", err)
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
		readBytesFunc: func(_ context.Context, path string) ([]byte, error) { return os.ReadFile(path) },
		getDigestsFunc: func(_ context.Context, path string) (ports.FileDigests, error) {
			if h, ok := hashes[path]; ok {
				return ports.FileDigests{MD5: h}, nil
			}
			return ports.FileDigests{}, fmt.Errorf("%w: %s", ports.ErrFileNotFound, path)
		},
	}
	return forecastEnv{
		uc:        NewCacheForecastUseCase(&stubMetadataReader{workflow: ref}, nil, nil, fp),
		wdlPath:   wdlPath,
		inputPath: inputPath,
	}
}

func subInputs() map[string]any {
	return map[string]any{"Cohort.reads": "/data/r.fq", "Cohort.sample": "S1"}
}

// Calls inside a subworkflow must be predicted individually — they are the
// units Cromwell actually caches.
func TestForecastCoversCallsInsideSubworkflow(t *testing.T) {
	env := newSubEnv(t, parentWDL, subWDL, subInputs(), subReference("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		map[string]string{"/data/r.fq": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"})

	got := run(t, env)

	for _, name := range []string{"AlignSample.Align", "AlignSample.Sort", "Report"} {
		p := predictionFor(t, got, name)
		if p.Fate != domain.FateReuse {
			t.Errorf("%s: got %v (%v), want reuse", name, p.Fate, p.Reasons)
		}
	}
	if len(got.Calls) != 3 {
		t.Errorf("got %d predictions, want 3 leaf calls", len(got.Calls))
	}
}

// A change to a task inside a subworkflow must be named as the root cause, and
// cascade both within the subworkflow and out to the parent's calls.
func TestForecastCascadesOutOfSubworkflow(t *testing.T) {
	changedSub := strings.Replace(subWDL, `String docker = "bwa:0.7.17"`, `String docker = "bwa:0.7.18"`, 1)
	if changedSub == subWDL {
		t.Fatal("test setup: docker substitution did not apply")
	}
	env := newSubEnv(t, parentWDL, changedSub, subInputs(), subReference("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		map[string]string{"/data/r.fq": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"})

	got := run(t, env)

	align := predictionFor(t, got, "AlignSample.Align")
	if align.Fate != domain.FateRerun {
		t.Fatalf("AlignSample.Align: got %v (%v), want rerun", align.Fate, align.Reasons)
	}
	if !strings.Contains(strings.Join(align.Reasons, ","), "docker") {
		t.Errorf("AlignSample.Align reasons = %v, want a docker change", align.Reasons)
	}

	// Inside the subworkflow.
	sortPred := predictionFor(t, got, "AlignSample.Sort")
	if sortPred.Fate != domain.FateRerunDownstream || sortPred.Cause != "AlignSample.Align" {
		t.Errorf("AlignSample.Sort: got %v cause=%q, want downstream of AlignSample.Align",
			sortPred.Fate, sortPred.Cause)
	}
	// And out into the parent, blaming the root rather than the nearest hop.
	report := predictionFor(t, got, "Report")
	if report.Fate != domain.FateRerunDownstream || report.Cause != "AlignSample.Align" {
		t.Errorf("Report: got %v cause=%q, want downstream of AlignSample.Align",
			report.Fate, report.Cause)
	}
}

// A change confined to one leaf must not drag in a sibling the consumer does
// not read from.
func TestForecastDoesNotCascadeFromUnrelatedSubworkflowLeaf(t *testing.T) {
	changedSub := strings.Replace(subWDL, `String docker = "samtools:1.21"`, `String docker = "samtools:1.22"`, 1)
	if changedSub == subWDL {
		t.Fatal("test setup: docker substitution did not apply")
	}
	env := newSubEnv(t, parentWDL, changedSub, subInputs(), subReference("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		map[string]string{"/data/r.fq": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"})

	got := run(t, env)

	// Align feeds Sort, not the other way round.
	if p := predictionFor(t, got, "AlignSample.Align"); p.Fate != domain.FateReuse {
		t.Errorf("AlignSample.Align: got %v (%v), want reuse", p.Fate, p.Reasons)
	}
	if p := predictionFor(t, got, "AlignSample.Sort"); p.Fate != domain.FateRerun {
		t.Errorf("AlignSample.Sort: got %v, want rerun", p.Fate)
	}
	// Report reads AlignSample.sorted, which Sort produces, so it does cascade.
	if p := predictionFor(t, got, "Report"); p.Fate != domain.FateRerunDownstream {
		t.Errorf("Report: got %v, want downstream", p.Fate)
	}
}

// Without the subworkflow source the call stays opaque and is reported as
// undetermined, with a warning telling the user what to bundle.
func TestForecastWarnsWhenSubworkflowSourceMissing(t *testing.T) {
	dir := t.TempDir()
	wdlPath := filepath.Join(dir, "cohort.wdl")
	if err := os.WriteFile(wdlPath, []byte(parentWDL), 0o600); err != nil {
		t.Fatalf("writing parent: %v", err)
	}
	inputsJSON, _ := json.Marshal(subInputs())
	inputPath := filepath.Join(dir, "inputs.json")
	if err := os.WriteFile(inputPath, inputsJSON, 0o600); err != nil {
		t.Fatalf("writing inputs: %v", err)
	}
	fp := &mockFileProvider{
		readBytesFunc: func(_ context.Context, path string) ([]byte, error) { return os.ReadFile(path) },
		getDigestsFunc: func(_ context.Context, _ string) (ports.FileDigests, error) {
			return ports.FileDigests{MD5: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}, nil
		},
	}
	env := forecastEnv{
		uc:        NewCacheForecastUseCase(&stubMetadataReader{workflow: subReference("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")}, nil, nil, fp),
		wdlPath:   wdlPath,
		inputPath: inputPath,
	}

	got := run(t, env)

	p := predictionFor(t, got, "AlignSample")
	if p.Fate != domain.FateUnknown {
		t.Errorf("AlignSample: got %v, want undetermined without its source", p.Fate)
	}
	joined := strings.Join(got.Warnings, " | ")
	if !strings.Contains(joined, "AlignSample") || !strings.Contains(joined, "--dependencies") {
		t.Errorf("warnings = %v, want AlignSample named and --dependencies suggested", got.Warnings)
	}
	// And the consumer downstream of it must not be claimed as reuse.
	if p := predictionFor(t, got, "Report"); p.Fate != domain.FateUnknown {
		t.Errorf("Report: got %v, want undetermined downstream of an unreadable subworkflow", p.Fate)
	}
}

// --- GCS hashing and derived inputs ---------------------------------------

// Cromwell records a crc32c for GCS inputs, not an MD5. Comparing only MD5s
// made every GCP call undetermined, which is the whole feature failing on the
// backend most users are on.
func TestForecastComparesGCSFilesByCRC32C(t *testing.T) {
	const gcsCRC = "tBGf4Q==" // as captured from a real GoogleBatch run
	ref := forecastFixture(forecastWDL, "PAPIv2")
	for key := range ref.Calls {
		calls := ref.Calls[key]
		if _, ok := calls[0].Fingerprint["input: File input_vcf"]; ok {
			calls[0].Fingerprint["input: File input_vcf"] = gcsCRC
		}
		ref.Calls[key] = calls
	}

	inputs := map[string]any{
		"VcfIndexAndStats.input_vcf":       "gs://bucket/in.vcf.gz",
		"VcfIndexAndStats.output_basename": "sample",
	}
	env := newForecastEnvWithDigests(t, forecastWDL, inputs, ref,
		map[string]ports.FileDigests{"gs://bucket/in.vcf.gz": {CRC32C: gcsCRC}})

	got := run(t, env)

	if p := predictionFor(t, got, "IndexVcf"); p.Fate != domain.FateReuse {
		t.Errorf("IndexVcf: got %v (%v), want reuse via crc32c", p.Fate, p.Reasons)
	}
}

func TestForecastDetectsChangedGCSFileByCRC32C(t *testing.T) {
	ref := forecastFixture(forecastWDL, "PAPIv2")
	for key := range ref.Calls {
		calls := ref.Calls[key]
		if _, ok := calls[0].Fingerprint["input: File input_vcf"]; ok {
			calls[0].Fingerprint["input: File input_vcf"] = "tBGf4Q=="
		}
		ref.Calls[key] = calls
	}

	inputs := map[string]any{
		"VcfIndexAndStats.input_vcf":       "gs://bucket/in.vcf.gz",
		"VcfIndexAndStats.output_basename": "sample",
	}
	env := newForecastEnvWithDigests(t, forecastWDL, inputs, ref,
		map[string]ports.FileDigests{"gs://bucket/in.vcf.gz": {CRC32C: "ZZZZZZ=="}})

	got := run(t, env)

	if p := predictionFor(t, got, "IndexVcf"); p.Fate != domain.FateRerun {
		t.Errorf("IndexVcf: got %v, want rerun for a different crc32c", p.Fate)
	}
}

// A value the WDL computes — a disk size derived from an input's size — must
// not make the whole call undetermined. Real pipelines have one on every task,
// and poisoning on it reported an entire production workflow as unknowable.
func TestForecastDoesNotPoisonCallOnDerivedInput(t *testing.T) {
	ref := forecastFixture(forecastWDL, "Local")
	for key := range ref.Calls {
		calls := ref.Calls[key]
		calls[0].Fingerprint["input: Int disk_size"] = "DERIVED"
		calls[0].Inputs["disk_size"] = 250
		ref.Calls[key] = calls
	}

	inputs := map[string]any{
		"VcfIndexAndStats.input_vcf":       "/data/in.vcf.gz",
		"VcfIndexAndStats.output_basename": "sample",
	}
	env := newForecastEnv(t, forecastWDL, inputs, ref,
		map[string]string{"/data/in.vcf.gz": refVcfHash})

	got := run(t, env)

	if p := predictionFor(t, got, "IndexVcf"); p.Fate != domain.FateReuse {
		t.Errorf("IndexVcf: got %v (%v), want reuse — a computed input must not poison",
			p.Fate, p.Reasons)
	}
	joined := strings.Join(got.Warnings, " | ")
	if !strings.Contains(joined, "disk_size") {
		t.Errorf("the assumption should be reported, warnings = %v", got.Warnings)
	}
}

// Inputs the reference did not fingerprint play no part in the cache key, so
// comparing them would invent differences Cromwell never sees.
func TestForecastIgnoresInputsAbsentFromTheFingerprint(t *testing.T) {
	ref := forecastFixture(forecastWDL, "Local")
	for key := range ref.Calls {
		calls := ref.Calls[key]
		// Present as an input, absent from the fingerprint.
		calls[0].Inputs["not_hashed"] = "whatever"
		ref.Calls[key] = calls
	}

	inputs := map[string]any{
		"VcfIndexAndStats.input_vcf":       "/data/in.vcf.gz",
		"VcfIndexAndStats.output_basename": "sample",
	}
	env := newForecastEnv(t, forecastWDL, inputs, ref,
		map[string]string{"/data/in.vcf.gz": refVcfHash})

	got := run(t, env)

	if p := predictionFor(t, got, "IndexVcf"); p.Fate != domain.FateReuse {
		t.Errorf("IndexVcf: got %v (%v), want reuse", p.Fate, p.Reasons)
	}
}
