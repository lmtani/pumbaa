// Package workflow contains the domain entities and business logic for workflows.
package workflow

import (
	"sort"
	"strconv"
	"strings"
)

// TaskMetrics represents metrics of a task execution (Value Object).
// It contains both resource requests and actual usage data.
type TaskMetrics struct {
	TaskName             string
	ShardIndex           int
	WorkflowID           string
	CPURequest           string
	MemoryRequestBytes   int64
	DiskSizeRequestBytes int64
	DiskType             string
	TotalInputBytes      int64
	Inputs               map[string]int64
	DurationSeconds      float64
	CPUMean              float64
	MemoryPeakMB         float64
	DiskPeakBytes        int64
	Error                string
}

// HasExecutionError returns true if the error indicates a task execution failure.
// Monitoring errors (e.g., "monitoring.log exists but no content") return false
// because the task completed successfully but monitoring data was not collected.
func (m TaskMetrics) HasExecutionError() bool {
	if m.Error == "" {
		return false
	}
	// Monitoring log issues are not execution errors - the task completed
	// but we couldn't collect metrics (e.g., task completed too quickly)
	if strings.Contains(m.Error, "monitoring.log exists but no content") {
		return false
	}
	if strings.Contains(m.Error, "monitoring.log") && strings.Contains(m.Error, "notFound") {
		return false
	}
	return true
}

// CalculateResourceCost calculates a dimensionless cost based on resources and duration.
// The formula is: CPU * Memory(GB) * Disk(GB) * Duration(hours)
func (m TaskMetrics) CalculateResourceCost() float64 {
	cpuVal := 1.0
	if parsed, err := strconv.ParseFloat(m.CPURequest, 64); err == nil && parsed > 0 {
		cpuVal = parsed
	}
	memGB := float64(m.MemoryRequestBytes) / (1024 * 1024 * 1024)
	diskGB := float64(m.DiskSizeRequestBytes) / (1024 * 1024 * 1024)
	durationHours := m.DurationSeconds / 3600
	if durationHours > 0 {
		return cpuVal * memGB * diskGB * durationHours
	}
	return 0
}

// TaskMetricsCollection represents a collection of task metrics with domain behavior.
type TaskMetricsCollection struct {
	metrics []TaskMetrics
}

// NewTaskMetricsCollection creates a new collection from a slice of TaskMetrics.
func NewTaskMetricsCollection(metrics []TaskMetrics) *TaskMetricsCollection {
	return &TaskMetricsCollection{metrics: metrics}
}

// Metrics returns the underlying slice of TaskMetrics.
func (c *TaskMetricsCollection) Metrics() []TaskMetrics {
	return c.metrics
}

// Len returns the number of metrics in the collection.
func (c *TaskMetricsCollection) Len() int {
	return len(c.metrics)
}

// FilterByValidExecution returns a new collection with only metrics that don't have execution errors.
func (c *TaskMetricsCollection) FilterByValidExecution() *TaskMetricsCollection {
	var valid []TaskMetrics
	for _, m := range c.metrics {
		if !m.HasExecutionError() {
			valid = append(valid, m)
		}
	}
	return NewTaskMetricsCollection(valid)
}

// GroupByTaskName groups metrics by task name.
func (c *TaskMetricsCollection) GroupByTaskName() map[string]*TaskMetricsCollection {
	groups := make(map[string][]TaskMetrics)
	for _, m := range c.metrics {
		groups[m.TaskName] = append(groups[m.TaskName], m)
	}

	result := make(map[string]*TaskMetricsCollection)
	for name, metrics := range groups {
		result[name] = NewTaskMetricsCollection(metrics)
	}
	return result
}

// UniqueTaskNames returns the count of unique task names in the collection.
func (c *TaskMetricsCollection) UniqueTaskNames() int {
	names := make(map[string]bool)
	for _, m := range c.metrics {
		names[m.TaskName] = true
	}
	return len(names)
}

// TaskAggregatedMetrics represents aggregated metrics for a single task across multiple samples.
type TaskAggregatedMetrics struct {
	TaskName        string
	SampleCount     int
	CPURequest      string
	MemoryReqGB     float64
	DiskReqGB       float64
	DiskPeaksGB     []float64
	MemoryPeaksMB   []float64
	CPUMeans        []float64
	DurationSeconds []float64
	InputSizes      map[string][]int64
	ResourceCost    float64
}

// ToAggregatedMetrics converts the collection to aggregated metrics per task.
// It groups metrics by task name, calculates statistics, and sorts by resource cost.
func (c *TaskMetricsCollection) ToAggregatedMetrics() []TaskAggregatedMetrics {
	groups := c.GroupByTaskName()

	var result []TaskAggregatedMetrics
	for taskName, group := range groups {
		tasks := group.Metrics()
		if len(tasks) < 1 {
			continue
		}

		agg := TaskAggregatedMetrics{
			TaskName:    taskName,
			SampleCount: len(tasks),
			InputSizes:  make(map[string][]int64),
		}

		// Collect metrics per sample
		var totalCost float64
		for _, t := range tasks {
			agg.DiskPeaksGB = append(agg.DiskPeaksGB, float64(t.DiskPeakBytes)/(1024*1024*1024))
			agg.MemoryPeaksMB = append(agg.MemoryPeaksMB, t.MemoryPeakMB)
			agg.CPUMeans = append(agg.CPUMeans, t.CPUMean)
			agg.DurationSeconds = append(agg.DurationSeconds, t.DurationSeconds)

			// Collect input sizes
			for name, size := range t.Inputs {
				agg.InputSizes[name] = append(agg.InputSizes[name], size)
			}

			totalCost += t.CalculateResourceCost()
		}

		// Use first sample for resource requests (should be consistent across shards)
		first := tasks[0]
		agg.CPURequest = first.CPURequest
		agg.MemoryReqGB = float64(first.MemoryRequestBytes) / (1024 * 1024 * 1024)
		agg.DiskReqGB = float64(first.DiskSizeRequestBytes) / (1024 * 1024 * 1024)
		agg.ResourceCost = totalCost

		result = append(result, agg)
	}

	// Sort by resource cost (highest first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].ResourceCost > result[j].ResourceCost
	})

	return result
}

// EfficiencyStatus represents the efficiency level of a resource.
type EfficiencyStatus string

const (
	EfficiencyGood     EfficiencyStatus = "good"
	EfficiencyWarning  EfficiencyStatus = "warning"
	EfficiencyCritical EfficiencyStatus = "critical"
)

// TaskEfficiencyStats contains basic efficiency statistics for a task.
type TaskEfficiencyStats struct {
	TaskName      string
	SampleCount   int
	OverallStatus EfficiencyStatus
	ResourceCost  float64
}

// CalculateEfficiencyStats generates basic efficiency statistics for each task group.
// This is used when LLM recommendations are disabled.
func (c *TaskMetricsCollection) CalculateEfficiencyStats() []TaskEfficiencyStats {
	aggregated := c.ToAggregatedMetrics()
	var stats []TaskEfficiencyStats

	for _, task := range aggregated {
		// Calculate average metrics
		var avgCPU, avgMemPeak, avgDiskPeak float64
		for i := range task.CPUMeans {
			avgCPU += task.CPUMeans[i]
			avgMemPeak += task.MemoryPeaksMB[i]
			avgDiskPeak += task.DiskPeaksGB[i]
		}
		n := float64(len(task.CPUMeans))
		if n > 0 {
			avgCPU /= n
			avgMemPeak /= n
			avgDiskPeak /= n
		}

		// Calculate efficiency percentages
		memEfficiency := 0.0
		if task.MemoryReqGB > 0 {
			memEfficiency = (avgMemPeak / 1024) / task.MemoryReqGB * 100 // Convert MB to GB
		}
		diskEfficiency := 0.0
		if task.DiskReqGB > 0 {
			diskEfficiency = avgDiskPeak / task.DiskReqGB * 100
		}

		// Determine overall status based on efficiency
		overallStatus := EfficiencyGood
		if avgCPU < 30 || memEfficiency < 30 || diskEfficiency < 30 {
			overallStatus = EfficiencyCritical
		} else if avgCPU < 50 || memEfficiency < 50 || diskEfficiency < 50 {
			overallStatus = EfficiencyWarning
		}

		stats = append(stats, TaskEfficiencyStats{
			TaskName:      task.TaskName,
			SampleCount:   task.SampleCount,
			OverallStatus: overallStatus,
			ResourceCost:  task.ResourceCost,
		})
	}

	return stats
}
