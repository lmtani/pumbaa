package history

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lmtani/pumbaa/internal/application/ports"
	"github.com/lmtani/pumbaa/internal/application/workflow"
	"github.com/lmtani/pumbaa/internal/infrastructure/agents/tools/types"
)

type stubExporter struct {
	lastInput workflow.ExportRunInput
	out       *workflow.ExportRunOutput
	err       error
}

func (s *stubExporter) Execute(_ context.Context, input workflow.ExportRunInput) (*workflow.ExportRunOutput, error) {
	s.lastInput = input
	if s.err != nil {
		return nil, s.err
	}
	out := *s.out
	out.Dir = input.Dir
	return &out, nil
}

func exported() *workflow.ExportRunOutput {
	return &workflow.ExportRunOutput{
		WorkflowID: "run-1",
		Files: []workflow.ExportedFile{
			{Kind: ports.RunFileWorkflow, Name: "workflow.wdl", Source: workflow.ExportFromHistory, Size: 120},
			{Kind: ports.RunFileOptions, Name: "options.json", Source: workflow.ExportFromServer, Size: 40, Warning: "reformatted by the server"},
		},
		Secrets:       []string{"user_service_account_json"},
		SubmitCommand: "pumbaa workflow submit -w workflow.wdl -o options.json",
	}
}

func TestExportWritesUnderTheWorkingDirectory(t *testing.T) {
	cwd := t.TempDir()
	t.Chdir(cwd)
	exporter := &stubExporter{out: exported()}
	h := NewExportHandler(exporter)

	out, err := h.Handle(context.Background(), types.Input{Action: exportAction, WorkflowID: "run-1"})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	got := data(t, out)

	if want := filepath.Join(cwd, "run-1"); exporter.lastInput.Dir != want {
		t.Errorf("dir = %q, want the workflow id under the working directory %q", exporter.lastInput.Dir, want)
	}
	if got["complete"] != true {
		t.Errorf("complete = %v, want true with nothing missing", got["complete"])
	}
	files := got["files"].([]map[string]any)
	if len(files) != 2 || files[1]["note"] != "reformatted by the server" {
		t.Errorf("files = %v, want both entries with the server's note carried over", files)
	}
	if submit, _ := got["submit"].(string); !strings.HasPrefix(submit, "cd "+filepath.Join(cwd, "run-1")+" && pumbaa workflow submit") {
		t.Errorf("submit = %q, want the command prefixed with the directory", submit)
	}
	if got["note"] == nil {
		t.Error("no note about the credentials in options.json")
	}
}

func TestExportHonoursPathAndOverwrite(t *testing.T) {
	t.Chdir(t.TempDir())
	exporter := &stubExporter{out: exported()}
	h := NewExportHandler(exporter)

	if _, err := h.Handle(context.Background(), types.Input{Action: exportAction, WorkflowID: "run-1", Path: "repro/run-1", Overwrite: true}); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !strings.HasSuffix(exporter.lastInput.Dir, filepath.Join("repro", "run-1")) || !exporter.lastInput.Overwrite {
		t.Errorf("input = %+v, want the relative path resolved and overwrite passed through", exporter.lastInput)
	}

	out, err := h.Handle(context.Background(), types.Input{Action: exportAction, WorkflowID: "run-1", Path: "/etc/run-1"})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if out.Success || !strings.Contains(out.Error, "relative") {
		t.Errorf("an absolute path was accepted: %+v", out)
	}
}

func TestExportExplainsRefusals(t *testing.T) {
	t.Chdir(t.TempDir())

	out, err := NewExportHandler(&stubExporter{}).Handle(context.Background(), types.Input{Action: exportAction})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if out.Success || !strings.Contains(out.Error, "workflow_id") {
		t.Errorf("a missing workflow_id was not reported: %+v", out)
	}

	exporter := &stubExporter{err: errors.New("export would overwrite existing files in x: workflow.wdl")}
	exporter.err = workflow.ErrExportWouldOverwrite
	out, err = NewExportHandler(exporter).Handle(context.Background(), types.Input{Action: exportAction, WorkflowID: "run-1"})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if out.Success || !strings.Contains(out.Error, "overwrite=true") {
		t.Errorf("an overwrite refusal does not say how to proceed: %+v", out)
	}
}
