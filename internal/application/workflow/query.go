package workflow

import (
	"context"
	"time"

	"github.com/lmtani/pumbaa/internal/domain/ports"
	workflow2 "github.com/lmtani/pumbaa/internal/domain/workflow"
)

// QueryUseCase handles workflow queries.
type QueryUseCase struct {
	repo ports.WorkflowRepository
}

// NewQueryUseCase creates a new query use case.
func NewQueryUseCase(repo ports.WorkflowRepository) *QueryUseCase {
	return &QueryUseCase{repo: repo}
}

// QueryInput represents the input for workflow queries.
type QueryInput struct {
	Name     string
	Status   []string
	Labels   map[string]string
	Page     int
	PageSize int
}

// QueryOutput represents the output of workflow queries.
type QueryOutput struct {
	Workflows  []WorkflowSummary
	TotalCount int
	Page       int
	PageSize   int
}

// WorkflowSummary represents a summary of a workflow for listing.
type WorkflowSummary struct {
	ID          string
	Name        string
	Status      string
	SubmittedAt time.Time
	Start       time.Time
	End         time.Time
}

// Execute queries workflows based on filters.
func (uc *QueryUseCase) Execute(ctx context.Context, input QueryInput) (*QueryOutput, error) {
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

	result, err := uc.repo.Query(ctx, filter)
	if err != nil {
		return nil, err
	}

	output := &QueryOutput{
		Workflows:  make([]WorkflowSummary, 0, len(result.Workflows)),
		TotalCount: result.TotalCount,
		Page:       input.Page,
		PageSize:   input.PageSize,
	}

	for _, wf := range result.Workflows {
		output.Workflows = append(output.Workflows, WorkflowSummary{
			ID:          wf.ID,
			Name:        wf.Name,
			Status:      string(wf.Status),
			SubmittedAt: wf.SubmittedAt,
			Start:       wf.Start,
			End:         wf.End,
		})
	}

	return output, nil
}
