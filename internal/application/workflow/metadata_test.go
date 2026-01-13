package workflow

import (
	"context"
	"errors"
	"testing"

	"github.com/lmtani/pumbaa/internal/application"
	"github.com/lmtani/pumbaa/internal/domain/workflow"
)

func TestMetadataUseCase_Execute_WithFailures(t *testing.T) {
	repo := &mockWorkflowRepository{
		getMetadataFunc: func(ctx context.Context, workflowID string) (*workflow.Workflow, error) {
			return &workflow.Workflow{
				ID:   "test-id",
				Name: "test-wf",
				Failures: []workflow.Failure{
					{Message: "root failure", CausedBy: []workflow.Failure{{Message: "cause"}}},
				},
				Calls: map[string][]workflow.Call{
					"task1": {{Name: "task1", Status: workflow.StatusFailed, Failures: []workflow.Failure{{Message: "task failure"}}}},
				},
			}, nil
		},
	}
	uc := NewMetadataUseCase(repo)

	input := MetadataInput{WorkflowID: "test-id"}
	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(output.Failures) == 0 {
		t.Error("expected failures in output")
	}
}

func TestMetadataUseCase_Execute_Validation(t *testing.T) {
	repo := &mockWorkflowRepository{}
	uc := NewMetadataUseCase(repo)

	_, err := uc.Execute(context.Background(), MetadataInput{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, application.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
	var inputErr *application.InputValidationError
	if !errors.As(err, &inputErr) {
		t.Fatalf("expected InputValidationError, got %T", err)
	}
	if inputErr.Field != "workflowID" {
		t.Errorf("expected field workflowID, got %s", inputErr.Field)
	}
}

func TestMetadataUseCase_Execute_Error(t *testing.T) {
	repo := &mockWorkflowRepository{
		getMetadataFunc: func(ctx context.Context, workflowID string) (*workflow.Workflow, error) {
			return nil, errors.New("metadata failed")
		},
	}
	uc := NewMetadataUseCase(repo)

	_, err := uc.Execute(context.Background(), MetadataInput{WorkflowID: "test-id"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, application.ErrOperationFailed) {
		t.Errorf("expected ErrOperationFailed, got %v", err)
	}
	var ucErr *application.UseCaseError
	if !errors.As(err, &ucErr) {
		t.Fatalf("expected UseCaseError, got %T", err)
	}
	if ucErr.Operation != "metadata" {
		t.Errorf("expected operation metadata, got %s", ucErr.Operation)
	}
}
