package usecase

import (
	"github.com/lmtani/pumbaa/internal/entities"
	"github.com/lmtani/pumbaa/internal/interfaces"
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
	CromwellClient interfaces.CromwellServer
}

// NewWorkflowMetadata creates a new WorkflowMetadata usecase
func NewWorkflowMetadata(c interfaces.CromwellServer) *WorkflowMetadataUseCase {
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
