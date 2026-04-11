package workflow

import (
	"context"

	"github.com/lmtani/pumbaa/internal/application"
	"github.com/lmtani/pumbaa/internal/application/ports"
	workflow2 "github.com/lmtani/pumbaa/internal/domain/workflow"
)

// QueryUseCase handles workflow queries.
type QueryUseCase struct {
	queryer ports.WorkflowQuerier
}

// NewQueryUseCase creates a new query use case.
func NewQueryUseCase(queryer ports.WorkflowQuerier) *QueryUseCase {
	return &QueryUseCase{queryer: queryer}
}

// QueryInput represents the input for workflow queries.
type QueryInput struct {
	Name     string
	Status   []string
	Labels   map[string]string
	Page     int
	PageSize int
}

// Execute queries workflows based on filters.
// Returns domain QueryResult directly - no DTO transformation needed.
func (uc *QueryUseCase) Execute(ctx context.Context, input QueryInput) (*workflow2.QueryResult, error) {
	// Convert string statuses to domain Status
	statuses := make([]workflow2.Status, 0, len(input.Status))
	for _, s := range input.Status {
		statuses = append(statuses, workflow2.Status(s))
	}

	filter := workflow2.QueryFilter{
		Name:     input.Name,
		Status:   statuses,
		Labels:   input.Labels,
		Page:     input.Page,
		PageSize: input.PageSize,
	}

	result, err := uc.queryer.Query(ctx, filter)
	if err != nil {
		return nil, application.NewUseCaseError("query", "failed to query workflows", err)
	}

	return result, nil
}
