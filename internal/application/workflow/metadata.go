package workflow

import (
	"context"
	"fmt"

	"github.com/lmtani/pumbaa/internal/domain/ports"
	workflow2 "github.com/lmtani/pumbaa/internal/domain/workflow"
)

// MetadataUseCase handles workflow metadata retrieval.
type MetadataUseCase struct {
	repo ports.WorkflowRepository
}

// NewMetadataUseCase creates a new metadata use case.
func NewMetadataUseCase(repo ports.WorkflowRepository) *MetadataUseCase {
	return &MetadataUseCase{repo: repo}
}

// MetadataInput represents the input for metadata retrieval.
type MetadataInput struct {
	WorkflowID string
}

// Execute retrieves metadata for a workflow.
// Returns domain Workflow directly - no DTO transformation needed.
func (uc *MetadataUseCase) Execute(ctx context.Context, input MetadataInput) (*workflow2.Workflow, error) {
	if input.WorkflowID == "" {
		return nil, workflow2.ErrInvalidWorkflowID
	}

	wf, err := uc.repo.GetMetadata(ctx, input.WorkflowID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow metadata: %w", err)
	}

	return wf, nil
}
