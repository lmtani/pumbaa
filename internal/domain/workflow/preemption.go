// preemption.go contains preemption analysis types and logic for workflows.
// This includes efficiency analysis, statistics, and reporting for preemptible tasks.
package workflow

import (
	"sort"
	"strings"
)

// PreemptionTaskStats is a Value Object containing preemption statistics for a single task/shard.
// Value Objects are immutable and compared by value.
type PreemptionTaskStats struct {
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

// PreemptionProblematicTask is a Value Object representing an aggregated view
// of a task with poor preemption efficiency.
type PreemptionProblematicTask struct {
	Name             string  // Short task name (without workflow prefix)
	ShardCount       int     // Number of shards (1 for non-scattered)
	TotalAttempts    int     // Total attempts across all shards
	TotalPreemptions int     // Total preemptions across all shards
	EfficiencyScore  float64 // Average efficiency across all shards

	// Cost-weighted metrics
	TotalCost      float64 // Total cost across all shards (resource-hours)
	WastedCost     float64 // Total cost of failed attempts (resource-hours)
	CostEfficiency float64 // 1 - (WastedCost / TotalCost)
	ImpactPercent  float64 // WastedCost / WorkflowTotalWastedCost × 100
}

// PreemptionSummary is a Value Object containing aggregated preemption statistics for a workflow.
// It is computed by Workflow.CalculatePreemptionSummary() and represents a snapshot analysis.
type PreemptionSummary struct {
	WorkflowID        string
	WorkflowName      string
	TotalTasks        int                         // Total number of tasks/shards
	PreemptibleTasks  int                         // Number of preemptible tasks/shards
	TotalAttempts     int                         // Total attempts across all tasks
	TotalPreemptions  int                         // Total preemptions across all tasks
	OverallEfficiency float64                     // Average efficiency across all preemptible tasks
	ProblematicTasks  []PreemptionProblematicTask // Tasks with low efficiency (aggregated by task name)

	// Cost-weighted metrics
	TotalCost      float64 // Total cost of all attempts (resource-hours)
	WastedCost     float64 // Total cost of failed attempts (resource-hours)
	CostEfficiency float64 // 1 - (WastedCost / TotalCost)
	CostUnit       string  // Unit for cost display (e.g., "resource-hours" or "$")
}

// CalculatePreemptionSummary analyzes preemption statistics for this workflow and returns a summary.
func (w *Workflow) CalculatePreemptionSummary() *PreemptionSummary {
	summary := &PreemptionSummary{
		WorkflowID:       w.ID,
		WorkflowName:     w.Name,
		ProblematicTasks: []PreemptionProblematicTask{},
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
		totalCost        float64
		wastedCost       float64
	}
	taskAggregations := make(map[string]*taskAggregation)

	for callName, callList := range w.Calls {
		// Group by shard index to calculate per-shard stats
		shardGroups := make(map[int][]Call)
		for _, call := range callList {
			shardGroups[call.ShardIndex] = append(shardGroups[call.ShardIndex], call)
		}

		for shardIndex, attempts := range shardGroups {
			key := taskKey{name: callName, shard: shardIndex}
			if seen[key] {
				continue
			}
			seen[key] = true

			stats := analyzeTaskShard(attempts)
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
				taskName := preemptionShortTaskName(callName)
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
		if (costEfficiency < 0.7 || impactPercent > 10) && agg.totalPreemptions > 0 {
			summary.ProblematicTasks = append(summary.ProblematicTasks, PreemptionProblematicTask{
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
func analyzeTaskShard(attempts []Call) PreemptionTaskStats {
	stats := PreemptionTaskStats{}

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
	stats.FinalStatus = string(finalAttempt.Status)
	stats.IsPreemptible = IsPreemptible(finalAttempt.Preemptible)
	stats.MaxPreemptible = ParseMaxPreemptible(finalAttempt.Preemptible)

	// Calculate costs for all attempts
	for i, attempt := range attempts {
		cost := calculateAttemptCost(attempt)
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

// calculateAttemptCost calculates the estimated cost of a single attempt.
func calculateAttemptCost(call Call) float64 {
	// If we have actual VM cost per hour, use it
	if call.VMCostPerHour > 0 && !call.Start.IsZero() && !call.End.IsZero() {
		durationHours := call.End.Sub(call.Start).Hours()
		return call.VMCostPerHour * durationHours
	}

	// Otherwise, estimate using resource-hours
	cpu := parseCPUFromString(call.CPU)
	if cpu <= 0 {
		cpu = 1
	}
	mem := parseMemoryGBFromString(call.Memory)
	if mem <= 0 {
		mem = 1
	}

	var durationHours float64
	if !call.Start.IsZero() && !call.End.IsZero() {
		durationHours = call.End.Sub(call.Start).Hours()
	}
	if durationHours <= 0 {
		durationHours = 0.01 // Minimum 36 seconds
	}

	return cpu * mem * durationHours
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

// preemptionShortTaskName extracts just the task name from "Workflow.Task" format.
func preemptionShortTaskName(fullName string) string {
	parts := strings.Split(fullName, ".")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return fullName
}

// parseCPUFromString parses CPU string (e.g., "4", "4.0") to float64.
func parseCPUFromString(s string) float64 {
	if s == "" {
		return 0
	}
	var cpu float64
	for i, c := range s {
		if c == '.' {
			var decimal float64
			var divisor float64 = 10
			for _, d := range s[i+1:] {
				if d < '0' || d > '9' {
					break
				}
				decimal += float64(d-'0') / divisor
				divisor *= 10
			}
			return cpu + decimal
		}
		if c < '0' || c > '9' {
			break
		}
		cpu = cpu*10 + float64(c-'0')
	}
	return cpu
}

// parseMemoryGBFromString parses memory string (e.g., "8 GB", "8GB", "8192 MB") to GB.
func parseMemoryGBFromString(s string) float64 {
	if s == "" {
		return 0
	}

	var num float64
	var i int
	for i = 0; i < len(s); i++ {
		c := s[i]
		if c == '.' {
			var decimal float64
			var divisor float64 = 10
			for j := i + 1; j < len(s); j++ {
				d := s[j]
				if d < '0' || d > '9' {
					i = j
					break
				}
				decimal += float64(d-'0') / divisor
				divisor *= 10
				i = j + 1
			}
			num += decimal
			break
		}
		if c < '0' || c > '9' {
			break
		}
		num = num*10 + float64(c-'0')
	}

	rest := strings.ToUpper(strings.TrimSpace(s[i:]))
	if strings.HasPrefix(rest, "MB") {
		return num / 1024
	}
	if strings.HasPrefix(rest, "TB") {
		return num * 1024
	}
	return num
}
