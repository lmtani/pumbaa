package workflow

import (
	"testing"
)

func TestTaskMetrics_HasExecutionError(t *testing.T) {
	tests := []struct {
		name     string
		error    string
		expected bool
	}{
		{
			name:     "empty error",
			error:    "",
			expected: false,
		},
		{
			name:     "monitoring log no content",
			error:    "monitoring.log exists but no content",
			expected: false,
		},
		{
			name:     "monitoring log not found",
			error:    "monitoring.log notFound",
			expected: false,
		},
		{
			name:     "actual execution error",
			error:    "task failed with exit code 1",
			expected: true,
		},
		{
			name:     "OOM error",
			error:    "out of memory",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := TaskMetrics{Error: tt.error}
			if got := m.HasExecutionError(); got != tt.expected {
				t.Errorf("HasExecutionError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTaskMetrics_CalculateResourceCost(t *testing.T) {
	tests := []struct {
		name     string
		metrics  TaskMetrics
		expected float64
	}{
		{
			name: "standard cost calculation",
			metrics: TaskMetrics{
				CPURequest:           "2",
				MemoryRequestBytes:   1024 * 1024 * 1024,      // 1 GB
				DiskSizeRequestBytes: 10 * 1024 * 1024 * 1024, // 10 GB
				DurationSeconds:      3600,                    // 1 hour
			},
			expected: 2 * 1 * 10 * 1, // CPU * Mem(GB) * Disk(GB) * Duration(hours) = 20
		},
		{
			name: "zero duration",
			metrics: TaskMetrics{
				CPURequest:           "4",
				MemoryRequestBytes:   2 * 1024 * 1024 * 1024,
				DiskSizeRequestBytes: 20 * 1024 * 1024 * 1024,
				DurationSeconds:      0,
			},
			expected: 0,
		},
		{
			name: "invalid CPU request defaults to 1",
			metrics: TaskMetrics{
				CPURequest:           "invalid",
				MemoryRequestBytes:   1024 * 1024 * 1024,
				DiskSizeRequestBytes: 10 * 1024 * 1024 * 1024,
				DurationSeconds:      3600,
			},
			expected: 1 * 1 * 10 * 1, // defaults to CPU=1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.metrics.CalculateResourceCost()
			if got != tt.expected {
				t.Errorf("CalculateResourceCost() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTaskMetricsCollection_FilterByValidExecution(t *testing.T) {
	collection := NewTaskMetricsCollection([]TaskMetrics{
		{TaskName: "task1", Error: ""},
		{TaskName: "task2", Error: "monitoring.log exists but no content"},
		{TaskName: "task3", Error: "execution failed"},
		{TaskName: "task4", Error: ""},
	})

	filtered := collection.FilterByValidExecution()

	if filtered.Len() != 3 {
		t.Errorf("FilterByValidExecution() returned %d items, want 3", filtered.Len())
	}

	// Check that task3 with execution error is filtered out
	for _, m := range filtered.Metrics() {
		if m.TaskName == "task3" {
			t.Error("FilterByValidExecution() should have filtered out task3")
		}
	}
}

func TestTaskMetricsCollection_GroupByTaskName(t *testing.T) {
	collection := NewTaskMetricsCollection([]TaskMetrics{
		{TaskName: "task1", ShardIndex: 0},
		{TaskName: "task1", ShardIndex: 1},
		{TaskName: "task2", ShardIndex: 0},
	})

	groups := collection.GroupByTaskName()

	if len(groups) != 2 {
		t.Errorf("GroupByTaskName() returned %d groups, want 2", len(groups))
	}

	if groups["task1"].Len() != 2 {
		t.Errorf("task1 group has %d items, want 2", groups["task1"].Len())
	}

	if groups["task2"].Len() != 1 {
		t.Errorf("task2 group has %d items, want 1", groups["task2"].Len())
	}
}

func TestTaskMetricsCollection_UniqueTaskNames(t *testing.T) {
	collection := NewTaskMetricsCollection([]TaskMetrics{
		{TaskName: "task1"},
		{TaskName: "task1"},
		{TaskName: "task2"},
		{TaskName: "task3"},
	})

	count := collection.UniqueTaskNames()
	if count != 3 {
		t.Errorf("UniqueTaskNames() = %d, want 3", count)
	}
}

func TestTaskMetricsCollection_ToAggregatedMetrics(t *testing.T) {
	collection := NewTaskMetricsCollection([]TaskMetrics{
		{
			TaskName:             "task1",
			CPURequest:           "2",
			MemoryRequestBytes:   1024 * 1024 * 1024,
			DiskSizeRequestBytes: 10 * 1024 * 1024 * 1024,
			DurationSeconds:      3600,
			CPUMean:              50.0,
			MemoryPeakMB:         512,
			DiskPeakBytes:        5 * 1024 * 1024 * 1024,
		},
		{
			TaskName:             "task1",
			CPURequest:           "2",
			MemoryRequestBytes:   1024 * 1024 * 1024,
			DiskSizeRequestBytes: 10 * 1024 * 1024 * 1024,
			DurationSeconds:      7200,
			CPUMean:              60.0,
			MemoryPeakMB:         600,
			DiskPeakBytes:        6 * 1024 * 1024 * 1024,
		},
	})

	aggregated := collection.ToAggregatedMetrics()

	if len(aggregated) != 1 {
		t.Fatalf("ToAggregatedMetrics() returned %d items, want 1", len(aggregated))
	}

	agg := aggregated[0]
	if agg.TaskName != "task1" {
		t.Errorf("TaskName = %s, want task1", agg.TaskName)
	}
	if agg.SampleCount != 2 {
		t.Errorf("SampleCount = %d, want 2", agg.SampleCount)
	}
	if len(agg.CPUMeans) != 2 {
		t.Errorf("CPUMeans has %d items, want 2", len(agg.CPUMeans))
	}
}

func TestTaskMetricsCollection_CalculateEfficiencyStats(t *testing.T) {
	collection := NewTaskMetricsCollection([]TaskMetrics{
		{
			TaskName:             "efficient_task",
			CPURequest:           "2",
			MemoryRequestBytes:   1024 * 1024 * 1024,      // 1 GB
			DiskSizeRequestBytes: 10 * 1024 * 1024 * 1024, // 10 GB
			DurationSeconds:      3600,
			CPUMean:              80.0,                   // 80% CPU usage
			MemoryPeakMB:         800,                    // 800 MB of 1 GB = 78%
			DiskPeakBytes:        8 * 1024 * 1024 * 1024, // 8 GB of 10 GB = 80%
		},
		{
			TaskName:             "inefficient_task",
			CPURequest:           "4",
			MemoryRequestBytes:   4 * 1024 * 1024 * 1024,   // 4 GB
			DiskSizeRequestBytes: 100 * 1024 * 1024 * 1024, // 100 GB
			DurationSeconds:      3600,
			CPUMean:              10.0,                   // 10% CPU usage
			MemoryPeakMB:         100,                    // 100 MB of 4 GB = 2.4%
			DiskPeakBytes:        1 * 1024 * 1024 * 1024, // 1 GB of 100 GB = 1%
		},
	})

	stats := collection.CalculateEfficiencyStats()

	if len(stats) != 2 {
		t.Fatalf("CalculateEfficiencyStats() returned %d items, want 2", len(stats))
	}

	// Find each task's stats
	var efficientStats, inefficientStats *TaskEfficiencyStats
	for i := range stats {
		switch stats[i].TaskName {
		case "efficient_task":
			efficientStats = &stats[i]
		case "inefficient_task":
			inefficientStats = &stats[i]
		}
	}

	if efficientStats == nil || inefficientStats == nil {
		t.Fatal("Could not find both tasks in stats")
	}

	if efficientStats.OverallStatus != EfficiencyGood {
		t.Errorf("efficient_task status = %s, want %s", efficientStats.OverallStatus, EfficiencyGood)
	}

	if inefficientStats.OverallStatus != EfficiencyCritical {
		t.Errorf("inefficient_task status = %s, want %s", inefficientStats.OverallStatus, EfficiencyCritical)
	}
}
