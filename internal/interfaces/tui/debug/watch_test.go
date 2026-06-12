package debug

import (
	"strings"
	"testing"

	"github.com/lmtani/pumbaa/internal/domain/workflow"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/debug/tree"
)

func watchTestMetadata(taskAStatus workflow.Status) *WorkflowMetadata {
	return &WorkflowMetadata{
		ID:     "wf-id",
		Name:   "wf",
		Status: workflow.StatusRunning,
		Calls: map[string][]workflow.Call{
			"wf.TaskA": {
				{Name: "wf.TaskA", ShardIndex: -1, Attempt: 1, Status: taskAStatus},
			},
			"wf.Scatter": {
				{Name: "wf.Scatter", ShardIndex: 0, Attempt: 1, Status: workflow.StatusSucceeded},
				{Name: "wf.Scatter", ShardIndex: 1, Attempt: 1, Status: workflow.StatusRunning},
			},
		},
	}
}

func TestApplyRefreshedMetadataPreservesStateAndReportsChanges(t *testing.T) {
	old := watchTestMetadata(workflow.StatusRunning)
	root := tree.BuildTree(old)
	m := Model{metadata: old, tree: root, nodes: tree.GetVisibleNodes(root)}

	// Expand the scatter node and put the cursor on it
	var scatterIdx int
	for i, node := range m.nodes {
		if node.ID == "wf.Scatter" {
			node.Expanded = true
			scatterIdx = i
		}
	}
	m.nodes = tree.GetVisibleNodes(root)
	m.cursor = scatterIdx

	changes, cmds := m.applyRefreshedMetadata(watchTestMetadata(workflow.StatusFailed))

	if len(cmds) != 0 {
		t.Errorf("expected no subworkflow re-fetches, got %d", len(cmds))
	}
	if len(changes) != 1 || !strings.Contains(changes[0], "TaskA: Running → Failed") {
		t.Errorf("changes = %v, want TaskA Running → Failed", changes)
	}

	scatter := m.findNodeByID(m.tree, "wf.Scatter")
	if scatter == nil || !scatter.Expanded {
		t.Error("scatter expansion should be preserved across refresh")
	}
	if m.cursor >= len(m.nodes) || m.nodes[m.cursor].ID != "wf.Scatter" {
		t.Errorf("cursor should stay on wf.Scatter, got index %d", m.cursor)
	}
	if m.metadata.Calls["wf.TaskA"][0].Status != workflow.StatusFailed {
		t.Error("metadata should be swapped for the refreshed version")
	}
}

func TestWatchStatusMessage(t *testing.T) {
	if got := watchStatusMessage(nil); !strings.Contains(got, "no status changes") {
		t.Errorf("empty changes message = %q", got)
	}

	changes := []string{"A: Running → Done", "B: Running → Failed", "C: Submitted → Running"}
	got := watchStatusMessage(changes)
	if !strings.Contains(got, "A: Running → Done") || !strings.Contains(got, "(+1 more)") {
		t.Errorf("message = %q, want first two changes plus (+1 more)", got)
	}
}
