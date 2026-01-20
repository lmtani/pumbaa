package workflow

import (
	"context"

	"github.com/lmtani/pumbaa/internal/application"
	"github.com/lmtani/pumbaa/internal/application/ports"
	workflow2 "github.com/lmtani/pumbaa/internal/domain/workflow"
)

// MetadataUseCase handles workflow metadata retrieval.
type MetadataUseCase struct {
	reader ports.WorkflowMetadataReader
}

// NewMetadataUseCase creates a new metadata use case.
func NewMetadataUseCase(reader ports.WorkflowMetadataReader) *MetadataUseCase {
	return &MetadataUseCase{reader: reader}
}

// MetadataInput represents the input for metadata retrieval.
type MetadataInput struct {
	WorkflowID string
}

// Execute retrieves metadata for a workflow.
// Returns domain Workflow directly - no DTO transformation needed.
func (uc *MetadataUseCase) Execute(ctx context.Context, input MetadataInput) (*workflow2.Workflow, error) {
	if input.WorkflowID == "" {
		return nil, application.NewInputValidationError("workflowID", "is required")
	}

	wf, err := uc.reader.GetMetadata(ctx, input.WorkflowID)
	if err != nil {
		return nil, application.NewUseCaseError("metadata", "failed to get workflow metadata", err)
	}

	return wf, nil
}
