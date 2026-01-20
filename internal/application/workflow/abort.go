package workflow

import (
	"context"

	"github.com/lmtani/pumbaa/internal/application"
	"github.com/lmtani/pumbaa/internal/application/ports"
	workflow2 "github.com/lmtani/pumbaa/internal/domain/workflow"
)

// AbortUseCase handles workflow abortion.
type AbortUseCase struct {
	aborter ports.WorkflowAborter
}

// NewAbortUseCase creates a new abort use case.
func NewAbortUseCase(aborter ports.WorkflowAborter) *AbortUseCase {
	return &AbortUseCase{aborter: aborter}
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
		return nil, application.NewInputValidationError("workflowID", "is required")
	}

	// Check current status before aborting
	status, err := uc.aborter.GetStatus(ctx, input.WorkflowID)
	if err != nil {
		return nil, application.NewUseCaseError("abort", "failed to get workflow status", err)
	}

	// Check if workflow is already in terminal state
	switch status {
	case workflow2.StatusSucceeded, workflow2.StatusFailed, workflow2.StatusAborted:
		return nil, application.NewUseCaseError("abort", "workflow is already in terminal state", workflow2.ErrWorkflowAlreadyTerminal)
	}

	// Abort the workflow
	if err := uc.aborter.Abort(ctx, input.WorkflowID); err != nil {
		return nil, application.NewUseCaseError("abort", "failed to abort workflow", err)
	}

	return &AbortOutput{
		WorkflowID: input.WorkflowID,
		Message:    "Workflow abort requested successfully",
	}, nil
}
