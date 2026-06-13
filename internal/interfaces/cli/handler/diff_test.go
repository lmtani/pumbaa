package handler

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/fatih/color"

	workflowdomain "github.com/lmtani/pumbaa/internal/domain/workflow"
	"github.com/lmtani/pumbaa/internal/interfaces/cli/presenter"
)

func newDiffHandlerWithBuffer() (*DiffHandler, *bytes.Buffer) {
	var buf bytes.Buffer
	return &DiffHandler{presenter: presenter.New(&buf)}, &buf
}

func TestDiffHandler_Display(t *testing.T) {
	color.NoColor = true // strip ANSI so we can assert on plain text

	diff := &workflowdomain.RunDiff{
		IDA: "aaaaaaaa1111", IDB: "bbbbbbbb2222",
		NameA: "wf", NameB: "wf",
		StatusA: workflowdomain.StatusSucceeded, StatusB: workflowdomain.StatusFailed,
		Inputs: []workflowdomain.KeyDiff{
			{Key: "wf.sample", Kind: workflowdomain.ChangeModified, ValueA: "NA1", ValueB: "NA2"},
			{Key: "wf.flag", Kind: workflowdomain.ChangeAdded, ValueB: "true"},
		},
		SourceChanged: true, SourceLinesA: 100, SourceLinesB: 105,
		Tasks: []workflowdomain.TaskDiff{
			{
				Name: "wf.Mark", Kind: workflowdomain.ChangeModified,
				StatusA: "Succeeded", StatusB: "Failed",
			},
			{
				Name: "wf.Slow", Kind: workflowdomain.ChangeModified,
				StatusA: "Succeeded", StatusB: "Succeeded",
				DurationA: 1 * time.Minute, DurationB: 10 * time.Minute,
			},
			{Name: "wf.New", Kind: workflowdomain.ChangeAdded, StatusB: "Succeeded"},
		},
		TotalTasks: 8,
	}

	h, buf := newDiffHandlerWithBuffer()
	h.display(diff)
	out := buf.String()

	for _, want := range []string{
		"Workflow Diff",
		"aaaaaaaa", "bbbbbbbb", // short IDs
		"Inputs (2 changed)",
		"wf.sample", "NA1", "NA2",
		"+ wf.flag",
		"WDL source changed (100 → 105 lines)",
		"Tasks (3 of 8 changed)",
		"wf.Mark",
		"status:",
		"10.0× slower",
		"+ wf.New",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("display output missing %q\n---\n%s", want, out)
		}
	}
}

func TestDiffHandler_DisplayCacheProvenance(t *testing.T) {
	color.NoColor = true

	diff := &workflowdomain.RunDiff{
		NameA: "wf", NameB: "wf",
		Tasks: []workflowdomain.TaskDiff{
			{
				// B's metrics were recovered from a cache source; the duration
				// line should be backed by a "recovered from" note.
				Name: "wf.Recovered", Kind: workflowdomain.ChangeModified,
				StatusA: "Succeeded", StatusB: "Succeeded",
				DurationA: 2 * time.Minute, DurationB: 30 * time.Minute,
				RecoveredB: true, CacheSourceB: "src12345abcd",
			},
			{
				// B cache hit that could not be resolved: status changed for a
				// real reason, and the note explains the duration was skipped.
				Name: "wf.Unresolved", Kind: workflowdomain.ChangeModified,
				StatusA: "Failed", StatusB: "Succeeded",
				DurationA: 5 * time.Minute, DurationB: 1 * time.Second,
				UnresolvedCacheB: true,
			},
			{
				// A is a subworkflow served from cache: its 1m wall-clock must
				// not be reported as a speedup vs B's real 2h.
				Name: "wf.AlignSub", Kind: workflowdomain.ChangeModified,
				StatusA: "Succeeded", StatusB: "Succeeded",
				DurationA: 1 * time.Minute, DurationB: 2 * time.Hour,
				ShardsA: 1, ShardsB: 2, // a real difference so the task is listed
				SubworkflowCachedA: true,
			},
		},
		TotalTasks: 2,
	}

	h, buf := newDiffHandlerWithBuffer()
	h.display(diff)
	out := buf.String()

	for _, want := range []string{
		"wf.Recovered",
		"B recovered from src12345",
		"duration: 2m 0s → 30m 0s", // recovered real duration, not cache-copy time
		"wf.Unresolved",
		"B cache hit (source unresolved)",
		"wf.AlignSub",
		"A subworkflow served from cache",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("display output missing %q\n---\n%s", want, out)
		}
	}

	// The only legitimate duration verdict here is the recovered task's 15×;
	// the unresolved and subworkflow-cached tasks must not show one.
	if strings.Contains(out, "120.0× slower") || strings.Contains(out, "× faster") {
		t.Errorf("cache-artifact duration must not produce a verdict:\n%s", out)
	}
}

func TestDiffHandler_DisplayNoDifferences(t *testing.T) {
	color.NoColor = true

	diff := &workflowdomain.RunDiff{
		NameA: "wf", NameB: "wf",
		StatusA: workflowdomain.StatusSucceeded, StatusB: workflowdomain.StatusSucceeded,
	}

	h, buf := newDiffHandlerWithBuffer()
	h.display(diff)

	if !strings.Contains(buf.String(), "No differences found") {
		t.Errorf("expected 'No differences found', got:\n%s", buf.String())
	}
}

func TestDiffHandler_DisplayNameMismatch(t *testing.T) {
	color.NoColor = true

	diff := &workflowdomain.RunDiff{
		NameA: "Alpha", NameB: "Beta",
		NameMismatch: true,
		Inputs:       []workflowdomain.KeyDiff{{Key: "x", Kind: workflowdomain.ChangeAdded, ValueB: "1"}},
	}

	h, buf := newDiffHandlerWithBuffer()
	h.display(diff)

	if !strings.Contains(buf.String(), "names differ") {
		t.Errorf("expected name mismatch warning, got:\n%s", buf.String())
	}
}
