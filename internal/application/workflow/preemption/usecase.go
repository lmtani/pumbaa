// Package preemption provides analysis of preemptible task efficiency.
package preemption

import (
	"sort"
	"strings"
)

// TaskStats represents preemption statistics for a single task/shard.
type TaskStats struct {
	TaskName        string  // Full task name (workflow.task)
	ShardIndex      int     // Shard index (-1 for non-scattered)
	TotalAttempts   int     // Total number of attempts
	PreemptedCount  int     // Number of times preempted (attempts - 1 if successful)
	FinalStatus     string  // Final execution status
	IsPreemptible   bool    // Whether the task was configured as preemptible
	EfficiencyScore float64 // 1.0 = first try success, 0.0 = max retries used
	MaxPreemptible  int     // Max preemptible attempts from config (if available)

	// Cost-weighted metrics
	TotalCost      float64 // Total cost of all attempts (resource-hours)
	WastedCost     float64 // Cost of failed attempts (resource-hours)
	CostEfficiency float64 // 1.0 = no waste, 0.0 = all wasted
}

// ProblematicTask represents an aggregated view of a task with poor preemption efficiency.
type ProblematicTask struct {
	Name             string  // Short task name (without workflow prefix)
	ShardCount       int     // Number of shards (1 for non-scattered)
	TotalAttempts    int     // Total attempts across all shards
	TotalPreemptions int     // Total preemptions across all shards
	EfficiencyScore  float64 // Average efficiency across all shards

	// Cost-weighted metrics
	TotalCost      float64 // Total cost across all shards (resource-hours)
	WastedCost     float64 // Cost of failed attempts (resource-hours)
	CostEfficiency float64 // 1 - (WastedCost / TotalCost)
	ImpactPercent  float64 // WastedCost / WorkflowTotalWastedCost × 100
}

// WorkflowSummary represents aggregated preemption statistics for a workflow.
type WorkflowSummary struct {
	WorkflowID        string
	WorkflowName      string
	TotalTasks        int               // Total number of tasks/shards
	PreemptibleTasks  int               // Number of preemptible tasks/shards
	TotalAttempts     int               // Total attempts across all tasks
	TotalPreemptions  int               // Total preemptions across all tasks
	OverallEfficiency float64           // Average efficiency across all preemptible tasks
	ProblematicTasks  []ProblematicTask // Tasks with low efficiency (aggregated by task name)

	// Cost-weighted metrics
	TotalCost      float64 // Total cost of all attempts (resource-hours)
	WastedCost     float64 // Cost of failed attempts (resource-hours)
	CostEfficiency float64 // 1 - (WastedCost / TotalCost)
	CostUnit       string  // Unit for cost display (e.g., "resource-hours" or "$")
}

// CallData represents the minimal call data needed for analysis.
type CallData struct {
	Name            string
	ShardIndex      int
	Attempt         int
	ExecutionStatus string
	Preemptible     string // "true", "false", or number of max attempts
	ReturnCode      *int

	// Resource information for cost-weighted analysis
	CPU           float64 // Number of CPUs
	MemoryGB      float64 // Memory in GB
	DurationHours float64 // Duration in hours
	VMCostPerHour float64 // If available from cloud provider
}

// AttemptCost calculates the estimated cost of a single attempt.
// Cost = CPU × Memory(GB) × Duration(hours)
// This gives a "resource-hours" metric that's proportional to actual cloud costs.
func (c CallData) AttemptCost() float64 {
	// If we have actual VM cost per hour, use it
	if c.VMCostPerHour > 0 && c.DurationHours > 0 {
		return c.VMCostPerHour * c.DurationHours
	}

	// Otherwise, estimate using resource-hours
	// Default to 1 CPU, 1GB if not specified
	cpu := c.CPU
	if cpu <= 0 {
		cpu = 1
	}
	mem := c.MemoryGB
	if mem <= 0 {
		mem = 1
	}
	dur := c.DurationHours
	if dur <= 0 {
		dur = 0.01 // Minimum 36 seconds
	}

	return cpu * mem * dur
}

// Analyzer analyzes preemption efficiency from workflow metadata.
type Analyzer struct{}

// NewAnalyzer creates a new preemption analyzer.
func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

// AnalyzeWorkflow analyzes preemption statistics and returns a summary.
func (a *Analyzer) AnalyzeWorkflow(workflowID, workflowName string, calls map[string][]CallData) *WorkflowSummary {
	summary := &WorkflowSummary{
		WorkflowID:       workflowID,
		WorkflowName:     workflowName,
		ProblematicTasks: []ProblematicTask{},
		CostUnit:         "resource-hours",
	}

	var totalEfficiency float64
	var preemptibleCount int

	// Track unique task/shard combinations
	type taskKey struct {
		name  string
		shard int
	}
	seen := make(map[taskKey]bool)

	// Aggregate stats by task name (for problematic tasks report)
	type taskAggregation struct {
		shardCount       int
		totalAttempts    int
		totalPreemptions int
		totalEfficiency  float64
		shardsCounted    int
		// Cost metrics
		totalCost  float64
		wastedCost float64
	}
	taskAggregations := make(map[string]*taskAggregation)

	for callName, callList := range calls {
		// Group by shard index to calculate per-shard stats
		shardGroups := make(map[int][]CallData)
		for _, call := range callList {
			shardGroups[call.ShardIndex] = append(shardGroups[call.ShardIndex], call)
		}

		for shardIndex, attempts := range shardGroups {
			key := taskKey{name: callName, shard: shardIndex}
			if seen[key] {
				continue
			}
			seen[key] = true

			stats := a.analyzeTaskShard(attempts)
			summary.TotalTasks++

			if stats.IsPreemptible {
				summary.PreemptibleTasks++
				summary.TotalAttempts += stats.TotalAttempts
				summary.TotalPreemptions += stats.PreemptedCount
				totalEfficiency += stats.EfficiencyScore
				preemptibleCount++

				// Accumulate cost metrics
				summary.TotalCost += stats.TotalCost
				summary.WastedCost += stats.WastedCost

				// Aggregate by short task name
				taskName := shortTaskName(callName)
				if taskAggregations[taskName] == nil {
					taskAggregations[taskName] = &taskAggregation{}
				}
				agg := taskAggregations[taskName]
				agg.shardCount++
				agg.totalAttempts += stats.TotalAttempts
				agg.totalPreemptions += stats.PreemptedCount
				agg.totalEfficiency += stats.EfficiencyScore
				agg.shardsCounted++
				agg.totalCost += stats.TotalCost
				agg.wastedCost += stats.WastedCost
			}
		}
	}

	// Calculate workflow-level cost efficiency
	if summary.TotalCost > 0 {
		summary.CostEfficiency = 1.0 - (summary.WastedCost / summary.TotalCost)
	} else {
		summary.CostEfficiency = 1.0
	}

	// Build problematic tasks list from aggregations
	// Now we prioritize by wasted cost (impact) rather than just efficiency
	for taskName, agg := range taskAggregations {
		avgEfficiency := agg.totalEfficiency / float64(agg.shardsCounted)

		var costEfficiency float64
		if agg.totalCost > 0 {
			costEfficiency = 1.0 - (agg.wastedCost / agg.totalCost)
		} else {
			costEfficiency = 1.0
		}

		var impactPercent float64
		if summary.WastedCost > 0 {
			impactPercent = (agg.wastedCost / summary.WastedCost) * 100
		}

		// Track problematic tasks: either low efficiency OR significant cost impact
		// - Cost efficiency < 70% (lots of waste relative to task's total cost)
		// - OR impact > 10% (this task contributes significantly to total waste)
		if (costEfficiency < 0.7 || impactPercent > 10) && agg.totalPreemptions > 0 {
			summary.ProblematicTasks = append(summary.ProblematicTasks, ProblematicTask{
				Name:             taskName,
				ShardCount:       agg.shardCount,
				TotalAttempts:    agg.totalAttempts,
				TotalPreemptions: agg.totalPreemptions,
				EfficiencyScore:  avgEfficiency,
				TotalCost:        agg.totalCost,
				WastedCost:       agg.wastedCost,
				CostEfficiency:   costEfficiency,
				ImpactPercent:    impactPercent,
			})
		}
	}

	// Calculate overall efficiency
	if preemptibleCount > 0 {
		summary.OverallEfficiency = totalEfficiency / float64(preemptibleCount)
	} else {
		summary.OverallEfficiency = 1.0
	}

	// Sort problematic tasks by wasted cost (highest impact first)
	sort.Slice(summary.ProblematicTasks, func(i, j int) bool {
		return summary.ProblematicTasks[i].WastedCost > summary.ProblematicTasks[j].WastedCost
	})

	return summary
}

// analyzeTaskShard analyzes preemption stats for a single task/shard.
func (a *Analyzer) analyzeTaskShard(attempts []CallData) TaskStats {
	stats := TaskStats{}

	if len(attempts) == 0 {
		stats.EfficiencyScore = 1.0
		stats.CostEfficiency = 1.0
		return stats
	}

	// Sort attempts by attempt number
	sort.Slice(attempts, func(i, j int) bool {
		return attempts[i].Attempt < attempts[j].Attempt
	})

	stats.TotalAttempts = len(attempts)

	// Get final attempt info
	finalAttempt := attempts[len(attempts)-1]
	stats.FinalStatus = finalAttempt.ExecutionStatus
	stats.IsPreemptible = IsPreemptible(finalAttempt.Preemptible)
	stats.MaxPreemptible = ParseMaxPreemptible(finalAttempt.Preemptible)

	// Calculate costs for all attempts
	for i, attempt := range attempts {
		cost := attempt.AttemptCost()
		stats.TotalCost += cost

		// All attempts except the last one are "wasted" (preempted)
		if i < len(attempts)-1 {
			stats.WastedCost += cost
		}
	}

	if stats.IsPreemptible {
		stats.PreemptedCount = stats.TotalAttempts - 1

		// Calculate attempt-based efficiency score
		if stats.MaxPreemptible > 0 {
			stats.EfficiencyScore = 1.0 - (float64(stats.PreemptedCount) / float64(stats.MaxPreemptible))
		} else {
			stats.EfficiencyScore = 1.0 / float64(stats.TotalAttempts)
		}

		// Clamp to [0, 1]
		if stats.EfficiencyScore < 0 {
			stats.EfficiencyScore = 0
		}
		if stats.EfficiencyScore > 1 {
			stats.EfficiencyScore = 1
		}

		// Calculate cost-weighted efficiency
		if stats.TotalCost > 0 {
			stats.CostEfficiency = 1.0 - (stats.WastedCost / stats.TotalCost)
		} else {
			stats.CostEfficiency = 1.0
		}
	} else {
		stats.EfficiencyScore = 1.0
		stats.CostEfficiency = 1.0
	}

	return stats
}

// IsPreemptible checks if a task is configured as preemptible.
func IsPreemptible(value string) bool {
	if value == "" || value == "false" || value == "0" {
		return false
	}
	return true
}

// ParseMaxPreemptible extracts the max preemptible attempts from config.
func ParseMaxPreemptible(value string) int {
	if value == "" || value == "false" {
		return 0
	}
	if value == "true" {
		return 1
	}
	// Try to parse as number
	var n int
	for _, c := range value {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}

// shortTaskName extracts just the task name from "Workflow.Task" format.
func shortTaskName(fullName string) string {
	parts := strings.Split(fullName, ".")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return fullName
}
