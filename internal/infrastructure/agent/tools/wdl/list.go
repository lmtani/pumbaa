package wdl

import (
	"context"

	"github.com/lmtani/pumbaa/internal/infrastructure/agent/tools/types"
)

// ListHandler handles the "wdl_list" action to list all indexed WDL tasks and workflows.
type ListHandler struct {
	repo Repository
}

// NewListHandler creates a new ListHandler.
func NewListHandler(repo Repository) *ListHandler {
	return &ListHandler{repo: repo}
}

// Handle implements types.Handler.
func (h *ListHandler) Handle(_ context.Context, _ types.Input) (types.Output, error) {
	const action = "wdl_list"

	if h.repo == nil {
		return types.NewErrorOutput(action, notConfiguredError), nil
	}

	index, err := h.repo.List()
	if err != nil {
		return types.NewErrorOutput(action, err.Error()), nil
	}

	tasks := make([]string, 0, len(index.Tasks))
	for name := range index.Tasks {
		tasks = append(tasks, name)
	}

	workflows := make([]string, 0, len(index.Workflows))
	for name := range index.Workflows {
		workflows = append(workflows, name)
	}

	return types.NewSuccessOutput(action, map[string]interface{}{
		"directory":      index.Directory,
		"indexed_at":     index.IndexedAt,
		"task_count":     len(tasks),
		"workflow_count": len(workflows),
		"tasks":          tasks,
		"workflows":      workflows,
	}), nil
}
