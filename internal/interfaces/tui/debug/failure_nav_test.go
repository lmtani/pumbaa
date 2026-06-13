package debug

import (
	"testing"

	"github.com/lmtani/pumbaa/internal/domain/workflow"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/debug/tree"
)

// buildFailureTestTree builds:
//
//	root (Failed)
//	├─ TaskA (Done)
//	├─ Scatter (Failed)
//	│  ├─ shard 0 (Done)
//	│  ├─ shard 1 (Failed)
//	│  └─ shard 2 (Failed)
//	└─ Sub (Failed, unloaded subworkflow)
func buildFailureTestTree() *TreeNode {
	root := &TreeNode{ID: "root", Name: "wf", Type: NodeTypeWorkflow, Status: "Failed", Expanded: true}

	taskA := &TreeNode{ID: "wf.TaskA", Name: "TaskA", Type: NodeTypeCall, Status: "Done", Parent: root}

	scatter := &TreeNode{ID: "wf.Scatter", Name: "Scatter", Type: NodeTypeCall, Status: "Failed", Parent: root}
	for i, status := range []string{"Done", "Failed", "Failed"} {
		scatter.Children = append(scatter.Children, &TreeNode{
			ID:     "wf.Scatter_" + string(rune('0'+i)),
			Name:   "Scatter",
			Type:   NodeTypeShard,
			Status: status,
			Parent: scatter,
		})
	}

	sub := &TreeNode{
		ID:            "wf.Sub",
		Name:          "Sub",
		Type:          NodeTypeSubWorkflow,
		Status:        "Failed",
		Parent:        root,
		SubWorkflowID: "sub-id",
	}

	root.Children = []*TreeNode{taskA, scatter, sub}
	return root
}

func TestCountFailedLeaves(t *testing.T) {
	root := buildFailureTestTree()

	// 2 failed shards + 1 unloaded failed subworkflow
	if got := countFailedLeaves(root); got != 3 {
		t.Errorf("countFailedLeaves(root) = %d, want 3", got)
	}

	scatter := root.Children[1]
	if got := countFailedLeaves(scatter); got != 2 {
		t.Errorf("countFailedLeaves(scatter) = %d, want 2", got)
	}

	taskA := root.Children[0]
	if got := countFailedLeaves(taskA); got != 0 {
		t.Errorf("countFailedLeaves(taskA) = %d, want 0", got)
	}
}

func TestExpandFailurePaths(t *testing.T) {
	root := buildFailureTestTree()
	scatter := root.Children[1]
	scatter.Expanded = false

	hasFailure, unloaded := expandFailurePaths(root)

	if !hasFailure {
		t.Error("expandFailurePaths should report failures")
	}
	if !root.Expanded {
		t.Error("root should stay expanded (failure path)")
	}
	if !scatter.Expanded {
		t.Error("scatter with failed shards should be expanded")
	}
	if len(unloaded) != 1 || unloaded[0].ID != "wf.Sub" {
		t.Errorf("unloaded = %v, want [wf.Sub]", unloaded)
	}
}

func TestExpandFailurePathsIncludesPreemptedTasks(t *testing.T) {
	// Succeeded workflow whose only "problem" is a task that recovered on
	// attempt 2 after a preemption (aggregate status Done, attempt > 1).
	root := &TreeNode{ID: "root", Name: "wf", Type: NodeTypeWorkflow, Status: "Succeeded", Expanded: true}
	sub := &TreeNode{ID: "wf.Align", Name: "Align", Type: NodeTypeCall, Status: "Done", Parent: root}
	preempted := &TreeNode{
		ID:       "wf.Align.MarkDup_0",
		Name:     "MarkDup [shard 0] (attempt 2)",
		Type:     NodeTypeShard,
		Status:   "Done",
		Parent:   sub,
		CallData: &workflow.Call{Attempt: 2, Status: workflow.StatusSucceeded},
	}
	clean := &TreeNode{
		ID:       "wf.Align.Sort_0",
		Name:     "Sort [shard 0]",
		Type:     NodeTypeShard,
		Status:   "Done",
		Parent:   sub,
		CallData: &workflow.Call{Attempt: 1, Status: workflow.StatusSucceeded},
	}
	sub.Children = []*TreeNode{preempted, clean}
	root.Children = []*TreeNode{sub}

	hasProblem, _ := expandFailurePaths(root)

	if !hasProblem {
		t.Error("expandFailurePaths should flag the preempted task as a problem")
	}
	if !sub.Expanded {
		t.Error("parent of a preempted task should be expanded")
	}
	if got := countRetriedLeaves(root); got != 1 {
		t.Errorf("countRetriedLeaves = %d, want 1", got)
	}
	if got := countFailedLeaves(root); got != 0 {
		t.Errorf("countFailedLeaves = %d, want 0", got)
	}
}

func TestExpandSummaryMessage(t *testing.T) {
	if got := expandSummaryMessage(2, 0); got != "2 failed task(s)" {
		t.Errorf("message = %q", got)
	}
	if got := expandSummaryMessage(0, 3); got != "3 preempted/retried task(s)" {
		t.Errorf("message = %q", got)
	}
	if got := expandSummaryMessage(1, 2); got != "1 failed, 2 preempted/retried task(s)" {
		t.Errorf("message = %q", got)
	}
}

func TestCountPreemptionsCountsRecoveredRetries(t *testing.T) {
	parent := &TreeNode{ID: "wf.Task", Name: "Task", Type: NodeTypeCall, Status: "Done"}
	recovered := &TreeNode{
		ID:       "wf.Task_0",
		Name:     "Task [shard 0] (attempt 2)",
		Type:     NodeTypeShard,
		Status:   "Done",
		Parent:   parent,
		CallData: &workflow.Call{Attempt: 2, Status: workflow.StatusSucceeded},
	}
	parent.Children = []*TreeNode{recovered}

	if got := countPreemptions(parent); got != 1 {
		t.Errorf("countPreemptions = %d, want 1 (recovered preemption)", got)
	}
}

func TestExpandFailurePathsCollapsesSuccessBranches(t *testing.T) {
	root := buildFailureTestTree()
	// Add an expanded all-Done scatter that must be collapsed
	okScatter := &TreeNode{ID: "wf.Ok", Name: "Ok", Type: NodeTypeCall, Status: "Done", Parent: root, Expanded: true}
	okScatter.Children = []*TreeNode{
		{ID: "wf.Ok_0", Name: "Ok", Type: NodeTypeShard, Status: "Done", Parent: okScatter},
	}
	root.Children = append(root.Children, okScatter)

	expandFailurePaths(root)

	if okScatter.Expanded {
		t.Error("all-Done scatter should be collapsed after failure expansion")
	}
}

func TestJumpToFailureCyclesThroughFailedLeaves(t *testing.T) {
	root := buildFailureTestTree()
	m := Model{tree: root, nodes: tree.GetVisibleNodes(root), metadata: &WorkflowMetadata{}}

	cmd := m.jumpToFailure(true)
	if cmd != nil {
		t.Error("jumpToFailure should not return a status cmd when a failure exists")
	}
	if got := m.nodes[m.cursor].ID; got != "wf.Scatter_1" {
		t.Errorf("first jump selected %q, want wf.Scatter_1", got)
	}

	m.jumpToFailure(true)
	if got := m.nodes[m.cursor].ID; got != "wf.Scatter_2" {
		t.Errorf("second jump selected %q, want wf.Scatter_2", got)
	}

	m.jumpToFailure(true)
	if got := m.nodes[m.cursor].ID; got != "wf.Sub" {
		t.Errorf("third jump selected %q, want wf.Sub", got)
	}

	// Wraps around back to the first failed leaf
	m.jumpToFailure(true)
	if got := m.nodes[m.cursor].ID; got != "wf.Scatter_1" {
		t.Errorf("wrap-around jump selected %q, want wf.Scatter_1", got)
	}

	// And backwards
	m.jumpToFailure(false)
	if got := m.nodes[m.cursor].ID; got != "wf.Sub" {
		t.Errorf("backward jump selected %q, want wf.Sub", got)
	}
}

func TestJumpToFailureWithoutFailures(t *testing.T) {
	root := &TreeNode{ID: "root", Name: "wf", Type: NodeTypeWorkflow, Status: "Succeeded", Expanded: true}
	root.Children = []*TreeNode{
		{ID: "wf.TaskA", Name: "TaskA", Type: NodeTypeCall, Status: "Done", Parent: root},
	}
	m := Model{tree: root, nodes: tree.GetVisibleNodes(root), metadata: &WorkflowMetadata{}}

	if cmd := m.jumpToFailure(true); cmd == nil {
		t.Error("jumpToFailure should return a status-clear cmd when nothing failed")
	}
	if m.statusMessage == "" {
		t.Error("jumpToFailure should set a status message when nothing failed")
	}
}

func TestPickWrapped(t *testing.T) {
	indices := []int{2, 5, 9}

	cases := []struct {
		current int
		next    bool
		want    int
	}{
		{current: -1, next: true, want: 2},
		{current: 2, next: true, want: 5},
		{current: 9, next: true, want: 2},  // wraps forward
		{current: 2, next: false, want: 9}, // wraps backward
		{current: 6, next: false, want: 5},
	}

	for _, tc := range cases {
		if got := pickWrapped(indices, tc.current, tc.next); got != tc.want {
			t.Errorf("pickWrapped(%v, %d, %v) = %d, want %d", indices, tc.current, tc.next, got, tc.want)
		}
	}
}
