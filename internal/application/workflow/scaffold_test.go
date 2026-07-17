package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func TestScaffoldInputsUseCase(t *testing.T) {
	fp := &mockFileProvider{
		readBytesFunc: func(ctx context.Context, path string) ([]byte, error) {
			if path != "align.wdl" {
				return nil, errors.New("unexpected path: " + path)
			}
			return []byte(preflightWDL), nil
		},
	}
	uc := NewScaffoldInputsUseCase(fp)

	out, err := uc.Execute(context.Background(), ScaffoldInputsInput{WorkflowFile: "align.wdl"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if out.WorkflowName != "Align" {
		t.Errorf("WorkflowName = %q, want Align", out.WorkflowName)
	}
	// Every declared input is described, even the ones kept out of the file.
	if len(out.Inputs) != 3 {
		t.Errorf("got %d specs, want 3: %+v", len(out.Inputs), out.Inputs)
	}

	var template map[string]any
	if err := json.Unmarshal(out.Template, &template); err != nil {
		t.Fatalf("template is not valid JSON: %v", err)
	}
	// Only the two required inputs; "threads" has a default.
	if len(template) != 2 {
		t.Errorf("template = %v, want only the required inputs", template)
	}
}

func TestScaffoldInputsUseCaseIncludeOptional(t *testing.T) {
	fp := &mockFileProvider{
		readBytesFunc: func(ctx context.Context, path string) ([]byte, error) {
			return []byte(preflightWDL), nil
		},
	}
	uc := NewScaffoldInputsUseCase(fp)

	out, err := uc.Execute(context.Background(), ScaffoldInputsInput{WorkflowFile: "align.wdl", IncludeOptional: true})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var template map[string]any
	if err := json.Unmarshal(out.Template, &template); err != nil {
		t.Fatalf("template is not valid JSON: %v", err)
	}
	if len(template) != 3 {
		t.Errorf("template = %v, want every input", template)
	}
	if template["Align.threads"] != float64(4) {
		t.Errorf("optional input should carry its default, got %v", template["Align.threads"])
	}
}

func TestScaffoldInputsUseCaseErrors(t *testing.T) {
	t.Run("workflow file is required", func(t *testing.T) {
		uc := NewScaffoldInputsUseCase(&mockFileProvider{})
		if _, err := uc.Execute(context.Background(), ScaffoldInputsInput{}); err == nil {
			t.Error("expected an error without a workflow file")
		}
	})

	t.Run("unreadable file", func(t *testing.T) {
		fp := &mockFileProvider{
			readBytesFunc: func(ctx context.Context, path string) ([]byte, error) {
				return nil, errors.New("no such file")
			},
		}
		uc := NewScaffoldInputsUseCase(fp)
		if _, err := uc.Execute(context.Background(), ScaffoldInputsInput{WorkflowFile: "nope.wdl"}); err == nil {
			t.Error("expected an error for an unreadable workflow file")
		}
	})

	t.Run("WDL without a workflow", func(t *testing.T) {
		fp := &mockFileProvider{
			readBytesFunc: func(ctx context.Context, path string) ([]byte, error) {
				return []byte("version 1.0\n\ntask alone {\n  command <<< echo hi >>>\n}\n"), nil
			},
		}
		uc := NewScaffoldInputsUseCase(fp)
		if _, err := uc.Execute(context.Background(), ScaffoldInputsInput{WorkflowFile: "tasks.wdl"}); err == nil {
			t.Error("scaffolding a task-only WDL should explain there is nothing to scaffold")
		}
	})
}
