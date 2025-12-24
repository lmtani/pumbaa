package wdl

import (
	"context"

	"github.com/lmtani/pumbaa/internal/infrastructure/agent/tools/types"
)

// SearchHandler handles the "wdl_search" action to search tasks/workflows.
type SearchHandler struct {
	repo Repository
}

// NewSearchHandler creates a new SearchHandler.
func NewSearchHandler(repo Repository) *SearchHandler {
	return &SearchHandler{repo: repo}
}

// Handle implements types.Handler.
func (h *SearchHandler) Handle(_ context.Context, input types.Input) (types.Output, error) {
	const action = "wdl_search"

	if h.repo == nil {
		return types.NewErrorOutput(action, notConfiguredError), nil
	}

	if input.Query == "" {
		return types.NewErrorOutput(action, "query is required"), nil
	}

	tasks, err := h.repo.SearchTasks(input.Query)
	if err != nil {
		return types.NewErrorOutput(action, err.Error()), nil
	}

	workflows, err := h.repo.SearchWorkflows(input.Query)
	if err != nil {
		return types.NewErrorOutput(action, err.Error()), nil
	}

	taskResults := make([]map[string]interface{}, 0, len(tasks))
	for _, t := range tasks {
		taskResults = append(taskResults, map[string]interface{}{
			"name":        t.Name,
			"source":      t.Source,
			"description": t.Description,
		})
	}

	workflowResults := make([]map[string]interface{}, 0, len(workflows))
	for _, w := range workflows {
		workflowResults = append(workflowResults, map[string]interface{}{
			"name":        w.Name,
			"source":      w.Source,
			"description": w.Description,
		})
	}

	return types.NewSuccessOutput(action, map[string]interface{}{
		"query":     input.Query,
		"tasks":     taskResults,
		"workflows": workflowResults,
	}), nil
}
