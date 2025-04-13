package usecase

import (
	"github.com/lmtani/pumbaa/internal/entities"
	"github.com/lmtani/pumbaa/internal/interfaces"
)

// WorkflowOutputsInputDTO - Input
type WorkflowOutputsInputDTO struct {
	WorkflowID string
}

// WorkflowOutputsOutputDTO - Output
type WorkflowOutputsOutputDTO struct {
	WorkflowID string
	Outputs    entities.OutputsResponse
}

// WorkflowOutputsUseCase is a usecase to get outputs from Cromwell
type WorkflowOutputsUseCase struct {
	CromwellClient interfaces.CromwellServer
}

// NewWorkflowOutputs creates a new WorkflowOutputs usecase
func NewWorkflowOutputs(c interfaces.CromwellServer) *WorkflowOutputsUseCase {
	return &WorkflowOutputsUseCase{CromwellClient: c}
}

// Execute gets outputs from Cromwell
func (w *WorkflowOutputsUseCase) Execute(i *WorkflowOutputsInputDTO) (*WorkflowOutputsOutputDTO, error) {
	result, err := w.CromwellClient.Outputs(i.WorkflowID)
	if err != nil {
		return nil, err
	}
	return &WorkflowOutputsOutputDTO{WorkflowID: i.WorkflowID, Outputs: result}, nil
}
