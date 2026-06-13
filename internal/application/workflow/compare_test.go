package workflow

import (
	"context"
	"errors"
	"testing"

	"github.com/lmtani/pumbaa/internal/domain/workflow"
)

func TestCompareUseCase_Execute(t *testing.T) {
	repo := &mockWorkflowRepository{
		getMetadataFunc: func(ctx context.Context, id string) (*workflow.Workflow, error) {
			switch id {
			case "run-a":
				return &workflow.Workflow{Name: "wf", SubmittedInputs: `{"wf.x":1}`}, nil
			case "run-b":
				return &workflow.Workflow{Name: "wf", SubmittedInputs: `{"wf.x":2}`}, nil
			}
			return nil, errors.New("unknown id")
		},
	}
	uc := NewCompareUseCase(repo)

	diff, err := uc.Execute(context.Background(), CompareInput{WorkflowIDA: "run-a", WorkflowIDB: "run-b"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if diff.IDA != "" { // IDs come from metadata, which our fixtures leave blank
		t.Logf("note: IDA populated from metadata = %q", diff.IDA)
	}
	if len(diff.Inputs) != 1 || diff.Inputs[0].Key != "wf.x" {
		t.Fatalf("expected one input diff on wf.x, got %+v", diff.Inputs)
	}
	if diff.Inputs[0].ValueA != "1" || diff.Inputs[0].ValueB != "2" {
		t.Errorf("wf.x diff = %s→%s, want 1→2", diff.Inputs[0].ValueA, diff.Inputs[0].ValueB)
	}
}

func TestCompareUseCase_Execute_Validation(t *testing.T) {
	uc := NewCompareUseCase(&mockWorkflowRepository{})

	if _, err := uc.Execute(context.Background(), CompareInput{WorkflowIDB: "b"}); err == nil {
		t.Error("expected error when first ID is empty")
	}
	if _, err := uc.Execute(context.Background(), CompareInput{WorkflowIDA: "a"}); err == nil {
		t.Error("expected error when second ID is empty")
	}
}

func TestCompareUseCase_Execute_FetchError(t *testing.T) {
	repo := &mockWorkflowRepository{
		getMetadataFunc: func(ctx context.Context, id string) (*workflow.Workflow, error) {
			if id == "bad" {
				return nil, errors.New("not found")
			}
			return &workflow.Workflow{}, nil
		},
	}
	uc := NewCompareUseCase(repo)

	if _, err := uc.Execute(context.Background(), CompareInput{WorkflowIDA: "bad", WorkflowIDB: "ok"}); err == nil {
		t.Error("expected error when first metadata fetch fails")
	}
	if _, err := uc.Execute(context.Background(), CompareInput{WorkflowIDA: "ok", WorkflowIDB: "bad"}); err == nil {
		t.Error("expected error when second metadata fetch fails")
	}
}
