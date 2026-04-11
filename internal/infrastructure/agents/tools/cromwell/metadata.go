package cromwell

import (
	"context"

	"github.com/lmtani/pumbaa/internal/application/ports"
	"github.com/lmtani/pumbaa/internal/infrastructure/agents/tools/types"
)

// MetadataHandler handles the "metadata" action to get workflow metadata.
type MetadataHandler struct {
	repo ports.WorkflowReader
}

// NewMetadataHandler creates a new MetadataHandler.
func NewMetadataHandler(repo ports.WorkflowReader) *MetadataHandler {
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

	calls := make(map[string][]map[string]any)
	for callName, callInstances := range wf.Calls {
		instances := make([]map[string]any, 0, len(callInstances))
		for _, call := range callInstances {
			instances = append(instances, map[string]any{
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

	return types.NewSuccessOutput("metadata", map[string]any{
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
