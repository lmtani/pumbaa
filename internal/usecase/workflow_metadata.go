package usecase

import (
	"github.com/lmtani/pumbaa/internal/entities"
)

// WorkflowMetadataInputDTO - Input
type WorkflowMetadataInputDTO struct {
	WorkflowID string
}

// WorkflowMetadataOutputDTO - Output
type WorkflowMetadataOutputDTO struct {
	WorkflowID string
	Metadata   entities.MetadataResponse
}

// WorkflowMetadataUseCase is a usecase to get metadata from Cromwell
type WorkflowMetadataUseCase struct {
	CromwellClient entities.CromwellServer
}

// NewWorkflowMetadata creates a new WorkflowMetadata usecase
func NewWorkflowMetadata(c entities.CromwellServer) *WorkflowMetadataUseCase {
	return &WorkflowMetadataUseCase{CromwellClient: c}
}

// Execute gets metadata from Cromwell
func (w *WorkflowMetadataUseCase) Execute(i *WorkflowMetadataInputDTO) (*WorkflowMetadataOutputDTO, error) {
	params := entities.ParamsMetadataGet{
		ExcludeKey: []string{"executionEvents", "jes", "inputs"},
	}
	result, err := w.CromwellClient.Metadata(i.WorkflowID, &params)
	if err != nil {
		return nil, err
	}
	return &WorkflowMetadataOutputDTO{WorkflowID: i.WorkflowID, Metadata: result}, nil
}
