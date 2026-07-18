package submit

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lmtani/pumbaa/internal/application/ports"
	"github.com/lmtani/pumbaa/internal/infrastructure/agents/tools/types"
)

const sampleWDL = `version 1.0

workflow AlignReads {
    input {
        File reads
        String sample
        Int threads = 4
    }

    parameter_meta {
        reads: "FASTQ reads"
    }
}
`

// chdir runs the test inside a fresh working directory, since the actions
// read files relative to it.
func chdir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
	return dir
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// fakeProvider answers GetSize from a set of known paths; anything else is
// reported as not found, and paths in errPaths as an unverifiable error.
type fakeProvider struct {
	exist    map[string]bool
	errPaths map[string]bool
}

func (f *fakeProvider) Read(ctx context.Context, path string) (string, error)      { return "", nil }
func (f *fakeProvider) ReadBytes(ctx context.Context, path string) ([]byte, error) { return nil, nil }
func (f *fakeProvider) GetSize(ctx context.Context, path string) (int64, error) {
	if f.errPaths[path] {
		return 0, errors.New("no credentials")
	}
	if f.exist[path] {
		return 10, nil
	}
	return 0, fmt.Errorf("%w: %s", ports.ErrFileNotFound, path)
}

func (f *fakeProvider) GetContentDigests(_ context.Context, path string) (ports.FileDigests, error) {
	if f.exist[path] {
		return ports.FileDigests{MD5: "d41d8cd98f00b204e9800998ecf8427e"}, nil
	}
	return ports.FileDigests{}, fmt.Errorf("%w: %s", ports.ErrFileNotFound, path)
}

func TestScaffoldHandler(t *testing.T) {
	dir := chdir(t)
	writeFile(t, dir, "main.wdl", sampleWDL)
	h := NewScaffoldHandler()

	out, err := h.Handle(context.Background(), types.Input{Action: "scaffold", WorkflowFile: "main.wdl"})
	if err != nil || !out.Success {
		t.Fatalf("Handle failed: err=%v out=%+v", err, out)
	}

	data := out.Data.(map[string]any)
	if data["workflow"] != "AlignReads" {
		t.Errorf("workflow = %v, want AlignReads", data["workflow"])
	}
	inputs := data["inputs"].([]map[string]any)
	if len(inputs) != 3 {
		t.Fatalf("got %d inputs, want 3", len(inputs))
	}
	// Required inputs are marked, documentation is surfaced.
	if inputs[0]["name"] != "AlignReads.reads" || inputs[0]["required"] != true {
		t.Errorf("first input = %+v, want required reads", inputs[0])
	}
	if inputs[0]["description"] != "FASTQ reads" {
		t.Errorf("parameter_meta not surfaced: %+v", inputs[0])
	}
	// The template holds only the required inputs by default.
	tpl := data["template"].(string)
	if tpl == "" || !strings.Contains(tpl, "AlignReads.reads") || strings.Contains(tpl, "AlignReads.threads") {
		t.Errorf("template should hold required inputs only:\n%s", tpl)
	}
}

func TestScaffoldHandlerErrors(t *testing.T) {
	chdir(t)
	h := NewScaffoldHandler()

	t.Run("workflow_file required", func(t *testing.T) {
		out, _ := h.Handle(context.Background(), types.Input{Action: "scaffold"})
		if out.Success {
			t.Error("missing workflow_file should fail")
		}
	})

	t.Run("path escaping the working directory is refused", func(t *testing.T) {
		out, _ := h.Handle(context.Background(), types.Input{Action: "scaffold", WorkflowFile: "../secret.wdl"})
		if out.Success {
			t.Error("a path outside the working directory must be refused")
		}
	})

	t.Run("absolute path is refused", func(t *testing.T) {
		out, _ := h.Handle(context.Background(), types.Input{Action: "scaffold", WorkflowFile: "/etc/passwd"})
		if out.Success {
			t.Error("an absolute path must be refused")
		}
	})
}

func TestPreflightHandlerReady(t *testing.T) {
	dir := chdir(t)
	writeFile(t, dir, "main.wdl", sampleWDL)
	writeFile(t, dir, "inputs.json", `{"AlignReads.reads": "gs://b/r.fastq", "AlignReads.sample": "NA12878"}`)
	fp := &fakeProvider{exist: map[string]bool{"gs://b/r.fastq": true}}
	h := NewPreflightHandler(fp)

	out, err := h.Handle(context.Background(), types.Input{Action: "preflight", WorkflowFile: "main.wdl", InputsFile: "inputs.json"})
	if err != nil || !out.Success {
		t.Fatalf("Handle failed: err=%v out=%+v", err, out)
	}

	data := out.Data.(map[string]any)
	if data["ready"] != true {
		t.Errorf("ready = %v, want true: %+v", data["ready"], data)
	}
	if data["files_checked"] != 1 {
		t.Errorf("files_checked = %v, want 1", data["files_checked"])
	}
}

func TestPreflightHandlerReportsProblems(t *testing.T) {
	dir := chdir(t)
	writeFile(t, dir, "main.wdl", sampleWDL)
	// Missing required "sample", and a reads path that does not exist.
	writeFile(t, dir, "inputs.json", `{"AlignReads.reads": "gs://b/missing.fastq"}`)
	fp := &fakeProvider{exist: map[string]bool{}}
	h := NewPreflightHandler(fp)

	out, err := h.Handle(context.Background(), types.Input{Action: "preflight", WorkflowFile: "main.wdl", InputsFile: "inputs.json"})
	if err != nil || !out.Success {
		t.Fatalf("Handle failed: err=%v out=%+v", err, out)
	}

	data := out.Data.(map[string]any)
	if data["ready"] != false {
		t.Errorf("ready = %v, want false", data["ready"])
	}
	missing := data["missing_files"].([]string)
	if len(missing) != 1 {
		t.Errorf("missing_files = %v, want the one bad path", missing)
	}
	findings := data["input_findings"].([]map[string]any)
	foundMissing := false
	for _, f := range findings {
		if f["input"] == "AlignReads.sample" && f["severity"] == "error" {
			foundMissing = true
		}
	}
	if !foundMissing {
		t.Errorf("missing required input should be a finding: %+v", findings)
	}
}

func TestPreflightHandlerUnverifiablePathNotFatal(t *testing.T) {
	dir := chdir(t)
	writeFile(t, dir, "main.wdl", sampleWDL)
	writeFile(t, dir, "inputs.json", `{"AlignReads.reads": "gs://b/r.fastq", "AlignReads.sample": "NA12878"}`)
	fp := &fakeProvider{errPaths: map[string]bool{"gs://b/r.fastq": true}}
	h := NewPreflightHandler(fp)

	out, err := h.Handle(context.Background(), types.Input{Action: "preflight", WorkflowFile: "main.wdl", InputsFile: "inputs.json"})
	if err != nil || !out.Success {
		t.Fatalf("Handle failed: err=%v out=%+v", err, out)
	}

	data := out.Data.(map[string]any)
	// A path we could not check does not block readiness.
	if data["ready"] != true {
		t.Errorf("an unverifiable path must not block: %+v", data)
	}
	if len(data["unverified_file"].([]string)) != 1 {
		t.Errorf("the unverifiable path should be reported: %+v", data["unverified_file"])
	}
}

func TestPreflightHandlerWorkflowFileRequired(t *testing.T) {
	chdir(t)
	h := NewPreflightHandler(&fakeProvider{})
	out, _ := h.Handle(context.Background(), types.Input{Action: "preflight"})
	if out.Success {
		t.Error("missing workflow_file should fail")
	}
}
