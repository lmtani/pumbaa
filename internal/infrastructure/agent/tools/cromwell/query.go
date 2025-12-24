// Package cromwell provides action handlers for Cromwell workflow operations.
package cromwell

import (
	"context"

	"github.com/lmtani/pumbaa/internal/domain/workflow"
	"github.com/lmtani/pumbaa/internal/infrastructure/agent/tools/types"
)

// Repository defines the interface for Cromwell operations.
type Repository interface {
	Query(ctx context.Context, filter workflow.QueryFilter) (*workflow.QueryResult, error)
	GetStatus(ctx context.Context, workflowID string) (workflow.Status, error)
	GetMetadata(ctx context.Context, workflowID string) (*workflow.Workflow, error)
	GetOutputs(ctx context.Context, workflowID string) (map[string]interface{}, error)
	GetLogs(ctx context.Context, workflowID string) (map[string][]workflow.CallLog, error)
}

// QueryHandler handles the "query" action to search Cromwell workflows.
type QueryHandler struct {
	repo Repository
}

// NewQueryHandler creates a new QueryHandler.
func NewQueryHandler(repo Repository) *QueryHandler {
	return &QueryHandler{repo: repo}
}

// Handle implements types.Handler.
func (h *QueryHandler) Handle(ctx context.Context, input types.Input) (types.Output, error) {
	pageSize := input.PageSize
	if pageSize <= 0 {
		pageSize = 10 // Default
	}

	filter := workflow.QueryFilter{
		Name:     input.Name,
		PageSize: pageSize,
	}
	if input.Status != "" {
		filter.Status = []workflow.Status{workflow.Status(input.Status)}
	}

	result, err := h.repo.Query(ctx, filter)
	if err != nil {
		return types.NewErrorOutput("query", err.Error()), nil
	}

	workflows := make([]map[string]interface{}, 0, len(result.Workflows))
	for _, wf := range result.Workflows {
		workflows = append(workflows, map[string]interface{}{
			"id":           wf.ID,
			"name":         wf.Name,
			"status":       string(wf.Status),
			"submitted_at": wf.SubmittedAt,
			"start":        wf.Start,
			"end":          wf.End,
			"labels":       wf.Labels,
		})
	}

	return types.NewSuccessOutput("query", map[string]interface{}{
		"total":     result.TotalCount,
		"workflows": workflows,
	}), nil
}
