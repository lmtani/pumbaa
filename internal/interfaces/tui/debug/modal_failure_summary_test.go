package debug

import (
	"strings"
	"testing"

	"github.com/lmtani/pumbaa/internal/domain/workflow"
)

func TestCollectFailureGroupsDeduplicatesAcrossShards(t *testing.T) {
	root := &TreeNode{ID: "root", Name: "wf", Type: NodeTypeWorkflow, Status: "Failed", Expanded: true}
	scatter := &TreeNode{ID: "wf.Scatter", Name: "Scatter", Type: NodeTypeCall, Status: "Failed", Parent: root}

	for i, msg := range []string{
		"Job exit code 137 at gs://bucket/run-1/shard-0/stderr",
		"Job exit code 137 at gs://bucket/run-1/shard-1/stderr",
		"Job exit code 1 at gs://bucket/run-1/shard-2/stderr",
	} {
		scatter.Children = append(scatter.Children, &TreeNode{
			ID:     "wf.Scatter_" + string(rune('0'+i)),
			Name:   "Scatter [shard " + string(rune('0'+i)) + "]",
			Type:   NodeTypeShard,
			Status: "Failed",
			Parent: scatter,
			CallData: &workflow.Call{
				Status:   workflow.StatusFailed,
				Failures: []Failure{{Message: msg}},
			},
		})
	}
	root.Children = []*TreeNode{scatter}

	groups := collectFailureGroups(root, nil)

	if len(groups) != 2 {
		t.Fatalf("got %d groups, want 2: %+v", len(groups), groups)
	}
	// Largest group first
	if len(groups[0].Tasks) != 2 {
		t.Errorf("first group has %d tasks, want 2", len(groups[0].Tasks))
	}
	if !strings.Contains(groups[0].Sample, "exit code 137") {
		t.Errorf("first group sample = %q, want the exit code 137 error", groups[0].Sample)
	}
	if len(groups[1].Tasks) != 1 {
		t.Errorf("second group has %d tasks, want 1", len(groups[1].Tasks))
	}
}

func TestCollectFailureGroupsFallsBackToWorkflowFailures(t *testing.T) {
	// Workflow-level failure with no failed calls (e.g. input validation)
	root := &TreeNode{ID: "root", Name: "wf", Type: NodeTypeWorkflow, Status: "Failed", Expanded: true}
	wfFailures := []Failure{{Message: "Required workflow input 'wf.sample' not specified"}}

	groups := collectFailureGroups(root, wfFailures)

	if len(groups) != 1 {
		t.Fatalf("got %d groups, want 1", len(groups))
	}
	if groups[0].Tasks[0].Name != "(workflow)" {
		t.Errorf("fallback group task = %q, want (workflow)", groups[0].Tasks[0].Name)
	}
}
