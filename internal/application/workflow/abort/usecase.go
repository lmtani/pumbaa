// Package abort contains the use case for aborting workflows.
package abort

import (
	"context"
	"fmt"

	"github.com/lmtani/pumbaa/internal/domain/ports"
	"github.com/lmtani/pumbaa/internal/domain/workflow"
)

// UseCase handles workflow abortion.
type UseCase struct {
	repo ports.WorkflowRepository
}

// New creates a new abort use case.
func New(repo ports.WorkflowRepository) *UseCase {
	return &UseCase{repo: repo}
}

// Input represents the input for the abort use case.
type Input struct {
	WorkflowID string
}

// Output represents the output of the abort use case.
type Output struct {
	WorkflowID string
	Message    string
}

// Execute aborts a running workflow.
func (uc *UseCase) Execute(ctx context.Context, input Input) (*Output, error) {
	if input.WorkflowID == "" {
		return nil, workflow.ErrInvalidWorkflowID
	}

	// Check current status before aborting
	status, err := uc.repo.GetStatus(ctx, input.WorkflowID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow status: %w", err)
	}

	// Check if workflow is already in terminal state
	switch status {
	case workflow.StatusSucceeded, workflow.StatusFailed, workflow.StatusAborted:
		return nil, workflow.ErrWorkflowAlreadyTerminal
	}

	// Abort the workflow
	if err := uc.repo.Abort(ctx, input.WorkflowID); err != nil {
		return nil, fmt.Errorf("failed to abort workflow: %w", err)
	}

	return &Output{
		WorkflowID: input.WorkflowID,
		Message:    "Workflow abort requested successfully",
	}, nil
}
