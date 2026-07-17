package cromwell

import (
	"context"

	"github.com/lmtani/pumbaa/internal/application/ports"
	"github.com/lmtani/pumbaa/internal/infrastructure/agents/tools/types"
)

// maxTasksPerGroup caps how many task instances are listed per failure
// group; the group count still reports the true total.
const maxTasksPerGroup = 10

// FailuresHandler handles the "failures" action: a compact, deduplicated
// summary of what failed and why, instead of the full (potentially huge)
// metadata payload.
type FailuresHandler struct {
	fetcher ports.WorkflowMetadataFetcher
}

// NewFailuresHandler creates a new FailuresHandler.
func NewFailuresHandler(fetcher ports.WorkflowMetadataFetcher) *FailuresHandler {
	return &FailuresHandler{fetcher: fetcher}
}

// Handle implements types.Handler.
func (h *FailuresHandler) Handle(ctx context.Context, input types.Input) (types.Output, error) {
	const action = "failures"
	if input.WorkflowID == "" {
		return types.NewErrorOutput(action, "workflow_id is required"), nil
	}

	wf, err := fetchExpandedWorkflow(ctx, h.fetcher, input.WorkflowID)
	if err != nil {
		return types.NewErrorOutput(action, err.Error()), nil
	}

	summary := wf.CalculateFailureSummary()

	groups := make([]map[string]any, 0, len(summary.Groups))
	for _, g := range summary.Groups {
		tasks := make([]map[string]any, 0, min(len(g.Tasks), maxTasksPerGroup))
		for i, t := range g.Tasks {
			if i >= maxTasksPerGroup {
				break
			}
			entry := map[string]any{"task": t.Name}
			if t.Stderr != "" {
				entry["stderr"] = t.Stderr
			}
			tasks = append(tasks, entry)
		}
		groups = append(groups, map[string]any{
			"count":         len(g.Tasks),
			"error":         g.Sample,
			"tasks":         tasks,
			"tasks_omitted": max(0, len(g.Tasks)-maxTasksPerGroup),
		})
	}

	return types.NewSuccessOutput(action, map[string]any{
		"id":           input.WorkflowID,
		"status":       string(wf.Status),
		"failed_tasks": summary.FailedTasks,
		"groups":       groups,
		"hint":         "use action=read_log with a stderr path (or workflow_id+task) to inspect a failure",
	}), nil
}
