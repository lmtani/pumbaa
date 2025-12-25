package cromwell

import (
	"context"

	"github.com/lmtani/pumbaa/internal/infrastructure/chat/agent/tools/types"
)

// StatusHandler handles the "status" action to get workflow status.
type StatusHandler struct {
	repo Repository
}

// NewStatusHandler creates a new StatusHandler.
func NewStatusHandler(repo Repository) *StatusHandler {
	return &StatusHandler{repo: repo}
}

// Handle implements types.Handler.
func (h *StatusHandler) Handle(ctx context.Context, input types.Input) (types.Output, error) {
	if input.WorkflowID == "" {
		return types.NewErrorOutput("status", "workflow_id is required"), nil
	}

	status, err := h.repo.GetStatus(ctx, input.WorkflowID)
	if err != nil {
		return types.NewErrorOutput("status", err.Error()), nil
	}

	return types.NewSuccessOutput("status", map[string]interface{}{
		"id":     input.WorkflowID,
		"status": string(status),
	}), nil
}
