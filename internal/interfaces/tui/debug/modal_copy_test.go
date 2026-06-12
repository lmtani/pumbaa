package debug

import (
	"testing"

	"github.com/lmtani/pumbaa/internal/domain/workflow"
)

func TestBuildCopyMenuItemsForTask(t *testing.T) {
	m := Model{metadata: &WorkflowMetadata{ID: "wf-id"}}
	node := &TreeNode{
		ID:   "wf.TaskA",
		Name: "TaskA",
		Type: NodeTypeCall,
		CallData: &workflow.Call{
			Stderr:      "gs://bucket/stderr.log",
			Stdout:      "gs://bucket/stdout.log",
			CallRoot:    "gs://bucket/call-TaskA",
			CommandLine: "echo hi",
			DockerImage: "ubuntu:22.04",
		},
	}

	items := m.buildCopyMenuItems(node)

	labels := make(map[string]string, len(items))
	for _, item := range items {
		labels[item.label] = item.value
	}

	for label, want := range map[string]string{
		"Stderr path":  "gs://bucket/stderr.log",
		"Stdout path":  "gs://bucket/stdout.log",
		"Call root":    "gs://bucket/call-TaskA",
		"Command line": "echo hi",
		"Docker image": "ubuntu:22.04",
		"Workflow ID":  "wf-id",
	} {
		if labels[label] != want {
			t.Errorf("item %q = %q, want %q", label, labels[label], want)
		}
	}

	// Empty fields (monitoring log, job ID) must be skipped
	if _, ok := labels["Monitoring log"]; ok {
		t.Error("empty monitoring log should not produce an item")
	}
	if _, ok := labels["Job ID"]; ok {
		t.Error("empty job ID should not produce an item")
	}
}

func TestBuildCopyMenuItemsForWorkflow(t *testing.T) {
	m := Model{metadata: &WorkflowMetadata{
		ID:              "wf-id",
		WorkflowRoot:    "gs://bucket/wf/",
		WorkflowLog:     "gs://bucket/wf.log",
		SubmittedInputs: `{"a":1}`,
	}}
	node := &TreeNode{ID: "wf-id", Name: "wf", Type: NodeTypeWorkflow}

	items := m.buildCopyMenuItems(node)

	if len(items) != 4 {
		t.Fatalf("got %d items, want 4: %v", len(items), items)
	}
	if items[0].label != "Workflow ID" || items[0].value != "wf-id" {
		t.Errorf("first item = %+v, want Workflow ID wf-id", items[0])
	}
}
