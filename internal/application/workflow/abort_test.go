package workflow

import (
	"context"
	"testing"

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
	if err != workflow.ErrWorkflowAlreadyTerminal {
		t.Errorf("expected ErrWorkflowAlreadyTerminal, got %v", err)
	}
}
