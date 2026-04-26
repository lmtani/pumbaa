package workflow

import (
	"context"
	"encoding/json"

	"github.com/lmtani/pumbaa/internal/application"
	"github.com/lmtani/pumbaa/internal/application/ports"
)

// InputsUseCase handles workflow inputs retrieval.
type InputsUseCase struct {
	reader ports.WorkflowMetadataReader
}

// NewInputsUseCase creates a new inputs use case.
func NewInputsUseCase(reader ports.WorkflowMetadataReader) *InputsUseCase {
	return &InputsUseCase{reader: reader}
}

// InputsInput represents the input for inputs retrieval.
type InputsInput struct {
	WorkflowID string
}

// InputsOutput represents the output for inputs retrieval.
type InputsOutput struct {
	WorkflowID   string
	WorkflowName string
	Inputs       map[string]any
}

// Execute retrieves inputs for a workflow.
func (uc *InputsUseCase) Execute(ctx context.Context, input InputsInput) (*InputsOutput, error) {
	if input.WorkflowID == "" {
		return nil, application.NewInputValidationError("workflowID", "is required")
	}

	wf, err := uc.reader.GetMetadata(ctx, input.WorkflowID)
	if err != nil {
		return nil, application.NewUseCaseError("inputs", "failed to get workflow inputs", err)
	}

	var inputs map[string]any
	if err := json.Unmarshal([]byte(wf.SubmittedInputs), &inputs); err != nil {
		return nil, application.NewUseCaseError("inputs", "failed to parse submitted inputs", err)
	}

	return &InputsOutput{
		WorkflowID:   wf.ID,
		WorkflowName: wf.Name,
		Inputs:       inputs,
	}, nil
}
