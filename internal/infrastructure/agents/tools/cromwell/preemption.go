package cromwell

import (
	"context"

	"github.com/lmtani/pumbaa/internal/application/ports"
	"github.com/lmtani/pumbaa/internal/infrastructure/agents/tools/types"
)

// PreemptionHandler handles the "preemption" action: preemption efficiency
// statistics with the tasks losing the most work to preemptions.
type PreemptionHandler struct {
	fetcher ports.WorkflowMetadataFetcher
}

// NewPreemptionHandler creates a new PreemptionHandler.
func NewPreemptionHandler(fetcher ports.WorkflowMetadataFetcher) *PreemptionHandler {
	return &PreemptionHandler{fetcher: fetcher}
}

// Handle implements types.Handler.
func (h *PreemptionHandler) Handle(ctx context.Context, input types.Input) (types.Output, error) {
	const action = "preemption"
	if input.WorkflowID == "" {
		return types.NewErrorOutput(action, "workflow_id is required"), nil
	}

	wf, err := fetchExpandedWorkflow(ctx, h.fetcher, input.WorkflowID)
	if err != nil {
		return types.NewErrorOutput(action, err.Error()), nil
	}

	s := wf.CalculatePreemptionSummary()

	problematic := make([]map[string]any, 0, len(s.ProblematicTasks))
	for _, t := range s.ProblematicTasks {
		problematic = append(problematic, map[string]any{
			"task":            t.Name,
			"shards":          t.ShardCount,
			"attempts":        t.TotalAttempts,
			"preemptions":     t.TotalPreemptions,
			"cost_efficiency": round2(t.CostEfficiency),
			"impact_percent":  round1(t.ImpactPercent),
		})
	}

	return types.NewSuccessOutput(action, map[string]any{
		"id":                 input.WorkflowID,
		"total_tasks":        s.TotalTasks,
		"preemptible_tasks":  s.PreemptibleTasks,
		"total_attempts":     s.TotalAttempts,
		"total_preemptions":  s.TotalPreemptions,
		"overall_efficiency": round2(s.OverallEfficiency),
		"cost_efficiency":    round2(s.CostEfficiency),
		"cost_unit":          s.CostUnit,
		"problematic_tasks":  problematic,
	}), nil
}
