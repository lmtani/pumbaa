package usecase

import "github.com/lmtani/pumbaa/internal/ports"

// WorkflowKillInputDTO - Input
type WorkflowKillInputDTO struct {
	WorkflowID string
}

// WorkflowKillOutputDTO - Output
type WorkflowKillOutputDTO struct {
	WorkflowID string
	Status     string
}

// WorkflowKillUseCase is a usecase to kill a workflow in Cromwell
type WorkflowKillUseCase struct {
	CromwellClient ports.CromwellServer
}

// NewWorkflowKill creates a new WorkflowKill usecase
func NewWorkflowKill(c ports.CromwellServer) *WorkflowKillUseCase {
	return &WorkflowKillUseCase{CromwellClient: c}
}

// Execute kills a workflow in Cromwell
func (w *WorkflowKillUseCase) Execute(i *WorkflowKillInputDTO) (*WorkflowKillOutputDTO, error) {
	result, err := w.CromwellClient.Kill(i.WorkflowID)
	if err != nil {
		return nil, err
	}
	return &WorkflowKillOutputDTO{Status: result.Status}, nil
}
