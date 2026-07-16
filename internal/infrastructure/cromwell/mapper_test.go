package cromwell

import (
	"os"
	"testing"

	"github.com/lmtani/pumbaa/internal/domain/workflow"
)

func TestParseDetailedMetadata_ValidJSON(t *testing.T) {
	// Load test data
	data, err := os.ReadFile("../../../test_data/metadata.json")
	if err != nil {
		t.Skipf("Skipping test: test data not found: %v", err)
	}

	wf, err := ParseDetailedMetadata(data)
	if err != nil {
		t.Fatalf("ParseDetailedMetadata() error = %v", err)
	}

	// Verify basic fields are populated
	if wf.ID == "" {
		t.Error("Expected workflow ID to be set")
	}
	if wf.Name == "" {
		t.Error("Expected workflow Name to be set")
	}
	if wf.Status == "" {
		t.Error("Expected workflow Status to be set")
	}
}

func TestParseDetailedMetadata_InvalidJSON(t *testing.T) {
	invalidJSON := []byte(`{"invalid": json}`)

	_, err := ParseDetailedMetadata(invalidJSON)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestParseDetailedMetadata_EmptyJSON(t *testing.T) {
	emptyJSON := []byte(`{}`)

	wf, err := ParseDetailedMetadata(emptyJSON)
	if err != nil {
		t.Fatalf("ParseDetailedMetadata() unexpected error = %v", err)
	}

	if wf == nil {
		t.Error("Expected workflow to be created even for empty JSON")
	}
}

func TestMapMetadataResponseToWorkflow(t *testing.T) {
	// Create a mock metadata response
	m := &metadataResponse{
		ID:           "test-workflow-id",
		WorkflowName: "TestWorkflow",
		Status:       "Succeeded",
		Labels:       map[string]string{"env": "test"},
		Inputs:       map[string]any{"input1": "value1"},
		Outputs:      map[string]any{"output1": "result1"},
	}

	wf := mapMetadataResponseToWorkflow(m)

	if wf.ID != "test-workflow-id" {
		t.Errorf("ID = %q, want %q", wf.ID, "test-workflow-id")
	}
	if wf.Name != "TestWorkflow" {
		t.Errorf("Name = %q, want %q", wf.Name, "TestWorkflow")
	}
	if wf.Status != workflow.StatusSucceeded {
		t.Errorf("Status = %q, want %q", wf.Status, workflow.StatusSucceeded)
	}
	if wf.Labels["env"] != "test" {
		t.Errorf("Labels[env] = %q, want %q", wf.Labels["env"], "test")
	}
}

func TestMapMetadataResponseToWorkflow_WithCalls(t *testing.T) {
	m := &metadataResponse{
		ID:           "test-id",
		WorkflowName: "TestWorkflow",
		Status:       "Succeeded",
		Calls: map[string][]callMetadata{
			"Task1": {
				{
					ShardIndex:      -1,
					Attempt:         1,
					ExecutionStatus: "Done",
					Backend:         "PAPIv2",
					RuntimeAttributes: map[string]any{
						"preemptible": "3",
					},
				},
			},
		},
	}

	wf := mapMetadataResponseToWorkflow(m)

	if len(wf.Calls) != 1 {
		t.Fatalf("len(Calls) = %d, want 1", len(wf.Calls))
	}

	calls, ok := wf.Calls["Task1"]
	if !ok {
		t.Fatal("Expected Task1 in Calls")
	}

	if len(calls) != 1 {
		t.Fatalf("len(Task1 calls) = %d, want 1", len(calls))
	}

	call := calls[0]
	if call.Attempt != 1 {
		t.Errorf("Attempt = %d, want 1", call.Attempt)
	}
	if call.Backend != "PAPIv2" {
		t.Errorf("Backend = %q, want %q", call.Backend, "PAPIv2")
	}
	// Note: Preemptible is mapped from RuntimeAttributes in mapCallMetadataToCall
	if call.Preemptible != "3" {
		t.Errorf("Preemptible = %q, want %q", call.Preemptible, "3")
	}
}

func TestMapMetadataResponseToWorkflow_WithSubWorkflow(t *testing.T) {
	m := &metadataResponse{
		ID:           "parent-id",
		WorkflowName: "ParentWorkflow",
		Status:       "Succeeded",
		Calls: map[string][]callMetadata{
			"SubWorkflowCall": {
				{
					ShardIndex:    -1,
					Attempt:       1,
					SubWorkflowID: "sub-workflow-id",
					SubWorkflowMetadata: &metadataResponse{
						ID:           "sub-workflow-id",
						WorkflowName: "SubWorkflow",
						Status:       "Succeeded",
						Calls:        map[string][]callMetadata{},
					},
				},
			},
		},
	}

	wf := mapMetadataResponseToWorkflow(m)

	calls := wf.Calls["SubWorkflowCall"]
	if len(calls) != 1 {
		t.Fatalf("len(SubWorkflowCall) = %d, want 1", len(calls))
	}

	call := calls[0]
	if call.SubWorkflowID != "sub-workflow-id" {
		t.Errorf("SubWorkflowID = %q, want %q", call.SubWorkflowID, "sub-workflow-id")
	}
	if call.SubWorkflowMetadata == nil {
		t.Error("Expected SubWorkflowMetadata to be set")
	}
	if call.SubWorkflowMetadata.Name != "SubWorkflow" {
		t.Errorf("SubWorkflow.Name = %q, want %q", call.SubWorkflowMetadata.Name, "SubWorkflow")
	}
}

func TestMapMetadataResponseToWorkflow_WithFailures(t *testing.T) {
	m := &metadataResponse{
		ID:           "failed-id",
		WorkflowName: "FailedWorkflow",
		Status:       "Failed",
		Failures: []failureMetadata{
			{
				Message: "Task failed",
				CausedBy: []failureMetadata{
					{Message: "Out of memory"},
				},
			},
		},
	}

	wf := mapMetadataResponseToWorkflow(m)

	if len(wf.Failures) != 1 {
		t.Fatalf("len(Failures) = %d, want 1", len(wf.Failures))
	}

	failure := wf.Failures[0]
	if failure.Message != "Task failed" {
		t.Errorf("Failure.Message = %q, want %q", failure.Message, "Task failed")
	}
	if len(failure.CausedBy) != 1 {
		t.Fatalf("len(CausedBy) = %d, want 1", len(failure.CausedBy))
	}
	if failure.CausedBy[0].Message != "Out of memory" {
		t.Errorf("CausedBy[0].Message = %q, want %q", failure.CausedBy[0].Message, "Out of memory")
	}
}

func TestParseDockerSize(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  int64
	}{
		{"nil", nil, 0},
		{"float64", float64(1234567890), 1234567890},
		{"int64", int64(1234567890), 1234567890},
		{"int", int(1234567), 1234567},
		{"string", "1234567890", 1234567890},
		{"empty string", "", 0},
		{"invalid string", "abc", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseDockerSize(tt.input)
			if got != tt.want {
				t.Errorf("parseDockerSize(%v) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseCPU(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"", 0},
		{"1", 1},
		{"4", 4},
		{"4.0", 4},
		{"2.5", 2.5},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseCPU(tt.input)
			if got != tt.want {
				t.Errorf("parseCPU(%q) = %f, want %f", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseMemoryGB(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"", 0},
		{"8 GB", 8},
		{"8GB", 8},
		{"4096 MB", 4}, // 4096MB = 4GB
		{"16", 16},     // Assume GB if no unit
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseMemoryGB(tt.input)
			if got != tt.want {
				t.Errorf("parseMemoryGB(%q) = %f, want %f", tt.input, got, tt.want)
			}
		})
	}
}

// TestParseDetailedMetadata_RunningAttempt guards the metadata shape that
// produced the running-cost bug: an in-flight attempt has start/vmStartTime
// and a cost rate but no end timestamps yet. The mapper must carry all of it
// through so the domain can accrue cost for the open window.
func TestParseDetailedMetadata_RunningAttempt(t *testing.T) {
	jsonData := `{
		"id": "wf-run",
		"workflowName": "TestWF",
		"status": "Running",
		"calls": {
			"TestWF.Align": [{
				"executionStatus": "Running",
				"shardIndex": 0,
				"attempt": 1,
				"start": "2026-07-16T12:52:24.020Z",
				"vmStartTime": "2026-07-16T12:52:45.457Z",
				"vmCostPerHour": 0.204282,
				"runtimeAttributes": {"cpu": "12", "memory": "54 GB", "preemptible": "2"}
			}]
		}
	}`

	wf, err := ParseDetailedMetadata([]byte(jsonData))
	if err != nil {
		t.Fatalf("ParseDetailedMetadata() error = %v", err)
	}

	calls := wf.Calls["TestWF.Align"]
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	c := calls[0]

	if c.Status != workflow.StatusRunning {
		t.Errorf("Status = %v, want Running", c.Status)
	}
	if c.Start.IsZero() {
		t.Errorf("Start should be parsed")
	}
	if !c.End.IsZero() {
		t.Errorf("End must stay zero for a running attempt, got %v", c.End)
	}
	if c.VMStartTime.IsZero() {
		t.Errorf("VMStartTime should be parsed")
	}
	if !c.VMEndTime.IsZero() {
		t.Errorf("VMEndTime must stay zero for a running attempt, got %v", c.VMEndTime)
	}
	if c.VMCostPerHour != 0.204282 {
		t.Errorf("VMCostPerHour = %v, want 0.204282", c.VMCostPerHour)
	}
	if c.CPU != "12" || c.Memory != "54 GB" || c.Preemptible != "2" {
		t.Errorf("runtime attributes not extracted: cpu=%q mem=%q preempt=%q", c.CPU, c.Memory, c.Preemptible)
	}
}

// TestParseDetailedMetadata_QueuedAttemptWithoutTimestamps ensures attempts
// that have not started (queued in Cromwell) map cleanly with zero times.
func TestParseDetailedMetadata_QueuedAttemptWithoutTimestamps(t *testing.T) {
	jsonData := `{
		"id": "wf-q",
		"workflowName": "TestWF",
		"status": "Running",
		"calls": {
			"TestWF.Waiting": [{
				"executionStatus": "QueuedInCromwell",
				"shardIndex": -1,
				"attempt": 1
			}]
		}
	}`

	wf, err := ParseDetailedMetadata([]byte(jsonData))
	if err != nil {
		t.Fatalf("ParseDetailedMetadata() error = %v", err)
	}

	c := wf.Calls["TestWF.Waiting"][0]
	if !c.Start.IsZero() || !c.End.IsZero() || !c.VMStartTime.IsZero() || !c.VMEndTime.IsZero() {
		t.Errorf("all timestamps should be zero for a queued attempt: %+v", c)
	}
	if c.VMCostPerHour != 0 {
		t.Errorf("VMCostPerHour should be zero, got %v", c.VMCostPerHour)
	}
}

// TestParseDetailedMetadata_NestedSubworkflows exercises two levels of
// expanded subworkflow metadata, the shape the cost breakdown recurses over.
func TestParseDetailedMetadata_NestedSubworkflows(t *testing.T) {
	jsonData := `{
		"id": "wf-root",
		"workflowName": "Root",
		"status": "Running",
		"calls": {
			"Root.Outer": [{
				"executionStatus": "Running",
				"shardIndex": 0,
				"attempt": 1,
				"subWorkflowId": "sub-1",
				"subWorkflowMetadata": {
					"id": "sub-1",
					"workflowName": "Outer",
					"status": "Running",
					"calls": {
						"Outer.Inner": [{
							"executionStatus": "Running",
							"shardIndex": -1,
							"attempt": 1,
							"subWorkflowId": "sub-2",
							"subWorkflowMetadata": {
								"id": "sub-2",
								"workflowName": "Inner",
								"status": "Running",
								"calls": {
									"Inner.Leaf": [{
										"executionStatus": "Done",
										"shardIndex": -1,
										"attempt": 1,
										"start": "2026-07-16T10:00:00.000Z",
										"end": "2026-07-16T11:00:00.000Z",
										"vmStartTime": "2026-07-16T10:05:00.000Z",
										"vmEndTime": "2026-07-16T10:55:00.000Z",
										"vmCostPerHour": 0.5
									}]
								}
							}
						}]
					}
				}
			}]
		}
	}`

	wf, err := ParseDetailedMetadata([]byte(jsonData))
	if err != nil {
		t.Fatalf("ParseDetailedMetadata() error = %v", err)
	}

	outer := wf.Calls["Root.Outer"][0]
	if outer.SubWorkflowID != "sub-1" {
		t.Errorf("SubWorkflowID = %q, want sub-1", outer.SubWorkflowID)
	}
	if outer.SubWorkflowMetadata == nil {
		t.Fatal("first-level subworkflow metadata not mapped")
	}

	inner := outer.SubWorkflowMetadata.Calls["Outer.Inner"][0]
	if inner.SubWorkflowMetadata == nil {
		t.Fatal("second-level subworkflow metadata not mapped")
	}

	leaf := inner.SubWorkflowMetadata.Calls["Inner.Leaf"][0]
	if leaf.VMCostPerHour != 0.5 {
		t.Errorf("leaf VMCostPerHour = %v, want 0.5", leaf.VMCostPerHour)
	}
	if leaf.VMStartTime.IsZero() || leaf.VMEndTime.IsZero() {
		t.Errorf("leaf VM window should be fully parsed")
	}
	if got := leaf.VMEndTime.Sub(leaf.VMStartTime); got.Minutes() != 50 {
		t.Errorf("leaf VM window = %v, want 50m", got)
	}
}

// TestParseDetailedMetadata_NumericRuntimeAttributes covers backends that
// report cpu/preemptible as JSON numbers instead of strings.
func TestParseDetailedMetadata_NumericRuntimeAttributes(t *testing.T) {
	jsonData := `{
		"id": "wf-num",
		"workflowName": "TestWF",
		"status": "Succeeded",
		"calls": {
			"TestWF.Task": [{
				"executionStatus": "Done",
				"shardIndex": -1,
				"attempt": 1,
				"runtimeAttributes": {"cpu": 4, "preemptible": 3}
			}]
		}
	}`

	wf, err := ParseDetailedMetadata([]byte(jsonData))
	if err != nil {
		t.Fatalf("ParseDetailedMetadata() error = %v", err)
	}

	c := wf.Calls["TestWF.Task"][0]
	if c.CPU != "4" {
		t.Errorf("numeric cpu not normalized, got %q", c.CPU)
	}
	if c.Preemptible != "3" {
		t.Errorf("numeric preemptible not normalized, got %q", c.Preemptible)
	}
}
