package debuginfo

import (
	"testing"
	"time"
)

// TestShardGroupingWithMultipleAttempts tests that multiple attempts of the same shard
// are grouped into a single node with the correct status.
func TestShardGroupingWithMultipleAttempts(t *testing.T) {
	// Create test data with multiple attempts of the same shard
	wm := &WorkflowMetadata{
		ID:     "test-workflow",
		Name:   "TestWorkflow",
		Status: "Running",
		Start:  time.Now().Add(-1 * time.Hour),
		End:    time.Time{}, // Still running
		Calls: map[string][]CallDetails{
			"TestWorkflow.TaskWithRetries": {
				// Shard 0, attempt 1 - Failed (preempted)
				{
					Name:            "TaskWithRetries",
					ShardIndex:      0,
					Attempt:         1,
					ExecutionStatus: "Failed",
					Start:           time.Now().Add(-50 * time.Minute),
					End:             time.Now().Add(-45 * time.Minute),
				},
				// Shard 0, attempt 2 - Running
				{
					Name:            "TaskWithRetries",
					ShardIndex:      0,
					Attempt:         2,
					ExecutionStatus: "Running",
					Start:           time.Now().Add(-10 * time.Minute),
					End:             time.Time{}, // Still running
				},
				// Shard 1, attempt 1 - Done
				{
					Name:            "TaskWithRetries",
					ShardIndex:      1,
					Attempt:         1,
					ExecutionStatus: "Done",
					Start:           time.Now().Add(-30 * time.Minute),
					End:             time.Now().Add(-20 * time.Minute),
				},
				// Shard 2, attempt 1 - Done
				{
					Name:            "TaskWithRetries",
					ShardIndex:      2,
					Attempt:         1,
					ExecutionStatus: "Done",
					Start:           time.Now().Add(-30 * time.Minute),
					End:             time.Now().Add(-15 * time.Minute),
				},
				// Shard 2, attempt 2 - Running (retry after preemption)
				{
					Name:            "TaskWithRetries",
					ShardIndex:      2,
					Attempt:         2,
					ExecutionStatus: "Running",
					Start:           time.Now().Add(-5 * time.Minute),
					End:             time.Time{}, // Still running
				},
			},
		},
	}

	// Build the tree
	tree := BuildTree(wm)

	// Verify the tree structure
	if tree == nil {
		t.Fatal("Expected tree to be non-nil")
	}

	if len(tree.Children) != 1 {
		t.Fatalf("Expected 1 child (TaskWithRetries parent node), got %d", len(tree.Children))
	}

	taskNode := tree.Children[0]
	if taskNode.Name != "TaskWithRetries" {
		t.Errorf("Expected task name 'TaskWithRetries', got '%s'", taskNode.Name)
	}

	// Verify that we have exactly 3 shard nodes (0, 1, 2), not 5 (which would be if all attempts were separate)
	if len(taskNode.Children) != 3 {
		t.Fatalf("Expected 3 shard nodes (one per unique shard), got %d", len(taskNode.Children))
	}

	// Verify shard 0 (has 2 attempts, should show Running status from attempt 2)
	shard0 := taskNode.Children[0]
	if shard0.Status != "Running" {
		t.Errorf("Expected shard 0 status 'Running' (from attempt 2), got '%s'", shard0.Status)
	}
	if !contains(shard0.Name, "shard 0") {
		t.Errorf("Expected shard 0 name to contain 'shard 0', got '%s'", shard0.Name)
	}
	if !contains(shard0.Name, "attempt 2") {
		t.Errorf("Expected shard 0 name to show 'attempt 2', got '%s'", shard0.Name)
	}

	// Verify shard 1 (has 1 attempt, should show Done)
	shard1 := taskNode.Children[1]
	if shard1.Status != "Done" {
		t.Errorf("Expected shard 1 status 'Done', got '%s'", shard1.Status)
	}
	if !contains(shard1.Name, "shard 1") {
		t.Errorf("Expected shard 1 name to contain 'shard 1', got '%s'", shard1.Name)
	}
	if contains(shard1.Name, "attempt") {
		t.Errorf("Expected shard 1 name NOT to show attempt (only 1 attempt), got '%s'", shard1.Name)
	}

	// Verify shard 2 (has 2 attempts, should show Running status from attempt 2)
	shard2 := taskNode.Children[2]
	if shard2.Status != "Running" {
		t.Errorf("Expected shard 2 status 'Running' (from attempt 2, not Done from attempt 1), got '%s'", shard2.Status)
	}
	if !contains(shard2.Name, "shard 2") {
		t.Errorf("Expected shard 2 name to contain 'shard 2', got '%s'", shard2.Name)
	}
	if !contains(shard2.Name, "attempt 2") {
		t.Errorf("Expected shard 2 name to show 'attempt 2', got '%s'", shard2.Name)
	}
}

// TestAggregateStatusWithMultipleAttempts tests the AggregateStatus function
// to ensure it correctly prioritizes Running over Done when there are retries.
func TestAggregateStatusWithMultipleAttempts(t *testing.T) {
	tests := []struct {
		name     string
		calls    []CallDetails
		expected string
	}{
		{
			name: "One attempt Done",
			calls: []CallDetails{
				{ExecutionStatus: "Done"},
			},
			expected: "Done",
		},
		{
			name: "First attempt Failed, second attempt Running",
			calls: []CallDetails{
				{ExecutionStatus: "Failed", Attempt: 1},
				{ExecutionStatus: "Running", Attempt: 2},
			},
			expected: "Running",
		},
		{
			name: "First attempt Done, second attempt Running (retry after preemption)",
			calls: []CallDetails{
				{ExecutionStatus: "Done", Attempt: 1},
				{ExecutionStatus: "Running", Attempt: 2},
			},
			expected: "Running",
		},
		{
			name: "Multiple attempts, all Done",
			calls: []CallDetails{
				{ExecutionStatus: "Failed", Attempt: 1},
				{ExecutionStatus: "Done", Attempt: 2},
			},
			expected: "Done",
		},
		{
			name: "Multiple attempts, final one Failed",
			calls: []CallDetails{
				{ExecutionStatus: "Failed", Attempt: 1},
				{ExecutionStatus: "Failed", Attempt: 2},
			},
			expected: "Failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AggregateStatus(tt.calls)
			if result != tt.expected {
				t.Errorf("AggregateStatus() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
