package workflow

import (
	"context"
	"fmt"

	"github.com/lmtani/pumbaa/internal/domain/ports"
	workflow2 "github.com/lmtani/pumbaa/internal/domain/workflow"
)

// AbortUseCase handles workflow abortion.
type AbortUseCase struct {
	repo ports.WorkflowRepository
}

// NewAbortUseCase creates a new abort use case.
func NewAbortUseCase(repo ports.WorkflowRepository) *AbortUseCase {
	return &AbortUseCase{repo: repo}
}

// AbortInput represents the input for workflow abortion.
type AbortInput struct {
	WorkflowID string
}

// AbortOutput represents the output of workflow abortion.
type AbortOutput struct {
	WorkflowID string
	Message    string
}

// Execute aborts a running workflow.
func (uc *AbortUseCase) Execute(ctx context.Context, input AbortInput) (*AbortOutput, error) {
	if input.WorkflowID == "" {
		return nil, workflow2.ErrInvalidWorkflowID
	}

	// Check current status before aborting
	status, err := uc.repo.GetStatus(ctx, input.WorkflowID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow status: %w", err)
	}

	// Check if workflow is already in terminal state
	switch status {
	case workflow2.StatusSucceeded, workflow2.StatusFailed, workflow2.StatusAborted:
		return nil, workflow2.ErrWorkflowAlreadyTerminal
	}

	// Abort the workflow
	if err := uc.repo.Abort(ctx, input.WorkflowID); err != nil {
		return nil, fmt.Errorf("failed to abort workflow: %w", err)
	}

	return &AbortOutput{
		WorkflowID: input.WorkflowID,
		Message:    "Workflow abort requested successfully",
	}, nil
}
