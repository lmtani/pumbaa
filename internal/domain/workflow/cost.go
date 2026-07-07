// cost.go contains per-task cost breakdown logic for a workflow. It reuses
// the same attempt-cost estimate as the preemption analysis, but aggregates
// every task (preemptible or not) so users can see where the money goes.
package workflow

import "sort"

// TaskCost is a Value Object with the aggregated cost of one task (summed
// across its shards and attempts).
type TaskCost struct {
	Name         string  // Short task name (without workflow prefix)
	TotalCost    float64 // Cost of all shards/attempts
	VMHours      float64 // Sum of compute time across shards/attempts, in hours
	ShardCount   int     // Number of shards
	AttemptCount int     // Total attempts across shards
	Preemptible  bool    // True if the task ran on preemptible VMs
	FromActual   bool    // True when cost came from real vmCostPerHour (not an estimate)
	Percent      float64 // Share of the workflow's TotalCost
}

// CostBreakdown is a Value Object with per-task costs for a workflow,
// sorted by cost descending.
type CostBreakdown struct {
	Tasks      []TaskCost
	TotalCost  float64 // Sum of Tasks' cost (reconstructed from loaded metadata)
	FromActual bool    // True when every task's cost came from real vmCostPerHour
	// SubworkflowsPending is the number of subworkflow calls whose metadata
	// was not loaded, so their tasks are absent from this breakdown.
	SubworkflowsPending int
}

// CalculateCostBreakdown aggregates the cost of every task in the workflow,
// recursing into any loaded subworkflow metadata. Subworkflow calls whose
// metadata has not been fetched are counted in SubworkflowsPending so callers
// can tell the breakdown is partial.
func (w *Workflow) CalculateCostBreakdown() *CostBreakdown {
	type agg struct {
		cost       float64
		vmHours    float64
		shards     map[int]bool
		attempts   int
		preempt    bool
		fromActual bool
		allActual  bool
	}
	aggregations := make(map[string]*agg)
	var order []string
	pending := 0

	var walk func(calls map[string][]Call)
	walk = func(calls map[string][]Call) {
		for callName, callList := range calls {
			short := preemptionShortTaskName(callName)
			for _, call := range callList {
				if call.SubWorkflowMetadata != nil {
					walk(call.SubWorkflowMetadata.Calls)
					continue
				}
				if call.SubWorkflowID != "" {
					// A subworkflow call whose metadata was not loaded.
					pending++
					continue
				}

				a := aggregations[short]
				if a == nil {
					a = &agg{shards: make(map[int]bool), allActual: true}
					aggregations[short] = a
					order = append(order, short)
				}
				cost := calculateAttemptCost(call)
				a.cost += cost
				a.attempts++
				a.shards[call.ShardIndex] = true
				if IsPreemptible(call.Preemptible) {
					a.preempt = true
				}
				if call.VMCostPerHour > 0 && billableHours(call) > 0 {
					a.vmHours += billableHours(call)
					a.fromActual = true
				} else {
					a.allActual = false
				}
			}
		}
	}
	walk(w.Calls)

	breakdown := &CostBreakdown{
		FromActual:          true,
		SubworkflowsPending: pending,
	}
	for _, name := range order {
		a := aggregations[name]
		breakdown.TotalCost += a.cost
		if !a.allActual {
			breakdown.FromActual = false
		}
		breakdown.Tasks = append(breakdown.Tasks, TaskCost{
			Name:         name,
			TotalCost:    a.cost,
			VMHours:      a.vmHours,
			ShardCount:   len(a.shards),
			AttemptCount: a.attempts,
			Preemptible:  a.preempt,
			FromActual:   a.fromActual && a.allActual,
		})
	}

	if breakdown.TotalCost > 0 {
		for i := range breakdown.Tasks {
			breakdown.Tasks[i].Percent = breakdown.Tasks[i].TotalCost / breakdown.TotalCost * 100
		}
	}

	sort.Slice(breakdown.Tasks, func(i, j int) bool {
		return breakdown.Tasks[i].TotalCost > breakdown.Tasks[j].TotalCost
	})

	return breakdown
}
