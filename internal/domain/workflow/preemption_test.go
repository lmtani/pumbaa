package workflow

import (
	"testing"
	"time"
)

// Helper function to create a test Call with preemption settings
func createTestCall(name string, shardIndex, attempt int, preemptible string, status Status, durationMinutes int) Call {
	start := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	end := start.Add(time.Duration(durationMinutes) * time.Minute)
	return Call{
		Name:          name,
		ShardIndex:    shardIndex,
		Attempt:       attempt,
		Preemptible:   preemptible,
		Status:        status,
		Start:         start,
		End:           end,
		VMCostPerHour: 0.1, // $0.10/hour for easy calculation
	}
}

func TestCalculatePreemptionSummary_EmptyWorkflow(t *testing.T) {
	wf := &Workflow{
		ID:    "test-id",
		Name:  "TestWorkflow",
		Calls: map[string][]Call{},
	}

	summary := wf.CalculatePreemptionSummary()

	if summary.WorkflowID != "test-id" {
		t.Errorf("WorkflowID = %q, want %q", summary.WorkflowID, "test-id")
	}
	if summary.TotalTasks != 0 {
		t.Errorf("TotalTasks = %d, want 0", summary.TotalTasks)
	}
	if summary.CostEfficiency != 1.0 {
		t.Errorf("CostEfficiency = %f, want 1.0", summary.CostEfficiency)
	}
}

func TestCalculatePreemptionSummary_WithPreemptibleTasks(t *testing.T) {
	tests := []struct {
		name                 string
		calls                map[string][]Call
		wantTotalTasks       int
		wantPreemptibleTasks int
		wantTotalAttempts    int
		wantTotalPreemptions int
	}{
		{
			name: "single task no preemption",
			calls: map[string][]Call{
				"Task1": {
					createTestCall("Task1", -1, 1, "3", StatusSucceeded, 60),
				},
			},
			wantTotalTasks:       1,
			wantPreemptibleTasks: 1,
			wantTotalAttempts:    1,
			wantTotalPreemptions: 0,
		},
		{
			name: "single task with one preemption",
			calls: map[string][]Call{
				"Task1": {
					createTestCall("Task1", -1, 1, "3", StatusFailed, 30),
					createTestCall("Task1", -1, 2, "3", StatusSucceeded, 60),
				},
			},
			wantTotalTasks:       1,
			wantPreemptibleTasks: 1,
			wantTotalAttempts:    2,
			wantTotalPreemptions: 1,
		},
		{
			name: "scattered task with preemptions",
			calls: map[string][]Call{
				"ScatteredTask": {
					// Shard 0: succeeded first try
					createTestCall("ScatteredTask", 0, 1, "3", StatusSucceeded, 60),
					// Shard 1: preempted once
					createTestCall("ScatteredTask", 1, 1, "3", StatusFailed, 30),
					createTestCall("ScatteredTask", 1, 2, "3", StatusSucceeded, 60),
				},
			},
			wantTotalTasks:       2, // 2 shards
			wantPreemptibleTasks: 2,
			wantTotalAttempts:    3,
			wantTotalPreemptions: 1,
		},
		{
			name: "non-preemptible task",
			calls: map[string][]Call{
				"NonPreemptible": {
					createTestCall("NonPreemptible", -1, 1, "false", StatusSucceeded, 60),
				},
			},
			wantTotalTasks:       1,
			wantPreemptibleTasks: 0,
			wantTotalAttempts:    0,
			wantTotalPreemptions: 0,
		},
		{
			name: "mixed preemptible and non-preemptible",
			calls: map[string][]Call{
				"PreemptibleTask": {
					createTestCall("PreemptibleTask", -1, 1, "3", StatusFailed, 30),
					createTestCall("PreemptibleTask", -1, 2, "3", StatusSucceeded, 60),
				},
				"NonPreemptibleTask": {
					createTestCall("NonPreemptibleTask", -1, 1, "false", StatusSucceeded, 120),
				},
			},
			wantTotalTasks:       2,
			wantPreemptibleTasks: 1,
			wantTotalAttempts:    2,
			wantTotalPreemptions: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf := &Workflow{
				ID:    "test-id",
				Name:  "TestWorkflow",
				Calls: tt.calls,
			}

			summary := wf.CalculatePreemptionSummary()

			if summary.TotalTasks != tt.wantTotalTasks {
				t.Errorf("TotalTasks = %d, want %d", summary.TotalTasks, tt.wantTotalTasks)
			}
			if summary.PreemptibleTasks != tt.wantPreemptibleTasks {
				t.Errorf("PreemptibleTasks = %d, want %d", summary.PreemptibleTasks, tt.wantPreemptibleTasks)
			}
			if summary.TotalAttempts != tt.wantTotalAttempts {
				t.Errorf("TotalAttempts = %d, want %d", summary.TotalAttempts, tt.wantTotalAttempts)
			}
			if summary.TotalPreemptions != tt.wantTotalPreemptions {
				t.Errorf("TotalPreemptions = %d, want %d", summary.TotalPreemptions, tt.wantTotalPreemptions)
			}
		})
	}
}

func TestCalculatePreemptionSummary_CostCalculation(t *testing.T) {
	// Task with 2 attempts: first 30 min (wasted), second 60 min (successful)
	// VMCostPerHour = $0.10
	// Total cost = 0.5h * $0.10 + 1h * $0.10 = $0.15
	// Wasted cost = 0.5h * $0.10 = $0.05
	// Cost efficiency = 1 - (0.05 / 0.15) = 0.6667
	wf := &Workflow{
		ID:   "test-id",
		Name: "TestWorkflow",
		Calls: map[string][]Call{
			"Task1": {
				createTestCall("Task1", -1, 1, "3", StatusFailed, 30),
				createTestCall("Task1", -1, 2, "3", StatusSucceeded, 60),
			},
		},
	}

	summary := wf.CalculatePreemptionSummary()

	// Allow small floating point tolerance
	const tolerance = 0.001

	expectedTotalCost := 0.15 // 1.5 hours * $0.10/hour
	if diff := summary.TotalCost - expectedTotalCost; diff < -tolerance || diff > tolerance {
		t.Errorf("TotalCost = %f, want %f", summary.TotalCost, expectedTotalCost)
	}

	expectedWastedCost := 0.05 // 0.5 hours * $0.10/hour
	if diff := summary.WastedCost - expectedWastedCost; diff < -tolerance || diff > tolerance {
		t.Errorf("WastedCost = %f, want %f", summary.WastedCost, expectedWastedCost)
	}

	expectedEfficiency := 0.6667
	if diff := summary.CostEfficiency - expectedEfficiency; diff < -tolerance || diff > tolerance {
		t.Errorf("CostEfficiency = %f, want ~%f", summary.CostEfficiency, expectedEfficiency)
	}
}

func TestCalculatePreemptionSummary_ProblematicTasks(t *testing.T) {
	// Create a workflow with one very inefficient task
	wf := &Workflow{
		ID:   "test-id",
		Name: "TestWorkflow",
		Calls: map[string][]Call{
			"Workflow.IneffientTask": {
				// Multiple preemptions
				createTestCall("Workflow.IneffientTask", -1, 1, "5", StatusFailed, 30),
				createTestCall("Workflow.IneffientTask", -1, 2, "5", StatusFailed, 30),
				createTestCall("Workflow.IneffientTask", -1, 3, "5", StatusFailed, 30),
				createTestCall("Workflow.IneffientTask", -1, 4, "5", StatusSucceeded, 60),
			},
		},
	}

	summary := wf.CalculatePreemptionSummary()

	if summary.TotalPreemptions != 3 {
		t.Errorf("TotalPreemptions = %d, want 3", summary.TotalPreemptions)
	}

	// Should identify as problematic (>70% waste or >10% impact)
	if len(summary.ProblematicTasks) == 0 {
		t.Error("Expected at least one problematic task")
	}

	if len(summary.ProblematicTasks) > 0 {
		pt := summary.ProblematicTasks[0]
		if pt.Name != "IneffientTask" {
			t.Errorf("ProblematicTask.Name = %q, want %q", pt.Name, "IneffientTask")
		}
		if pt.TotalPreemptions != 3 {
			t.Errorf("ProblematicTask.TotalPreemptions = %d, want 3", pt.TotalPreemptions)
		}
	}
}

func TestIsPreemptible(t *testing.T) {
	tests := []struct {
		value string
		want  bool
	}{
		{"", false},
		{"false", false},
		{"0", false},
		{"true", true},
		{"1", true},
		{"3", true},
		{"5", true},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			got := IsPreemptible(tt.value)
			if got != tt.want {
				t.Errorf("IsPreemptible(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestParseMaxPreemptible(t *testing.T) {
	tests := []struct {
		value string
		want  int
	}{
		{"", 0},
		{"false", 0},
		{"true", 1},
		{"0", 0},
		{"1", 1},
		{"3", 3},
		{"10", 10},
		{"abc", 0},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			got := ParseMaxPreemptible(tt.value)
			if got != tt.want {
				t.Errorf("ParseMaxPreemptible(%q) = %d, want %d", tt.value, got, tt.want)
			}
		})
	}
}

func TestParseCPUFromString(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"", 0},
		{"1", 1},
		{"4.0", 4.0},
		{"4.5", 4.5},
		{"0.5", 0.5},
		{"invalid", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := parseCPUFromString(tt.input); got != tt.expected {
				t.Errorf("parseCPUFromString(%s) = %f, want %f", tt.input, got, tt.expected)
			}
		})
	}
}

func TestParseMemoryGBFromString(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"", 0},
		{"8", 8},
		{"8GB", 8},
		{"8 GB", 8},
		{"8192 MB", 8},
		{"1 TB", 1024},
		{"0.5 GB", 0.5},
		{"invalid", 0},
		{"10.5 GB", 10.5},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := parseMemoryGBFromString(tt.input); got != tt.expected {
				t.Errorf("parseMemoryGBFromString(%s) = %f, want %f", tt.input, got, tt.expected)
			}
		})
	}
}
