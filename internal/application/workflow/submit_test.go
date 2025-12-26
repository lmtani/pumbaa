package workflow

import (
	"context"
	"os"
	"testing"

	"github.com/lmtani/pumbaa/internal/domain/workflow"
)

func TestSubmitUseCase_Execute(t *testing.T) {
	repo := &mockWorkflowRepository{
		submitFunc: func(ctx context.Context, req workflow.SubmitRequest) (*workflow.SubmitResponse, error) {
			return &workflow.SubmitResponse{ID: "test-id", Status: workflow.StatusSubmitted}, nil
		},
	}
	uc := NewSubmitUseCase(repo)

	// Create a temporary file for the workflow
	tmpFile, err := os.CreateTemp("", "workflow-*.wdl")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.WriteString("workflow test {}"); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	input := SubmitInput{
		WorkflowFile: tmpFile.Name(),
	}
	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.WorkflowID != "test-id" {
		t.Errorf("expected WorkflowID test-id, got %s", output.WorkflowID)
	}
}

func TestSubmitUseCase_Execute_Error(t *testing.T) {
	repo := &mockWorkflowRepository{}
	uc := NewSubmitUseCase(repo)

	input := SubmitInput{
		WorkflowFile: "non-existent.wdl",
	}
	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
