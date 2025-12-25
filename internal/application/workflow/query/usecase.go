// Package query contains the use case for querying workflows.
package query

import (
	"context"
	"time"

	"github.com/lmtani/pumbaa/internal/domain/ports"
	"github.com/lmtani/pumbaa/internal/domain/workflow"
)

// UseCase handles workflow queries.
type UseCase struct {
	repo ports.WorkflowRepository
}

// New creates a new query use case.
func New(repo ports.WorkflowRepository) *UseCase {
	return &UseCase{repo: repo}
}

// Input represents the input for the query use case.
type Input struct {
	Name     string
	Status   []string
	Labels   map[string]string
	Page     int
	PageSize int
}

// Output represents the output of the query use case.
type Output struct {
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
func (uc *UseCase) Execute(ctx context.Context, input Input) (*Output, error) {
	// Convert string statuses to domain Status
	statuses := make([]workflow.Status, 0, len(input.Status))
	for _, s := range input.Status {
		statuses = append(statuses, workflow.Status(s))
	}

	filter := workflow.QueryFilter{
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

	output := &Output{
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
