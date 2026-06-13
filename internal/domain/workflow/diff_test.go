package workflow

import (
	"encoding/json"
	"testing"
	"time"
)

func keyDiffByKey(diffs []KeyDiff, key string) (KeyDiff, bool) {
	for _, d := range diffs {
		if d.Key == key {
			return d, true
		}
	}
	return KeyDiff{}, false
}

func taskDiffByName(diffs []TaskDiff, name string) (TaskDiff, bool) {
	for _, d := range diffs {
		if d.Name == name {
			return d, true
		}
	}
	return TaskDiff{}, false
}

func TestCompareWorkflows_Inputs(t *testing.T) {
	a := &Workflow{
		SubmittedInputs: `{"wf.sample":"NA12878","wf.threads":4,"wf.old":"x","wf.fastqs":["gs://a","gs://b"]}`,
	}
	b := &Workflow{
		SubmittedInputs: `{"wf.sample":"NA12879","wf.threads":4,"wf.new":true,"wf.fastqs":["gs://a","gs://c"]}`,
	}

	d := CompareWorkflows(a, b)

	// wf.threads unchanged → must not appear
	if _, ok := keyDiffByKey(d.Inputs, "wf.threads"); ok {
		t.Error("unchanged key wf.threads should not be reported")
	}

	if kd, ok := keyDiffByKey(d.Inputs, "wf.sample"); !ok || kd.Kind != ChangeModified ||
		kd.ValueA != "NA12878" || kd.ValueB != "NA12879" {
		t.Errorf("wf.sample diff = %+v (ok=%v), want modified NA12878→NA12879", kd, ok)
	}
	if kd, ok := keyDiffByKey(d.Inputs, "wf.new"); !ok || kd.Kind != ChangeAdded || kd.ValueB != "true" {
		t.Errorf("wf.new diff = %+v (ok=%v), want added true", kd, ok)
	}
	if kd, ok := keyDiffByKey(d.Inputs, "wf.old"); !ok || kd.Kind != ChangeRemoved || kd.ValueA != "x" {
		t.Errorf("wf.old diff = %+v (ok=%v), want removed x", kd, ok)
	}
	// only the second fastq changed
	if kd, ok := keyDiffByKey(d.Inputs, "wf.fastqs[1]"); !ok || kd.Kind != ChangeModified {
		t.Errorf("wf.fastqs[1] diff = %+v (ok=%v), want modified", kd, ok)
	}
	if _, ok := keyDiffByKey(d.Inputs, "wf.fastqs[0]"); ok {
		t.Error("unchanged wf.fastqs[0] should not be reported")
	}
}

func TestCompareWorkflows_EmptyAndUnparseableInputs(t *testing.T) {
	// One side empty, other present → all keys added.
	d := CompareWorkflows(&Workflow{}, &Workflow{SubmittedInputs: `{"wf.x":1}`})
	if kd, ok := keyDiffByKey(d.Inputs, "wf.x"); !ok || kd.Kind != ChangeAdded {
		t.Errorf("wf.x = %+v (ok=%v), want added", kd, ok)
	}

	// Unparseable but different → reported under (unparseable).
	d = CompareWorkflows(
		&Workflow{SubmittedInputs: "not json A"},
		&Workflow{SubmittedInputs: "not json B"},
	)
	if kd, ok := keyDiffByKey(d.Inputs, "(unparseable)"); !ok || kd.Kind != ChangeModified {
		t.Errorf("(unparseable) = %+v (ok=%v), want modified", kd, ok)
	}
}

func TestCompareWorkflows_Source(t *testing.T) {
	a := &Workflow{SubmittedWorkflow: "version 1.0\nworkflow wf {\n}\n"}
	b := &Workflow{SubmittedWorkflow: "version 1.0\nworkflow wf {\n  call x\n}\n"}

	d := CompareWorkflows(a, b)
	if !d.SourceChanged {
		t.Error("source should be reported as changed")
	}
	// Source is trimmed before counting, so trailing newlines do not count.
	if d.SourceLinesA != 3 || d.SourceLinesB != 4 {
		t.Errorf("source lines = %d/%d, want 3/4", d.SourceLinesA, d.SourceLinesB)
	}

	same := CompareWorkflows(a, a)
	if same.SourceChanged {
		t.Error("identical source should not be reported as changed")
	}
}

func TestCompareWorkflows_Tasks(t *testing.T) {
	a := &Workflow{
		Calls: map[string][]Call{
			// status change Succeeded → Failed
			"wf.Mark": {{ShardIndex: -1, Attempt: 1, Status: StatusSucceeded, DockerImage: "img:1"}},
			// docker bump
			"wf.Star": {{ShardIndex: -1, Attempt: 1, Status: StatusSucceeded, DockerImage: "img:1.0"}},
			// only in A → removed
			"wf.Old": {{ShardIndex: -1, Attempt: 1, Status: StatusSucceeded}},
			// unchanged → must not appear
			"wf.Same": {{ShardIndex: -1, Attempt: 1, Status: StatusSucceeded, DockerImage: "x"}},
		},
	}
	b := &Workflow{
		Calls: map[string][]Call{
			"wf.Mark": {{ShardIndex: -1, Attempt: 1, Status: StatusFailed, DockerImage: "img:1"}},
			"wf.Star": {{ShardIndex: -1, Attempt: 1, Status: StatusSucceeded, DockerImage: "img:1.1"}},
			// only in B → added
			"wf.New":  {{ShardIndex: -1, Attempt: 1, Status: StatusSucceeded}},
			"wf.Same": {{ShardIndex: -1, Attempt: 1, Status: StatusSucceeded, DockerImage: "x"}},
		},
	}

	d := CompareWorkflows(a, b)

	// Union of task names: Mark, Star, Old, Same, New = 5.
	if d.TotalTasks != 5 {
		t.Errorf("TotalTasks = %d, want 5", d.TotalTasks)
	}
	if _, ok := taskDiffByName(d.Tasks, "wf.Same"); ok {
		t.Error("unchanged task wf.Same should not be reported")
	}
	if td, ok := taskDiffByName(d.Tasks, "wf.Mark"); !ok || td.Kind != ChangeModified || !td.StatusChanged() {
		t.Errorf("wf.Mark = %+v (ok=%v), want modified status change", td, ok)
	}
	if td, ok := taskDiffByName(d.Tasks, "wf.Star"); !ok || !td.DockerChanged() ||
		td.DockerA != "img:1.0" || td.DockerB != "img:1.1" {
		t.Errorf("wf.Star = %+v (ok=%v), want docker change", td, ok)
	}
	if td, ok := taskDiffByName(d.Tasks, "wf.New"); !ok || td.Kind != ChangeAdded {
		t.Errorf("wf.New = %+v (ok=%v), want added", td, ok)
	}
	if td, ok := taskDiffByName(d.Tasks, "wf.Old"); !ok || td.Kind != ChangeRemoved {
		t.Errorf("wf.Old = %+v (ok=%v), want removed", td, ok)
	}
}

func TestCompareWorkflows_TaskDurationSignificance(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mk := func(seconds int) []Call {
		return []Call{{
			ShardIndex: -1, Attempt: 1, Status: StatusSucceeded,
			Start: base, End: base.Add(time.Duration(seconds) * time.Second),
		}}
	}

	// 60s → 600s: 10× slower, well over thresholds → flagged.
	slow := CompareWorkflows(
		&Workflow{Calls: map[string][]Call{"wf.T": mk(60)}},
		&Workflow{Calls: map[string][]Call{"wf.T": mk(600)}},
	)
	if td, ok := taskDiffByName(slow.Tasks, "wf.T"); !ok || !td.DurationChangedSignificantly() {
		t.Errorf("10x slower task should be flagged, got %+v (ok=%v)", td, ok)
	}

	// 10s → 20s: 2× but only 10s absolute (< 30s) → not flagged, task omitted.
	noise := CompareWorkflows(
		&Workflow{Calls: map[string][]Call{"wf.T": mk(10)}},
		&Workflow{Calls: map[string][]Call{"wf.T": mk(20)}},
	)
	if _, ok := taskDiffByName(noise.Tasks, "wf.T"); ok {
		t.Error("small absolute duration change should not flag the task")
	}
}

func TestCompareWorkflows_ShardsAndRunning(t *testing.T) {
	a := &Workflow{
		Calls: map[string][]Call{
			"wf.Scatter": {
				{ShardIndex: 0, Attempt: 1, Status: StatusSucceeded},
				{ShardIndex: 1, Attempt: 1, Status: StatusSucceeded},
			},
		},
	}
	b := &Workflow{
		Calls: map[string][]Call{
			"wf.Scatter": {
				{ShardIndex: 0, Attempt: 1, Status: StatusSucceeded},
				{ShardIndex: 1, Attempt: 1, Status: StatusRunning},
				{ShardIndex: 2, Attempt: 1, Status: StatusSucceeded},
			},
		},
	}

	d := CompareWorkflows(a, b)
	td, ok := taskDiffByName(d.Tasks, "wf.Scatter")
	if !ok {
		t.Fatal("wf.Scatter should be reported (shard count changed)")
	}
	if td.ShardsA != 2 || td.ShardsB != 3 {
		t.Errorf("shards = %d/%d, want 2/3", td.ShardsA, td.ShardsB)
	}
	if td.StatusB != string(StatusRunning) || !td.RunningB {
		t.Errorf("run B should aggregate to Running, got status=%q running=%v", td.StatusB, td.RunningB)
	}
}

func TestCompareWorkflows_NameMismatch(t *testing.T) {
	d := CompareWorkflows(&Workflow{Name: "Alpha"}, &Workflow{Name: "Beta"})
	if !d.NameMismatch {
		t.Error("different names should set NameMismatch")
	}
	d = CompareWorkflows(&Workflow{Name: "Alpha"}, &Workflow{Name: "Alpha"})
	if d.NameMismatch {
		t.Error("equal names should not set NameMismatch")
	}
}

func TestRunDiff_HasDifferences(t *testing.T) {
	identical := &Workflow{
		Name:            "wf",
		SubmittedInputs: `{"wf.x":1}`,
		Calls:           map[string][]Call{"wf.T": {{ShardIndex: -1, Attempt: 1, Status: StatusSucceeded}}},
	}
	if CompareWorkflows(identical, identical).HasDifferences() {
		t.Error("comparing a run with itself should report no differences")
	}
}

func TestChangeKind_MarshalJSON(t *testing.T) {
	b, err := json.Marshal(ChangeModified)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(b) != `"modified"` {
		t.Errorf("ChangeModified JSON = %s, want \"modified\"", b)
	}
}
