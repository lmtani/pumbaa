// cost.go contains per-task cost breakdown logic for a workflow. It reuses
// the same attempt-cost estimate as the preemption analysis, but aggregates
// every task (preemptible or not) so users can see where the money goes.
package workflow

import (
	"sort"
	"time"
)

// TaskCost is a Value Object with the aggregated cost of one task (summed
// across its shards and attempts). Real dollars and resource estimates are
// different units, so they are kept in separate fields and never added
// together.
type TaskCost struct {
	Name          string
	ActualCost    float64 // Dollars: vmCostPerHour × billed hours, for attempts that report both
	EstimatedCost float64 // Resource-hours (cpu × memGB × hours) for attempts without real cost data
	VMHours       float64 // Sum of billed compute time across shards/attempts, in hours
	ShardCount    int     // Number of shards
	AttemptCount  int     // Total attempts across shards
	Preemptible   bool    // True if the task ran on preemptible VMs
	// Percent is the task's share within its unit: of the workflow's actual
	// dollars when any exist, otherwise of the estimated resource-hours.
	Percent float64
}

// CostBreakdown is a Value Object with per-task costs for a workflow,
// sorted by actual cost (then estimate) descending.
type CostBreakdown struct {
	Tasks []TaskCost
	// ActualTotal is the sum of real dollar costs (reconstructed from loaded
	// metadata). EstimatedTotal is the sum of resource-hours estimates — a
	// different unit, reported separately and never added to ActualTotal.
	ActualTotal    float64
	EstimatedTotal float64
	// SubworkflowsPending is the number of subworkflow calls whose metadata
	// was not loaded, so their tasks are absent from this breakdown.
	SubworkflowsPending int
}

// HasEstimates reports whether any attempt lacked real cost data and fell
// back to the resource-hours estimate.
func (b *CostBreakdown) HasEstimates() bool {
	return b.EstimatedTotal > 0
}

// CalculateCostBreakdown aggregates the cost of every task in the workflow,
// recursing into any loaded subworkflow metadata. Subworkflow calls whose
// metadata has not been fetched are counted in SubworkflowsPending so callers
// can tell the breakdown is partial.
func (w *Workflow) CalculateCostBreakdown() *CostBreakdown {
	type agg struct {
		actual    float64
		estimated float64
		vmHours   float64
		shards    map[int]bool
		attempts  int
		preempt   bool
	}
	aggregations := make(map[string]*agg)
	var order []string
	pending := 0
	// A single anchor for the whole traversal: running attempts accrue their
	// cost up to this instant.
	now := time.Now()

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
					a = &agg{shards: make(map[int]bool)}
					aggregations[short] = a
					order = append(order, short)
				}
				actual, estimated := attemptCostParts(call, now)
				a.actual += actual
				a.estimated += estimated
				a.attempts++
				a.shards[call.ShardIndex] = true
				if IsPreemptible(call.Preemptible) {
					a.preempt = true
				}
				if actual > 0 {
					a.vmHours += billableHours(call, now)
				}
			}
		}
	}
	walk(w.Calls)

	breakdown := &CostBreakdown{
		SubworkflowsPending: pending,
	}
	for _, name := range order {
		a := aggregations[name]
		breakdown.ActualTotal += a.actual
		breakdown.EstimatedTotal += a.estimated
		breakdown.Tasks = append(breakdown.Tasks, TaskCost{
			Name:          name,
			ActualCost:    a.actual,
			EstimatedCost: a.estimated,
			VMHours:       a.vmHours,
			ShardCount:    len(a.shards),
			AttemptCount:  a.attempts,
			Preemptible:   a.preempt,
		})
	}

	// Percent within a single unit: prefer real dollars; when the workflow
	// has none (e.g. a backend that reports no cost), fall back to the
	// estimate so relative weights are still meaningful.
	for i := range breakdown.Tasks {
		switch {
		case breakdown.ActualTotal > 0:
			breakdown.Tasks[i].Percent = breakdown.Tasks[i].ActualCost / breakdown.ActualTotal * 100
		case breakdown.EstimatedTotal > 0:
			breakdown.Tasks[i].Percent = breakdown.Tasks[i].EstimatedCost / breakdown.EstimatedTotal * 100
		}
	}

	sort.Slice(breakdown.Tasks, func(i, j int) bool {
		if breakdown.Tasks[i].ActualCost != breakdown.Tasks[j].ActualCost {
			return breakdown.Tasks[i].ActualCost > breakdown.Tasks[j].ActualCost
		}
		return breakdown.Tasks[i].EstimatedCost > breakdown.Tasks[j].EstimatedCost
	})

	return breakdown
}

// attemptCostParts splits one attempt's cost into its unit: real dollars when
// Cromwell reported a VM rate and a billable window, or a resource-hours
// estimate otherwise. Exactly one of the two is non-zero.
func attemptCostParts(call Call, now time.Time) (actual, estimated float64) {
	if call.VMCostPerHour > 0 {
		if hours := billableHours(call, now); hours > 0 {
			return call.VMCostPerHour * hours, 0
		}
	}
	return 0, resourceHoursEstimate(call, now)
}

// resourceHoursEstimate is the dimensionless fallback cost (cpu × memGB ×
// hours) used when an attempt has no real cost data.
func resourceHoursEstimate(call Call, now time.Time) float64 {
	cpu := parseCPUFromString(call.CPU)
	if cpu <= 0 {
		cpu = 1
	}
	mem := parseMemoryGBFromString(call.Memory)
	if mem <= 0 {
		mem = 1
	}

	durationHours := billableHours(call, now)
	if durationHours <= 0 {
		durationHours = 0.01 // Minimum 36 seconds
	}

	return cpu * mem * durationHours
}
