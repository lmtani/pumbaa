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
	if got := b.Tasks[0].ActualCost; got < 1.99 || got > 2.01 {
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
	if total := b.ActualTotal; total < 2.19 || total > 2.21 {
		t.Errorf("actual total = %.3f, want ~2.20", total)
	}
	if p := b.Tasks[0].Percent; p < 90 || p > 91 {
		t.Errorf("Merge percent = %.1f, want ~90.9", p)
	}
	if b.HasEstimates() {
		t.Errorf("no estimates expected when every attempt has vmCostPerHour, got %v", b.EstimatedTotal)
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

func TestBillableHoursRunningAttemptAccruesToNow(t *testing.T) {
	now, _ := time.Parse(time.RFC3339, "2026-07-16T13:00:00Z")

	running := Call{Status: StatusRunning, VMStartTime: now.Add(-2 * time.Hour)}
	if h := billableHours(running, now); h != 2.0 {
		t.Errorf("running attempt should accrue vmStart→now, got %v hours", h)
	}

	// Without VM timestamps it falls back to the task start.
	runningNoVM := Call{Status: StatusRunning, Start: now.Add(-30 * time.Minute)}
	if h := billableHours(runningNoVM, now); h != 0.5 {
		t.Errorf("running attempt without VM times should accrue start→now, got %v hours", h)
	}

	// A non-running attempt with a missing end has no billable window: it
	// must not accrue forever.
	stale := Call{Status: StatusFailed, VMStartTime: now.Add(-2 * time.Hour)}
	if h := billableHours(stale, now); h != 0 {
		t.Errorf("non-running attempt without end should be 0, got %v hours", h)
	}
}

func TestCalculateAttemptCostRunningUsesRealRate(t *testing.T) {
	now, _ := time.Parse(time.RFC3339, "2026-07-16T13:00:00Z")

	// A running attempt with a real vmCostPerHour must be charged for the
	// time consumed so far — not the cpu×mem resource estimate (the bug that
	// inflated running workflows: 12 CPU × 54 GB × 0.01 = 6.48 pseudo-$).
	call := Call{
		Status:        StatusRunning,
		VMStartTime:   now.Add(-2 * time.Hour),
		VMCostPerHour: 0.20,
		CPU:           "12",
		Memory:        "54 GB",
	}
	if cost := calculateAttemptCost(call, now); cost != 0.40 {
		t.Errorf("running attempt cost = %v, want 0.40 (rate × accrued hours)", cost)
	}
}

func TestCalculateCostBreakdownIncludesRunningTasks(t *testing.T) {
	wf := &Workflow{
		Calls: map[string][]Call{
			"WF.Live": {{
				Name:          "WF.Live",
				ShardIndex:    -1,
				Attempt:       1,
				Status:        StatusRunning,
				VMStartTime:   time.Now().Add(-2 * time.Hour),
				VMCostPerHour: 0.10,
				Preemptible:   "false",
			}},
		},
	}

	b := wf.CalculateCostBreakdown()

	if len(b.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %+v", b.Tasks)
	}
	got := b.Tasks[0]
	if got.ActualCost < 0.19 || got.ActualCost > 0.21 {
		t.Errorf("running task cost = %v, want ≈0.20 (0.10/h × 2h accrued)", got.ActualCost)
	}
	if got.VMHours < 1.9 || got.VMHours > 2.1 {
		t.Errorf("running task VMHours = %v, want ≈2", got.VMHours)
	}
	if got.EstimatedCost != 0 {
		t.Errorf("cost from real vmCostPerHour should not produce an estimate, got %v", got.EstimatedCost)
	}
}

func TestCalculateCostBreakdownKeepsActualAndEstimateApart(t *testing.T) {
	s1, e1 := hoursApart("2026-07-06T06:00:00Z", 10)
	s2, e2 := hoursApart("2026-07-06T06:00:00Z", 2)

	wf := &Workflow{
		Calls: map[string][]Call{
			// Real cost: 10h × $0.20 = $2.00
			"WF.Priced": {{Name: "WF.Priced", ShardIndex: -1, Attempt: 1, Start: s1, End: e1, VMCostPerHour: 0.20, Preemptible: "false"}},
			// No vmCostPerHour: falls back to 4 cpu × 8 GB × 2h = 64 resource-hours
			"WF.Unpriced": {{Name: "WF.Unpriced", ShardIndex: -1, Attempt: 1, Start: s2, End: e2, CPU: "4", Memory: "8 GB", Preemptible: "true"}},
		},
	}

	b := wf.CalculateCostBreakdown()

	if b.ActualTotal < 1.99 || b.ActualTotal > 2.01 {
		t.Errorf("actual total = %v, want ~2.00 — the 64 resource-hours estimate must not leak into dollars", b.ActualTotal)
	}
	if b.EstimatedTotal < 63.9 || b.EstimatedTotal > 64.1 {
		t.Errorf("estimated total = %v, want ~64 resource-hours", b.EstimatedTotal)
	}
	if !b.HasEstimates() {
		t.Errorf("HasEstimates should be true when an attempt lacks cost data")
	}

	// The priced task sorts first even though 64 > 2 numerically: units are
	// not comparable, dollars lead.
	if b.Tasks[0].Name != "Priced" {
		t.Fatalf("expected Priced first, got %s", b.Tasks[0].Name)
	}
	// Percent is a share of the dollar total only: the priced task owns 100%
	// and the estimate-only task has no dollar share.
	if p := b.Tasks[0].Percent; p < 99.9 || p > 100.1 {
		t.Errorf("Priced percent = %v, want 100 (share of dollars only)", p)
	}
	if p := b.Tasks[1].Percent; p != 0 {
		t.Errorf("Unpriced percent = %v, want 0 (it has no dollar cost)", p)
	}
}

func TestCalculateCostBreakdownPercentFallsBackToEstimates(t *testing.T) {
	s1, e1 := hoursApart("2026-07-06T06:00:00Z", 3)
	s2, e2 := hoursApart("2026-07-06T06:00:00Z", 1)

	// A backend that reports no cost at all: percentages come from the
	// estimates so relative weights are still meaningful.
	wf := &Workflow{
		Calls: map[string][]Call{
			"WF.Big":   {{Name: "WF.Big", ShardIndex: -1, Attempt: 1, Start: s1, End: e1, CPU: "1", Memory: "1 GB"}},
			"WF.Small": {{Name: "WF.Small", ShardIndex: -1, Attempt: 1, Start: s2, End: e2, CPU: "1", Memory: "1 GB"}},
		},
	}

	b := wf.CalculateCostBreakdown()

	if b.ActualTotal != 0 {
		t.Fatalf("actual total = %v, want 0", b.ActualTotal)
	}
	if b.Tasks[0].Name != "Big" {
		t.Fatalf("expected Big first, got %s", b.Tasks[0].Name)
	}
	if p := b.Tasks[0].Percent; p < 74.9 || p > 75.1 {
		t.Errorf("Big percent = %v, want ~75 (3 of 4 resource-hours)", p)
	}
}
