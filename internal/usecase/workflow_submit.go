package usecase

import (
	"github.com/lmtani/pumbaa/internal/entities"
)

// WorkflowSubmitInputDTO - Input
type WorkflowSubmitInputDTO struct {
	Wdl          string
	Inputs       string
	Dependencies string
	Options      string
}

// WorkflowSubmitOutputDTO - Output
type WorkflowSubmitOutputDTO struct {
	WorkflowID string
	Status     string
}

// WorkflowSubmitUseCase is a usecase to submit a workflow to Cromwell
type WorkflowSubmitUseCase struct {
	CromwellClient entities.CromwellServer
}

// NewWorkflowSubmit creates a new WorkflowSubmit usecase
func NewWorkflowSubmit(c entities.CromwellServer) *WorkflowSubmitUseCase {
	return &WorkflowSubmitUseCase{CromwellClient: c}
}

// Execute submits a workflow to Cromwell
func (w *WorkflowSubmitUseCase) Execute(i *WorkflowSubmitInputDTO) (*WorkflowSubmitOutputDTO, error) {
	result, err := w.CromwellClient.Submit(i.Wdl, i.Inputs, i.Dependencies, i.Options)
	if err != nil {
		return nil, err
	}
	return &WorkflowSubmitOutputDTO{WorkflowID: result.ID, Status: result.Status}, nil
}
