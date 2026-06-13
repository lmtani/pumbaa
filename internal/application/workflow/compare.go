package workflow

import (
	"context"

	"github.com/lmtani/pumbaa/internal/application"
	"github.com/lmtani/pumbaa/internal/application/ports"
	workflow2 "github.com/lmtani/pumbaa/internal/domain/workflow"
)

// CompareUseCase compares two workflow runs by fetching their metadata and
// diffing them in the domain layer.
type CompareUseCase struct {
	reader ports.WorkflowMetadataReader
}

// NewCompareUseCase creates a new compare use case.
func NewCompareUseCase(reader ports.WorkflowMetadataReader) *CompareUseCase {
	return &CompareUseCase{reader: reader}
}

// CompareInput identifies the two workflow runs to compare.
type CompareInput struct {
	WorkflowIDA string
	WorkflowIDB string
}

// Execute fetches both workflows' metadata and returns their diff.
func (uc *CompareUseCase) Execute(ctx context.Context, input CompareInput) (*workflow2.RunDiff, error) {
	if input.WorkflowIDA == "" {
		return nil, application.NewInputValidationError("workflowIDA", "is required")
	}
	if input.WorkflowIDB == "" {
		return nil, application.NewInputValidationError("workflowIDB", "is required")
	}

	a, err := uc.reader.GetMetadata(ctx, input.WorkflowIDA)
	if err != nil {
		return nil, application.NewUseCaseError("compare", "failed to get metadata for first workflow", err)
	}

	b, err := uc.reader.GetMetadata(ctx, input.WorkflowIDB)
	if err != nil {
		return nil, application.NewUseCaseError("compare", "failed to get metadata for second workflow", err)
	}

	return workflow2.CompareWorkflows(a, b), nil
}
