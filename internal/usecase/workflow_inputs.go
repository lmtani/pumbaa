package usecase

import (
	"github.com/lmtani/pumbaa/internal/interfaces"
)

// WorkflowInputsInputDTO - Input
type WorkflowInputsInputDTO struct {
	WorkflowID string
}

// WorkflowInputsOutputDTO - Output
type WorkflowInputsOutputDTO struct {
	WorkflowID string
	Inputs     map[string]interface{}
}

// WorkflowInputsUseCase is a usecase to get inputs from Cromwell
type WorkflowInputsUseCase struct {
	CromwellClient interfaces.CromwellServer
}

// NewWorkflowInputs creates a new WorkflowInputs usecase
func NewWorkflowInputs(c interfaces.CromwellServer) *WorkflowInputsUseCase {
	return &WorkflowInputsUseCase{CromwellClient: c}
}

// Execute gets inputs from Cromwell
func (w *WorkflowInputsUseCase) Execute(i *WorkflowInputsInputDTO) (*WorkflowInputsOutputDTO, error) {
	result, err := w.CromwellClient.Metadata(i.WorkflowID, nil)
	if err != nil {
		return nil, err
	}
	return &WorkflowInputsOutputDTO{WorkflowID: i.WorkflowID, Inputs: result.Inputs}, nil
}
