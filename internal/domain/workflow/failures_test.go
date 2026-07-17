package workflow

import (
	"strings"
	"testing"
)

func TestNormalizeFailureSignature(t *testing.T) {
	a := NormalizeFailureSignature(
		"Task failed (shard 3): exit code 137. See gs://bucket/wf/11111111-2222-3333-4444-555555555555/call-X/shard-3/stderr")
	b := NormalizeFailureSignature(
		"Task failed (shard 47): exit code 137. See gs://bucket/wf/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/call-X/shard-47/stderr")

	if a != b {
		t.Errorf("signatures should match:\n a=%q\n b=%q", a, b)
	}

	c := NormalizeFailureSignature("Task failed (shard 3): exit code 1.")
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

	msgs := RootCauseMessages(f)
	if len(msgs) != 2 || msgs[0] != "OOM killed" || msgs[1] != "Disk full" {
		t.Errorf("RootCauseMessages = %v, want [OOM killed, Disk full]", msgs)
	}
}

func TestCalculateFailureSummaryGroupsAcrossSubworkflows(t *testing.T) {
	s, e := hoursApart("2026-07-06T06:00:00Z", 1)

	// Two subworkflow instances, each with the same failing task; a third
	// task fails with a different error.
	makeSub := func(stderr string) *Workflow {
		return &Workflow{
			Calls: map[string][]Call{
				"Sub.Align": {{
					Name: "Sub.Align", ShardIndex: -1, Attempt: 1, Status: StatusFailed,
					Start: s, End: e, Stderr: stderr,
					Failures: []Failure{{Message: "Job exit code 137 at gs://bucket/run/stderr"}},
				}},
			},
		}
	}

	wf := &Workflow{
		Calls: map[string][]Call{
			"WF.SubA": {{Name: "WF.SubA", ShardIndex: 0, SubWorkflowID: "a", SubWorkflowMetadata: makeSub("gs://b/a/stderr")}},
			"WF.SubB": {{Name: "WF.SubB", ShardIndex: 1, SubWorkflowID: "b", SubWorkflowMetadata: makeSub("gs://b/b/stderr")}},
			"WF.Other": {{
				Name: "WF.Other", ShardIndex: -1, Attempt: 1, Status: StatusFailed,
				Failures: []Failure{{Message: "PAPI error code 9: disk full"}},
			}},
		},
	}

	sum := wf.CalculateFailureSummary()

	if sum.FailedTasks != 3 {
		t.Errorf("FailedTasks = %d, want 3", sum.FailedTasks)
	}
	if len(sum.Groups) != 2 {
		t.Fatalf("got %d groups, want 2: %+v", len(sum.Groups), sum.Groups)
	}
	// Largest group first: the two Align failures share a signature.
	if len(sum.Groups[0].Tasks) != 2 || !strings.Contains(sum.Groups[0].Sample, "exit code 137") {
		t.Errorf("first group should be the 2 Align failures, got %+v", sum.Groups[0])
	}
	if sum.Groups[0].Tasks[0].Stderr == "" {
		t.Errorf("failed task should carry its stderr path")
	}
}

func TestCalculateFailureSummaryUsesFinalAttemptOnly(t *testing.T) {
	s, e := hoursApart("2026-07-06T06:00:00Z", 1)

	// Attempt 1 preempted (RetryableFailure), attempt 2 succeeded: the task
	// did NOT fail.
	wf := &Workflow{
		Calls: map[string][]Call{
			"WF.Retried": {
				{Name: "WF.Retried", ShardIndex: -1, Attempt: 1, Status: "RetryableFailure", Start: s, End: e,
					Failures: []Failure{{Message: "preempted"}}},
				{Name: "WF.Retried", ShardIndex: -1, Attempt: 2, Status: StatusSucceeded, Start: s, End: e},
			},
		},
	}

	sum := wf.CalculateFailureSummary()

	if sum.FailedTasks != 0 || len(sum.Groups) != 0 {
		t.Errorf("retried-then-succeeded task must not appear as failed: %+v", sum)
	}
}

func TestCalculateFailureSummaryFallsBackToWorkflowFailures(t *testing.T) {
	wf := &Workflow{
		Failures: []Failure{{Message: "Required workflow input 'wf.sample' not specified"}},
	}

	sum := wf.CalculateFailureSummary()

	if len(sum.Groups) != 1 {
		t.Fatalf("got %d groups, want 1", len(sum.Groups))
	}
	if sum.Groups[0].Tasks[0].Name != "(workflow)" {
		t.Errorf("fallback task = %q, want (workflow)", sum.Groups[0].Tasks[0].Name)
	}
}

func TestCalculateFailureSummaryShardLabels(t *testing.T) {
	wf := &Workflow{
		Calls: map[string][]Call{
			"WF.Scatter": {
				{Name: "WF.Scatter", ShardIndex: 3, Attempt: 1, Status: StatusFailed,
					Failures: []Failure{{Message: "boom"}}},
			},
		},
	}

	sum := wf.CalculateFailureSummary()

	if got := sum.Groups[0].Tasks[0].Name; got != "Scatter[3]" {
		t.Errorf("task label = %q, want Scatter[3]", got)
	}
}
