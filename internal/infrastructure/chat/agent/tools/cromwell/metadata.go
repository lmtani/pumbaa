package cromwell

import (
	"context"

	"github.com/lmtani/pumbaa/internal/infrastructure/chat/agent/tools/types"
)

// MetadataHandler handles the "metadata" action to get workflow metadata.
type MetadataHandler struct {
	repo Repository
}

// NewMetadataHandler creates a new MetadataHandler.
func NewMetadataHandler(repo Repository) *MetadataHandler {
	return &MetadataHandler{repo: repo}
}

// Handle implements types.Handler.
func (h *MetadataHandler) Handle(ctx context.Context, input types.Input) (types.Output, error) {
	if input.WorkflowID == "" {
		return types.NewErrorOutput("metadata", "workflow_id is required"), nil
	}

	wf, err := h.repo.GetMetadata(ctx, input.WorkflowID)
	if err != nil {
		return types.NewErrorOutput("metadata", err.Error()), nil
	}

	calls := make(map[string][]map[string]interface{})
	for callName, callInstances := range wf.Calls {
		instances := make([]map[string]interface{}, 0, len(callInstances))
		for _, call := range callInstances {
			instances = append(instances, map[string]interface{}{
				"name":        call.Name,
				"status":      string(call.Status),
				"start":       call.Start,
				"end":         call.End,
				"attempt":     call.Attempt,
				"shard_index": call.ShardIndex,
			})
		}
		calls[callName] = instances
	}

	return types.NewSuccessOutput("metadata", map[string]interface{}{
		"id":           wf.ID,
		"name":         wf.Name,
		"status":       string(wf.Status),
		"submitted_at": wf.SubmittedAt,
		"start":        wf.Start,
		"end":          wf.End,
		"inputs":       wf.Inputs,
		"outputs":      wf.Outputs,
		"calls":        calls,
		"labels":       wf.Labels,
	}), nil
}
