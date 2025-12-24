package cromwell

import (
	"context"

	"github.com/lmtani/pumbaa/internal/infrastructure/agent/tools/types"
)

// OutputsHandler handles the "outputs" action to get workflow output files.
type OutputsHandler struct {
	repo Repository
}

// NewOutputsHandler creates a new OutputsHandler.
func NewOutputsHandler(repo Repository) *OutputsHandler {
	return &OutputsHandler{repo: repo}
}

// Handle implements types.Handler.
func (h *OutputsHandler) Handle(ctx context.Context, input types.Input) (types.Output, error) {
	if input.WorkflowID == "" {
		return types.NewErrorOutput("outputs", "workflow_id is required"), nil
	}

	outputs, err := h.repo.GetOutputs(ctx, input.WorkflowID)
	if err != nil {
		return types.NewErrorOutput("outputs", err.Error()), nil
	}

	return types.NewSuccessOutput("outputs", map[string]interface{}{
		"id":      input.WorkflowID,
		"outputs": outputs,
	}), nil
}
