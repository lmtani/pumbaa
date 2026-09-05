package workflow

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/lmtani/pumbaa/internal/application/ports"
	"github.com/lmtani/pumbaa/internal/domain/workflow"
)

// fakeRunHistory records what the submit path asks it to remember. Only the
// write side is exercised here; the reads have their own tests in the store.
type fakeRunHistory struct {
	ports.RunHistory
	recorded  []ports.RunRecord
	recordErr error
}

func (f *fakeRunHistory) Record(_ context.Context, rec ports.RunRecord) error {
	if f.recordErr != nil {
		return f.recordErr
	}
	f.recorded = append(f.recorded, rec)
	return nil
}

type fakeHostProvider struct {
	url   string
	alias string
}

func (f fakeHostProvider) CurrentHost() (string, string) { return f.url, f.alias }

func submitFixture(t *testing.T) (*mockWorkflowRepository, *mockFileProvider) {
	t.Helper()
	repo := &mockWorkflowRepository{
		submitFunc: func(_ context.Context, _ workflow.SubmitRequest) (*workflow.SubmitResponse, error) {
			return &workflow.SubmitResponse{ID: "run-1", Status: workflow.StatusSubmitted}, nil
		},
	}
	fp := &mockFileProvider{
		readBytesFunc: func(_ context.Context, path string) ([]byte, error) {
			switch path {
			case "hello.wdl":
				return []byte("version 1.0\nworkflow HelloWorld {\n}\n"), nil
			case "hello.inputs.json":
				return []byte(`{}`), nil
			default:
				return nil, errors.New("unexpected path: " + path)
			}
		},
	}
	return repo, fp
}

func TestSubmitRemembersTheRun(t *testing.T) {
	repo, fp := submitFixture(t)
	store := &fakeRunHistory{}
	uc := NewSubmitUseCase(repo, fp, NewPreflightUseCase(fp, nil),
		&SubmitHistory{Store: store, Host: fakeHostProvider{url: "http://localhost:8000", alias: "local"}})

	out, err := uc.Execute(context.Background(), SubmitInput{
		WorkflowFile: "hello.wdl",
		InputsFile:   "hello.inputs.json",
		Description:  "smoke test after the docker bump",
		Labels:       map[string]string{"owner": "lucas"},
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !out.Remembered || out.HistoryError != nil {
		t.Fatalf("Remembered=%v, HistoryError=%v; want the run recorded", out.Remembered, out.HistoryError)
	}
	if len(store.recorded) != 1 {
		t.Fatalf("recorded %d runs, want 1", len(store.recorded))
	}

	rec := store.recorded[0]
	if rec.WorkflowID != "run-1" || rec.Host != "http://localhost:8000" || rec.HostAlias != "local" {
		t.Errorf("got %+v, want the run keyed by the active host", rec)
	}
	if rec.Description != "smoke test after the docker bump" {
		t.Errorf("description = %q, want the one given at submission", rec.Description)
	}
	// Paths are absolutised: a record saying "hello.wdl" is worthless read
	// from another directory later.
	wantWDL, _ := filepath.Abs("hello.wdl")
	wantInputs, _ := filepath.Abs("hello.inputs.json")
	if rec.WorkflowFile != wantWDL || rec.InputsFile != wantInputs {
		t.Errorf("got %q/%q, want the absolute local paths %q/%q", rec.WorkflowFile, rec.InputsFile, wantWDL, wantInputs)
	}
	if rec.Origin != ports.OriginSubmit {
		t.Errorf("origin = %q, want %q", rec.Origin, ports.OriginSubmit)
	}
	if rec.Name != "HelloWorld" {
		t.Errorf("name = %q, want the name preflight already parsed", rec.Name)
	}
	if rec.LastStatus != string(workflow.StatusSubmitted) {
		t.Errorf("last status = %q, want the submission status", rec.LastStatus)
	}
}

func TestSubmitDoesNotSendTheDescriptionToCromwell(t *testing.T) {
	var sent workflow.SubmitRequest
	repo, fp := submitFixture(t)
	repo.submitFunc = func(_ context.Context, req workflow.SubmitRequest) (*workflow.SubmitResponse, error) {
		sent = req
		return &workflow.SubmitResponse{ID: "run-1", Status: workflow.StatusSubmitted}, nil
	}
	uc := NewSubmitUseCase(repo, fp, NewPreflightUseCase(fp, nil),
		&SubmitHistory{Store: &fakeRunHistory{}, Host: fakeHostProvider{url: "http://localhost:8000"}})

	if _, err := uc.Execute(context.Background(), SubmitInput{
		WorkflowFile: "hello.wdl",
		Description:  "free text that would not survive a Cromwell label",
	}); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	for key, value := range sent.Labels {
		if value == "free text that would not survive a Cromwell label" {
			t.Errorf("description leaked into label %q", key)
		}
	}
}

func TestSubmitSurvivesAFailingHistory(t *testing.T) {
	repo, fp := submitFixture(t)
	store := &fakeRunHistory{recordErr: errors.New("disk full")}
	uc := NewSubmitUseCase(repo, fp, NewPreflightUseCase(fp, nil),
		&SubmitHistory{Store: store, Host: fakeHostProvider{url: "http://localhost:8000"}})

	out, err := uc.Execute(context.Background(), SubmitInput{WorkflowFile: "hello.wdl", Description: "note"})
	if err != nil {
		t.Fatalf("Execute returned %v; the workflow is already running, so history must not fail it", err)
	}
	if out.WorkflowID != "run-1" {
		t.Errorf("workflow id = %q, want the submitted run", out.WorkflowID)
	}
	if out.Remembered {
		t.Error("Remembered = true, want false when the store failed")
	}
	if out.HistoryError == nil {
		t.Error("HistoryError = nil, want the failure reported so the caller can warn")
	}
}

func TestSubmitKeepsRemotePathsVerbatim(t *testing.T) {
	repo, fp := submitFixture(t)
	fp.readBytesFunc = func(_ context.Context, path string) ([]byte, error) {
		if path == "gs://bucket/hello.wdl" {
			return []byte("version 1.0\nworkflow HelloWorld {\n}\n"), nil
		}
		return nil, errors.New("unexpected path: " + path)
	}
	store := &fakeRunHistory{}
	uc := NewSubmitUseCase(repo, fp, NewPreflightUseCase(fp, nil),
		&SubmitHistory{Store: store, Host: fakeHostProvider{url: "http://localhost:8000"}})

	if _, err := uc.Execute(context.Background(), SubmitInput{WorkflowFile: "gs://bucket/hello.wdl"}); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if got := store.recorded[0].WorkflowFile; got != "gs://bucket/hello.wdl" {
		t.Errorf("workflow file = %q, want the remote URI untouched", got)
	}
}

func TestSubmitWithoutHistoryConfigured(t *testing.T) {
	repo, fp := submitFixture(t)
	uc := NewSubmitUseCase(repo, fp, NewPreflightUseCase(fp, nil), nil)

	out, err := uc.Execute(context.Background(), SubmitInput{WorkflowFile: "hello.wdl"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.Remembered || out.HistoryError != nil {
		t.Errorf("Remembered=%v, HistoryError=%v; want a silent no-op", out.Remembered, out.HistoryError)
	}
}
