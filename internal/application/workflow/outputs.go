package workflow

import (
	"context"

	"github.com/lmtani/pumbaa/internal/application"
	"github.com/lmtani/pumbaa/internal/application/ports"
)

// OutputsUseCase handles workflow outputs retrieval.
type OutputsUseCase struct {
	reader ports.WorkflowMetadataReader
}

// NewOutputsUseCase creates a new outputs use case.
func NewOutputsUseCase(reader ports.WorkflowMetadataReader) *OutputsUseCase {
	return &OutputsUseCase{reader: reader}
}

// OutputsInput represents the input for outputs retrieval.
type OutputsInput struct {
	WorkflowID string
}

// OutputsOutput represents the output for outputs retrieval.
type OutputsOutput struct {
	WorkflowID   string
	WorkflowName string
	Outputs      map[string]interface{}
}

// Execute retrieves outputs for a workflow.
func (uc *OutputsUseCase) Execute(ctx context.Context, input OutputsInput) (*OutputsOutput, error) {
	if input.WorkflowID == "" {
		return nil, application.NewInputValidationError("workflowID", "is required")
	}

	wf, err := uc.reader.GetMetadata(ctx, input.WorkflowID)
	if err != nil {
		return nil, application.NewUseCaseError("outputs", "failed to get workflow outputs", err)
	}

	return &OutputsOutput{
		WorkflowID:   wf.ID,
		WorkflowName: wf.Name,
		Outputs:      wf.Outputs,
	}, nil
}
