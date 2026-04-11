// Package cromwell provides action handlers for Cromwell workflow operations.
package cromwell

import (
	"context"

	"github.com/lmtani/pumbaa/internal/application/ports"
	"github.com/lmtani/pumbaa/internal/domain/workflow"
	"github.com/lmtani/pumbaa/internal/infrastructure/agents/tools/types"
)

// QueryHandler handles the "query" action to search Cromwell workflows.
type QueryHandler struct {
	repo ports.WorkflowReader
}

// NewQueryHandler creates a new QueryHandler.
func NewQueryHandler(repo ports.WorkflowReader) *QueryHandler {
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

	workflows := make([]map[string]any, 0, len(result.Workflows))
	for _, wf := range result.Workflows {
		workflows = append(workflows, map[string]any{
			"id":           wf.ID,
			"name":         wf.Name,
			"status":       string(wf.Status),
			"submitted_at": wf.SubmittedAt,
			"start":        wf.Start,
			"end":          wf.End,
			"labels":       wf.Labels,
		})
	}

	return types.NewSuccessOutput("query", map[string]any{
		"total":     result.TotalCount,
		"workflows": workflows,
	}), nil
}
