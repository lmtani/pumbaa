package debug

import (
	"strings"
	"testing"

	"github.com/lmtani/pumbaa/internal/domain/workflow"
)

func TestNormalizeFailureSignature(t *testing.T) {
	a := normalizeFailureSignature(
		"Task failed (shard 3): exit code 137. See gs://bucket/wf/11111111-2222-3333-4444-555555555555/call-X/shard-3/stderr")
	b := normalizeFailureSignature(
		"Task failed (shard 47): exit code 137. See gs://bucket/wf/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/call-X/shard-47/stderr")

	if a != b {
		t.Errorf("signatures should match:\n a=%q\n b=%q", a, b)
	}

	c := normalizeFailureSignature("Task failed (shard 3): exit code 1.")
	if a == c {
		t.Error("different exit codes should produce different signatures")
	}
}

func TestRootCauseMessages(t *testing.T) {
	f := Failure{
		Message: "Workflow failed",
		CausedBy: []Failure{
			{Message: "Job failed", CausedBy: []Failure{{Message: "OOM killed"}}},
			{Message: "Disk full"},
		},
	}

	msgs := rootCauseMessages(f)
	if len(msgs) != 2 || msgs[0] != "OOM killed" || msgs[1] != "Disk full" {
		t.Errorf("rootCauseMessages = %v, want [OOM killed, Disk full]", msgs)
	}
}

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
	if len(groups[0].tasks) != 2 {
		t.Errorf("first group has %d tasks, want 2", len(groups[0].tasks))
	}
	if !strings.Contains(groups[0].sample, "exit code 137") {
		t.Errorf("first group sample = %q, want the exit code 137 error", groups[0].sample)
	}
	if len(groups[1].tasks) != 1 {
		t.Errorf("second group has %d tasks, want 1", len(groups[1].tasks))
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
	if groups[0].tasks[0] != "(workflow)" {
		t.Errorf("fallback group task = %q, want (workflow)", groups[0].tasks[0])
	}
}
