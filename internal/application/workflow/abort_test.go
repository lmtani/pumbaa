package workflow

import (
	"context"
	"errors"
	"testing"

	"github.com/lmtani/pumbaa/internal/application"
	"github.com/lmtani/pumbaa/internal/domain/workflow"
)

// mockWorkflowRepository is defined in testutil_test.go

func TestAbortUseCase_Execute(t *testing.T) {
	repo := &mockWorkflowRepository{
		abortFunc: func(ctx context.Context, workflowID string) error {
			return nil
		},
	}
	uc := NewAbortUseCase(repo)

	input := AbortInput{WorkflowID: "test-id"}
	_, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAbortUseCase_Execute_TerminalState(t *testing.T) {
	// Create a specific mock for this test
	repoWithTerminalStatus := &mockWorkflowRepository{
		getStatusFunc: func(ctx context.Context, workflowID string) (workflow.Status, error) {
			return workflow.StatusSucceeded, nil
		},
	}

	uc := NewAbortUseCase(repoWithTerminalStatus)

	input := AbortInput{WorkflowID: "test-id"}
	_, err := uc.Execute(context.Background(), input)
	if !errors.Is(err, workflow.ErrWorkflowAlreadyTerminal) {
		t.Errorf("expected ErrWorkflowAlreadyTerminal, got %v", err)
	}
}

func TestAbortUseCase_Execute_Validation(t *testing.T) {
	repo := &mockWorkflowRepository{}
	uc := NewAbortUseCase(repo)

	_, err := uc.Execute(context.Background(), AbortInput{})
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

func TestAbortUseCase_Execute_StatusError(t *testing.T) {
	repo := &mockWorkflowRepository{
		getStatusFunc: func(ctx context.Context, workflowID string) (workflow.Status, error) {
			return workflow.StatusRunning, errors.New("status failed")
		},
	}
	uc := NewAbortUseCase(repo)

	_, err := uc.Execute(context.Background(), AbortInput{WorkflowID: "test-id"})
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
	if ucErr.Operation != "abort" {
		t.Errorf("expected operation abort, got %s", ucErr.Operation)
	}
}

func TestAbortUseCase_Execute_AbortError(t *testing.T) {
	repo := &mockWorkflowRepository{
		abortFunc: func(ctx context.Context, workflowID string) error {
			return errors.New("abort failed")
		},
	}
	uc := NewAbortUseCase(repo)

	_, err := uc.Execute(context.Background(), AbortInput{WorkflowID: "test-id"})
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
	if ucErr.Operation != "abort" {
		t.Errorf("expected operation abort, got %s", ucErr.Operation)
	}
}
