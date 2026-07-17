package cromwell

import (
	"context"
	"math"

	"github.com/lmtani/pumbaa/internal/application/ports"
	"github.com/lmtani/pumbaa/internal/infrastructure/agents/tools/types"
)

// CostHandler handles the "cost" action: the per-task cost breakdown,
// keeping real dollars and resource-hour estimates in separate fields.
type CostHandler struct {
	fetcher ports.WorkflowMetadataFetcher
}

// NewCostHandler creates a new CostHandler.
func NewCostHandler(fetcher ports.WorkflowMetadataFetcher) *CostHandler {
	return &CostHandler{fetcher: fetcher}
}

// Handle implements types.Handler.
func (h *CostHandler) Handle(ctx context.Context, input types.Input) (types.Output, error) {
	const action = "cost"
	if input.WorkflowID == "" {
		return types.NewErrorOutput(action, "workflow_id is required"), nil
	}

	wf, err := fetchExpandedWorkflow(ctx, h.fetcher, input.WorkflowID)
	if err != nil {
		return types.NewErrorOutput(action, err.Error()), nil
	}

	b := wf.CalculateCostBreakdown()

	tasks := make([]map[string]any, 0, len(b.Tasks))
	for _, t := range b.Tasks {
		entry := map[string]any{
			"task":     t.Name,
			"percent":  round1(t.Percent),
			"vm_hours": round1(t.VMHours),
			"shards":   t.ShardCount,
			"attempts": t.AttemptCount,
		}
		if t.ActualCost > 0 {
			entry["cost_usd"] = round2(t.ActualCost)
		}
		if t.EstimatedCost > 0 {
			entry["estimated_resource_hours"] = round2(t.EstimatedCost)
		}
		if !t.Preemptible {
			entry["on_demand"] = true
		}
		tasks = append(tasks, entry)
	}

	data := map[string]any{
		"id":             input.WorkflowID,
		"status":         string(wf.Status),
		"total_cost_usd": round2(b.ActualTotal),
		"tasks":          tasks,
		"note":           "running tasks accrue cost up to now; cost_usd comes from real VM rates, estimated_resource_hours is a dimensionless fallback (cpu×memGB×hours), never added to dollars",
	}
	if b.EstimatedTotal > 0 {
		data["estimated_total_resource_hours"] = round2(b.EstimatedTotal)
	}
	if b.SubworkflowsPending > 0 {
		data["subworkflows_not_included"] = b.SubworkflowsPending
	}

	// The API total is authoritative when the server provides it.
	if apiCost, currency, err := h.fetcher.GetWorkflowCost(ctx, input.WorkflowID); err == nil && apiCost > 0 {
		data["api_total"] = round2(apiCost)
		data["api_currency"] = currency
	}

	return types.NewSuccessOutput(action, data), nil
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }
func round2(v float64) float64 { return math.Round(v*100) / 100 }
