package preemption

import (
	"testing"
)

func TestAnalyzeWorkflow_NoPreemptibleTasks(t *testing.T) {
	analyzer := NewAnalyzer()

	calls := map[string][]CallData{
		"MyWorkflow.Task1": {
			{Name: "Task1", ShardIndex: -1, Attempt: 1, ExecutionStatus: "Done", Preemptible: "false"},
		},
	}

	stats := analyzer.AnalyzeWorkflow("wf-123", "MyWorkflow", calls)

	if stats.TotalTasks != 1 {
		t.Errorf("Expected 1 total task, got %d", stats.TotalTasks)
	}
	if stats.PreemptibleTasks != 0 {
		t.Errorf("Expected 0 preemptible tasks, got %d", stats.PreemptibleTasks)
	}
	if stats.OverallEfficiency != 1.0 {
		t.Errorf("Expected overall efficiency 1.0, got %f", stats.OverallEfficiency)
	}
}

func TestAnalyzeWorkflow_PreemptibleFirstTrySuccess(t *testing.T) {
	analyzer := NewAnalyzer()

	calls := map[string][]CallData{
		"MyWorkflow.Task1": {
			{Name: "Task1", ShardIndex: -1, Attempt: 1, ExecutionStatus: "Done", Preemptible: "3"},
		},
	}

	stats := analyzer.AnalyzeWorkflow("wf-123", "MyWorkflow", calls)

	if stats.TotalTasks != 1 {
		t.Errorf("Expected 1 total task, got %d", stats.TotalTasks)
	}
	if stats.PreemptibleTasks != 1 {
		t.Errorf("Expected 1 preemptible task, got %d", stats.PreemptibleTasks)
	}
	if stats.TotalPreemptions != 0 {
		t.Errorf("Expected 0 preemptions, got %d", stats.TotalPreemptions)
	}
	if stats.OverallEfficiency != 1.0 {
		t.Errorf("Expected overall efficiency 1.0, got %f", stats.OverallEfficiency)
	}
}

func TestAnalyzeWorkflow_PreemptibleWithRetries(t *testing.T) {
	analyzer := NewAnalyzer()

	// Task was preempted twice before succeeding
	calls := map[string][]CallData{
		"MyWorkflow.Task1": {
			{Name: "Task1", ShardIndex: -1, Attempt: 1, ExecutionStatus: "Preempted", Preemptible: "3"},
			{Name: "Task1", ShardIndex: -1, Attempt: 2, ExecutionStatus: "Preempted", Preemptible: "3"},
			{Name: "Task1", ShardIndex: -1, Attempt: 3, ExecutionStatus: "Done", Preemptible: "3"},
		},
	}

	stats := analyzer.AnalyzeWorkflow("wf-123", "MyWorkflow", calls)

	if stats.TotalTasks != 1 {
		t.Errorf("Expected 1 total task, got %d", stats.TotalTasks)
	}
	if stats.TotalAttempts != 3 {
		t.Errorf("Expected 3 total attempts, got %d", stats.TotalAttempts)
	}
	if stats.TotalPreemptions != 2 {
		t.Errorf("Expected 2 preemptions, got %d", stats.TotalPreemptions)
	}

	// Efficiency = 1 - (2 preemptions / 3 max) = 1 - 0.666 = 0.333
	expectedEfficiency := 1.0 - (2.0 / 3.0)
	if stats.OverallEfficiency < expectedEfficiency-0.01 || stats.OverallEfficiency > expectedEfficiency+0.01 {
		t.Errorf("Expected overall efficiency ~%.2f, got %f", expectedEfficiency, stats.OverallEfficiency)
	}
}

func TestAnalyzeWorkflow_MultipleTasks(t *testing.T) {
	analyzer := NewAnalyzer()

	calls := map[string][]CallData{
		"MyWorkflow.Task1": {
			{Name: "Task1", ShardIndex: -1, Attempt: 1, ExecutionStatus: "Done", Preemptible: "3"},
		},
		"MyWorkflow.Task2": {
			{Name: "Task2", ShardIndex: -1, Attempt: 1, ExecutionStatus: "Preempted", Preemptible: "3"},
			{Name: "Task2", ShardIndex: -1, Attempt: 2, ExecutionStatus: "Done", Preemptible: "3"},
		},
		"MyWorkflow.Task3": {
			{Name: "Task3", ShardIndex: -1, Attempt: 1, ExecutionStatus: "Done", Preemptible: "false"},
		},
	}

	stats := analyzer.AnalyzeWorkflow("wf-123", "MyWorkflow", calls)

	if stats.TotalTasks != 3 {
		t.Errorf("Expected 3 total tasks, got %d", stats.TotalTasks)
	}
	if stats.PreemptibleTasks != 2 {
		t.Errorf("Expected 2 preemptible tasks, got %d", stats.PreemptibleTasks)
	}
}

func TestAnalyzeWorkflow_ScatteredTask(t *testing.T) {
	analyzer := NewAnalyzer()

	// Scattered task with 3 shards, shard 1 was preempted once
	calls := map[string][]CallData{
		"MyWorkflow.Task1": {
			{Name: "Task1", ShardIndex: 0, Attempt: 1, ExecutionStatus: "Done", Preemptible: "3"},
			{Name: "Task1", ShardIndex: 1, Attempt: 1, ExecutionStatus: "Preempted", Preemptible: "3"},
			{Name: "Task1", ShardIndex: 1, Attempt: 2, ExecutionStatus: "Done", Preemptible: "3"},
			{Name: "Task1", ShardIndex: 2, Attempt: 1, ExecutionStatus: "Done", Preemptible: "3"},
		},
	}

	stats := analyzer.AnalyzeWorkflow("wf-123", "MyWorkflow", calls)

	// 3 shards = 3 "tasks" from preemption perspective
	if stats.TotalTasks != 3 {
		t.Errorf("Expected 3 total tasks (shards), got %d", stats.TotalTasks)
	}
	if stats.TotalAttempts != 4 {
		t.Errorf("Expected 4 total attempts, got %d", stats.TotalAttempts)
	}
	if stats.TotalPreemptions != 1 {
		t.Errorf("Expected 1 preemption, got %d", stats.TotalPreemptions)
	}
}

func TestAnalyzeWorkflow_ProblematicTasks(t *testing.T) {
	analyzer := NewAnalyzer()

	// Task with efficiency < 50%
	calls := map[string][]CallData{
		"MyWorkflow.BadTask": {
			{Name: "BadTask", ShardIndex: -1, Attempt: 1, ExecutionStatus: "Preempted", Preemptible: "3"},
			{Name: "BadTask", ShardIndex: -1, Attempt: 2, ExecutionStatus: "Preempted", Preemptible: "3"},
			{Name: "BadTask", ShardIndex: -1, Attempt: 3, ExecutionStatus: "Done", Preemptible: "3"},
		},
	}

	stats := analyzer.AnalyzeWorkflow("wf-123", "MyWorkflow", calls)

	if len(stats.ProblematicTasks) != 1 {
		t.Errorf("Expected 1 problematic task, got %d", len(stats.ProblematicTasks))
	}
	if len(stats.ProblematicTasks) > 0 && stats.ProblematicTasks[0].Name != "BadTask" {
		t.Errorf("Expected problematic task 'BadTask', got '%s'", stats.ProblematicTasks[0].Name)
	}
}

func TestIsPreemptible(t *testing.T) {
	tests := []struct {
		value    string
		expected bool
	}{
		{"", false},
		{"false", false},
		{"0", false},
		{"true", true},
		{"1", true},
		{"3", true},
		{"10", true},
	}

	for _, tt := range tests {
		result := IsPreemptible(tt.value)
		if result != tt.expected {
			t.Errorf("IsPreemptible(%q) = %v, expected %v", tt.value, result, tt.expected)
		}
	}
}

func TestParseMaxPreemptible(t *testing.T) {
	tests := []struct {
		value    string
		expected int
	}{
		{"", 0},
		{"false", 0},
		{"true", 1},
		{"0", 0},
		{"1", 1},
		{"3", 3},
		{"10", 10},
	}

	for _, tt := range tests {
		result := ParseMaxPreemptible(tt.value)
		if result != tt.expected {
			t.Errorf("ParseMaxPreemptible(%q) = %d, expected %d", tt.value, result, tt.expected)
		}
	}
}

func TestAttemptCost(t *testing.T) {
	tests := []struct {
		name     string
		call     CallData
		expected float64
	}{
		{
			name:     "basic cost calculation",
			call:     CallData{CPU: 4, MemoryGB: 8, DurationHours: 1},
			expected: 32.0, // 4 * 8 * 1 = 32
		},
		{
			name:     "with VM cost per hour",
			call:     CallData{CPU: 4, MemoryGB: 8, DurationHours: 2, VMCostPerHour: 0.50},
			expected: 1.0, // 0.50 * 2 = 1.0 (uses VM cost when available)
		},
		{
			name:     "defaults for missing values",
			call:     CallData{CPU: 0, MemoryGB: 0, DurationHours: 0},
			expected: 0.01, // 1 * 1 * 0.01 (minimum duration)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.call.AttemptCost()
			if result < tt.expected-0.001 || result > tt.expected+0.001 {
				t.Errorf("AttemptCost() = %f, expected %f", result, tt.expected)
			}
		})
	}
}

func TestAnalyzeWorkflow_CostWeightedEfficiency(t *testing.T) {
	analyzer := NewAnalyzer()

	// Cheap task that fails a lot vs expensive task that succeeds
	calls := map[string][]CallData{
		"MyWorkflow.CheapTask": {
			// Cheap task: 1 CPU, 1GB, 0.1h each attempt - fails twice
			{Name: "CheapTask", ShardIndex: -1, Attempt: 1, ExecutionStatus: "Preempted", Preemptible: "3", CPU: 1, MemoryGB: 1, DurationHours: 0.1},
			{Name: "CheapTask", ShardIndex: -1, Attempt: 2, ExecutionStatus: "Preempted", Preemptible: "3", CPU: 1, MemoryGB: 1, DurationHours: 0.1},
			{Name: "CheapTask", ShardIndex: -1, Attempt: 3, ExecutionStatus: "Done", Preemptible: "3", CPU: 1, MemoryGB: 1, DurationHours: 0.1},
		},
		"MyWorkflow.ExpensiveTask": {
			// Expensive task: 32 CPU, 64GB, 2h - succeeds first try
			{Name: "ExpensiveTask", ShardIndex: -1, Attempt: 1, ExecutionStatus: "Done", Preemptible: "3", CPU: 32, MemoryGB: 64, DurationHours: 2},
		},
	}

	stats := analyzer.AnalyzeWorkflow("wf-123", "MyWorkflow", calls)

	// CheapTask: total cost = 3 * (1 * 1 * 0.1) = 0.3, wasted = 0.2
	// ExpensiveTask: total cost = 32 * 64 * 2 = 4096, wasted = 0
	// Total cost = 4096.3, Wasted = 0.2
	// Cost efficiency = 1 - (0.2 / 4096.3) ≈ 99.995%

	if stats.TotalCost < 4000 {
		t.Errorf("Expected total cost > 4000, got %f", stats.TotalCost)
	}

	if stats.WastedCost < 0.1 || stats.WastedCost > 0.3 {
		t.Errorf("Expected wasted cost ~0.2, got %f", stats.WastedCost)
	}

	// Cost efficiency should be very high because the expensive task succeeded
	if stats.CostEfficiency < 0.99 {
		t.Errorf("Expected cost efficiency > 99%%, got %f", stats.CostEfficiency)
	}
}

func TestAnalyzeWorkflow_ProblematicTasksByImpact(t *testing.T) {
	analyzer := NewAnalyzer()

	// Two tasks that fail, but one has much higher cost impact
	calls := map[string][]CallData{
		"MyWorkflow.SmallBadTask": {
			// Small task fails - low cost impact
			{Name: "SmallBadTask", ShardIndex: -1, Attempt: 1, ExecutionStatus: "Preempted", Preemptible: "3", CPU: 1, MemoryGB: 1, DurationHours: 0.1},
			{Name: "SmallBadTask", ShardIndex: -1, Attempt: 2, ExecutionStatus: "Preempted", Preemptible: "3", CPU: 1, MemoryGB: 1, DurationHours: 0.1},
			{Name: "SmallBadTask", ShardIndex: -1, Attempt: 3, ExecutionStatus: "Done", Preemptible: "3", CPU: 1, MemoryGB: 1, DurationHours: 0.1},
		},
		"MyWorkflow.BigBadTask": {
			// Big task fails once - high cost impact
			{Name: "BigBadTask", ShardIndex: -1, Attempt: 1, ExecutionStatus: "Preempted", Preemptible: "3", CPU: 16, MemoryGB: 32, DurationHours: 1},
			{Name: "BigBadTask", ShardIndex: -1, Attempt: 2, ExecutionStatus: "Done", Preemptible: "3", CPU: 16, MemoryGB: 32, DurationHours: 1},
		},
	}

	stats := analyzer.AnalyzeWorkflow("wf-123", "MyWorkflow", calls)

	// Both should be problematic, but BigBadTask should be first (higher wasted cost)
	if len(stats.ProblematicTasks) < 1 {
		t.Fatalf("Expected at least 1 problematic task, got %d", len(stats.ProblematicTasks))
	}

	// First problematic task should be BigBadTask (sorted by wasted cost)
	if stats.ProblematicTasks[0].Name != "BigBadTask" {
		t.Errorf("Expected first problematic task to be 'BigBadTask' (highest cost impact), got '%s'", stats.ProblematicTasks[0].Name)
	}

	// Verify wasted cost for BigBadTask: 16 * 32 * 1 = 512
	if stats.ProblematicTasks[0].WastedCost < 500 {
		t.Errorf("Expected BigBadTask wasted cost ~512, got %f", stats.ProblematicTasks[0].WastedCost)
	}
}