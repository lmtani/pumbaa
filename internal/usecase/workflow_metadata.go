package usecase

import (
	"github.com/lmtani/pumbaa/internal/ports"
	"github.com/lmtani/pumbaa/internal/types"
)

// WorkflowMetadataInputDTO - Input
type WorkflowMetadataInputDTO struct {
	WorkflowID string
}

// WorkflowMetadataOutputDTO - Output
type WorkflowMetadataOutputDTO struct {
	WorkflowID string
	Metadata   types.MetadataResponse
}

// WorkflowMetadataUseCase is a usecase to get metadata from Cromwell
type WorkflowMetadataUseCase struct {
	CromwellClient ports.CromwellServer
}

// NewWorkflowMetadata creates a new WorkflowMetadata usecase
func NewWorkflowMetadata(c ports.CromwellServer) *WorkflowMetadataUseCase {
	return &WorkflowMetadataUseCase{CromwellClient: c}
}

// Execute gets metadata from Cromwell
func (w *WorkflowMetadataUseCase) Execute(i *WorkflowMetadataInputDTO) (*WorkflowMetadataOutputDTO, error) {
	params := types.ParamsMetadataGet{
		ExcludeKey: []string{"executionEvents", "jes", "inputs"},
	}
	result, err := w.CromwellClient.Metadata(i.WorkflowID, &params)
	if err != nil {
		return nil, err
	}
	return &WorkflowMetadataOutputDTO{WorkflowID: i.WorkflowID, Metadata: result}, nil
}
