package workflow

import (
	"testing"
	"time"
)

func hoursApart(start string, h float64) (time.Time, time.Time) {
	s, _ := time.Parse(time.RFC3339, start)
	return s, s.Add(time.Duration(h * float64(time.Hour)))
}

func TestCalculateCostBreakdownAggregatesAndSorts(t *testing.T) {
	s1, e1 := hoursApart("2026-07-06T06:00:00Z", 10) // expensive, non-preemptible
	s2, e2 := hoursApart("2026-07-06T06:00:00Z", 2)  // cheaper, preemptible
	s3, e3 := hoursApart("2026-07-06T06:00:00Z", 2)  // second shard of the cheap task

	wf := &Workflow{
		Calls: map[string][]Call{
			"WF.Merge": {
				{Name: "WF.Merge", ShardIndex: -1, Attempt: 1, Start: s1, End: e1, VMCostPerHour: 0.20, Preemptible: "false"},
			},
			"WF.Collect": {
				{Name: "WF.Collect", ShardIndex: 0, Attempt: 1, Start: s2, End: e2, VMCostPerHour: 0.05, Preemptible: "true"},
				{Name: "WF.Collect", ShardIndex: 1, Attempt: 1, Start: s3, End: e3, VMCostPerHour: 0.05, Preemptible: "true"},
			},
		},
	}

	b := wf.CalculateCostBreakdown()

	if len(b.Tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(b.Tasks))
	}
	// Merge = 10h * 0.20 = 2.00; Collect = 2*(2h*0.05) = 0.20
	if b.Tasks[0].Name != "Merge" {
		t.Errorf("expected Merge first (most expensive), got %s", b.Tasks[0].Name)
	}
	if got := b.Tasks[0].TotalCost; got < 1.99 || got > 2.01 {
		t.Errorf("Merge cost = %.3f, want ~2.00", got)
	}
	if b.Tasks[0].Preemptible {
		t.Errorf("Merge should be non-preemptible")
	}
	if b.Tasks[1].Name != "Collect" || b.Tasks[1].ShardCount != 2 {
		t.Errorf("Collect should have 2 shards, got %s/%d", b.Tasks[1].Name, b.Tasks[1].ShardCount)
	}
	if !b.Tasks[1].Preemptible {
		t.Errorf("Collect should be preemptible")
	}
	if total := b.TotalCost; total < 2.19 || total > 2.21 {
		t.Errorf("total = %.3f, want ~2.20", total)
	}
	if p := b.Tasks[0].Percent; p < 90 || p > 91 {
		t.Errorf("Merge percent = %.1f, want ~90.9", p)
	}
	if !b.FromActual {
		t.Errorf("expected FromActual true when all costs use vmCostPerHour")
	}
}

func TestCalculateCostBreakdownRecursesLoadedSubworkflowAndCountsPending(t *testing.T) {
	s, e := hoursApart("2026-07-06T06:00:00Z", 1)

	loaded := &Workflow{
		Calls: map[string][]Call{
			"Sub.Inner": {{Name: "Sub.Inner", ShardIndex: -1, Attempt: 1, Start: s, End: e, VMCostPerHour: 0.10, Preemptible: "false"}},
		},
	}

	wf := &Workflow{
		Calls: map[string][]Call{
			"WF.LoadedSub":   {{Name: "WF.LoadedSub", SubWorkflowID: "sub-1", SubWorkflowMetadata: loaded}},
			"WF.PendingSub":  {{Name: "WF.PendingSub", SubWorkflowID: "sub-2"}}, // not loaded
			"WF.PendingSub2": {{Name: "WF.PendingSub2", SubWorkflowID: "sub-3"}},
		},
	}

	b := wf.CalculateCostBreakdown()

	if len(b.Tasks) != 1 || b.Tasks[0].Name != "Inner" {
		t.Fatalf("expected only the loaded subworkflow's Inner task, got %+v", b.Tasks)
	}
	if b.SubworkflowsPending != 2 {
		t.Errorf("expected 2 pending subworkflows, got %d", b.SubworkflowsPending)
	}
}
