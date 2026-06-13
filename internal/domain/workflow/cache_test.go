package workflow

import (
	"testing"
	"time"
)

func TestParseCacheResult(t *testing.T) {
	src, ok := ParseCacheResult("Cache Hit: 84e93adf-9b82-46a3-9b16-ca964f715abf:Tso500SomaticAnalysis.Amber:-1")
	if !ok {
		t.Fatal("expected a valid cache hit to parse")
	}
	if src.WorkflowID != "84e93adf-9b82-46a3-9b16-ca964f715abf" {
		t.Errorf("workflowID = %q", src.WorkflowID)
	}
	if src.CallName != "Tso500SomaticAnalysis.Amber" {
		t.Errorf("callName = %q", src.CallName)
	}
	if src.ShardIndex != -1 {
		t.Errorf("shard = %d, want -1", src.ShardIndex)
	}

	shardHit, ok := ParseCacheResult("Cache Hit: 6d02e0f9-e671-4ff5-881f-46642f064b6a:HmftoolsDnaAlignment.Fastp:0")
	if !ok || shardHit.ShardIndex != 0 {
		t.Errorf("sharded hit = %+v (ok=%v), want shard 0", shardHit, ok)
	}
}

func TestParseCacheResult_NonHits(t *testing.T) {
	for _, s := range []string{
		"Cache Miss",
		"",
		"Cache Hit: incomplete",
		"Cache Hit: wf:call",          // missing shard
		"Cache Hit: wf:call:notanint", // bad shard
		"random text",
	} {
		if _, ok := ParseCacheResult(s); ok {
			t.Errorf("ParseCacheResult(%q) should not parse", s)
		}
	}
}

func TestWorkflowFindCall(t *testing.T) {
	w := &Workflow{
		Calls: map[string][]Call{
			"wf.T": {
				{Name: "wf.T", ShardIndex: -1, Attempt: 1, Status: StatusFailed},
				{Name: "wf.T", ShardIndex: -1, Attempt: 2, Status: StatusSucceeded},
				{Name: "wf.T", ShardIndex: 0, Attempt: 1, Status: StatusSucceeded},
			},
		},
	}

	// Latest attempt for shard -1 wins.
	c, ok := w.FindCall("wf.T", -1)
	if !ok || c.Attempt != 2 || c.Status != StatusSucceeded {
		t.Errorf("FindCall(-1) = %+v (ok=%v), want attempt 2 Succeeded", c, ok)
	}
	if c, ok := w.FindCall("wf.T", 0); !ok || c.ShardIndex != 0 {
		t.Errorf("FindCall(0) = %+v (ok=%v)", c, ok)
	}
	if _, ok := w.FindCall("wf.T", 7); ok {
		t.Error("FindCall for missing shard should be false")
	}
	if _, ok := w.FindCall("wf.Absent", -1); ok {
		t.Error("FindCall for missing call should be false")
	}
}

func TestCompareWorkflows_RecoveredCacheDuration(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// Run A executed for real: 40 min.
	a := &Workflow{Calls: map[string][]Call{
		"wf.Mark": {{
			ShardIndex: -1, Attempt: 1, Status: StatusSucceeded, DockerImage: "img:1",
			Start: base, End: base.Add(40 * time.Minute),
		}},
	}}
	// Run B cache-hit; the cached call's own timing is ~3s, but recovery carries
	// the real 38 min from the source run.
	b := &Workflow{Calls: map[string][]Call{
		"wf.Mark": {{
			ShardIndex: -1, Attempt: 1, Status: StatusSucceeded, DockerImage: "img:1",
			CacheHit: true, CacheResult: "Cache Hit: src:wf.Mark:-1",
			Start: base, End: base.Add(3 * time.Second),
			Recovery: &CacheRecovery{
				SourceWorkflowID: "src-wf",
				Start:            base, End: base.Add(38 * time.Minute),
				DockerImage: "img:1", Status: StatusSucceeded, Depth: 1,
			},
		}},
	}}

	d := CompareWorkflows(a, b)

	// 40m vs recovered 38m: <1.5x and not the misleading 3s → no false signal,
	// and nothing else differs, so the task is omitted entirely.
	if td, ok := taskDiffByName(d.Tasks, "wf.Mark"); ok {
		t.Errorf("recovered task with equivalent real duration should not be flagged, got %+v", td)
	}
}

func TestCompareWorkflows_RecoveredRevealsRealRegression(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	a := &Workflow{Calls: map[string][]Call{
		"wf.Mark": {{
			ShardIndex: -1, Attempt: 1, Status: StatusSucceeded,
			Start: base, End: base.Add(2 * time.Minute),
		}},
	}}
	// B cached from a source whose real run took 30 min — a genuine difference
	// that recovery surfaces (the cached copy time alone would hide it).
	b := &Workflow{Calls: map[string][]Call{
		"wf.Mark": {{
			ShardIndex: -1, Attempt: 1, Status: StatusSucceeded,
			CacheHit: true, CacheResult: "Cache Hit: src:wf.Mark:-1",
			Start: base, End: base.Add(4 * time.Second),
			Recovery: &CacheRecovery{
				SourceWorkflowID: "src-wf",
				Start:            base, End: base.Add(30 * time.Minute),
				Status: StatusSucceeded, Depth: 2,
			},
		}},
	}}

	d := CompareWorkflows(a, b)
	td, ok := taskDiffByName(d.Tasks, "wf.Mark")
	if !ok {
		t.Fatal("recovered duration regression should be reported")
	}
	if !td.DurationChangedSignificantly() {
		t.Errorf("2m → recovered 30m should be significant, got %+v", td)
	}
	if !td.RecoveredB || td.CacheSourceB != "src-wf" {
		t.Errorf("recovery provenance missing: recoveredB=%v source=%q", td.RecoveredB, td.CacheSourceB)
	}
	if td.DurationB != 30*time.Minute {
		t.Errorf("durationB = %s, want 30m (recovered)", td.DurationB)
	}
}

func TestCompareWorkflows_SubworkflowCacheServedSuppressesDuration(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// Side A: subworkflow call whose children were served from cache, so it
	// "ran" in 1 minute (cache artifact). Flag set by the application layer.
	a := &Workflow{Calls: map[string][]Call{
		"wf.Align": {{
			ShardIndex: -1, Attempt: 1, Status: StatusSucceeded,
			SubWorkflowID: "sub-a", SubworkflowCacheServed: true,
			Start: base, End: base.Add(1 * time.Minute),
		}},
	}}
	// Side B: same subworkflow ran for real (2h).
	b := &Workflow{Calls: map[string][]Call{
		"wf.Align": {{
			ShardIndex: -1, Attempt: 1, Status: StatusSucceeded,
			SubWorkflowID: "sub-b",
			Start:         base, End: base.Add(2 * time.Hour),
		}},
	}}

	d := CompareWorkflows(a, b)

	// The 1m vs 2h gap is a cache artifact, not a real 120× slowdown → the task
	// must not be flagged on duration; nothing else differs, so it is omitted.
	if td, ok := taskDiffByName(d.Tasks, "wf.Align"); ok {
		t.Errorf("subworkflow served from cache must not produce a duration diff, got %+v", td)
	}

	// Direct check of the suppression flag.
	td := TaskDiff{
		DurationA: 1 * time.Minute, DurationB: 2 * time.Hour,
		SubworkflowCachedA: true,
	}
	if td.DurationChangedSignificantly() {
		t.Error("subworkflow-cache-served side must suppress duration significance")
	}
}

func TestCompareWorkflows_SubworkflowFreshStillCompared(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	// Neither side cache-served → genuine subworkflow slowdown is still flagged.
	a := &Workflow{Calls: map[string][]Call{
		"wf.Align": {{ShardIndex: -1, Attempt: 1, Status: StatusSucceeded, SubWorkflowID: "x", Start: base, End: base.Add(20 * time.Minute)}},
	}}
	b := &Workflow{Calls: map[string][]Call{
		"wf.Align": {{ShardIndex: -1, Attempt: 1, Status: StatusSucceeded, SubWorkflowID: "y", Start: base, End: base.Add(2 * time.Hour)}},
	}}

	d := CompareWorkflows(a, b)
	if td, ok := taskDiffByName(d.Tasks, "wf.Align"); !ok || !td.DurationChangedSignificantly() {
		t.Errorf("fresh subworkflow slowdown should still be flagged, got %+v (ok=%v)", td, ok)
	}
}

func TestCompareWorkflows_RecoveredAttemptCount(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// A cache-hit (one local attempt) whose real source execution took 4
	// attempts — the recovered attempt count should be surfaced, not the
	// local "1".
	a := &Workflow{Calls: map[string][]Call{
		"wf.Mark": {{
			ShardIndex: -1, Attempt: 1, Status: StatusSucceeded,
			CacheHit: true, CacheResult: "Cache Hit: src:wf.Mark:-1",
			Start: base, End: base.Add(2 * time.Second),
			Recovery: &CacheRecovery{
				SourceWorkflowID: "src", Status: StatusSucceeded,
				Start: base, End: base.Add(20 * time.Minute), Attempt: 4,
			},
		}},
	}}
	// B ran fresh in a single attempt.
	b := &Workflow{Calls: map[string][]Call{
		"wf.Mark": {{
			ShardIndex: -1, Attempt: 1, Status: StatusSucceeded,
			Start: base, End: base.Add(19 * time.Minute),
		}},
	}}

	d := CompareWorkflows(a, b)
	td, ok := taskDiffByName(d.Tasks, "wf.Mark")
	if !ok {
		t.Fatal("attempts difference (4 vs 1) should flag the task")
	}
	if td.AttemptsA != 4 {
		t.Errorf("AttemptsA = %d, want 4 (recovered, not the local cache-copy attempt)", td.AttemptsA)
	}
	if td.AttemptsB != 1 {
		t.Errorf("AttemptsB = %d, want 1", td.AttemptsB)
	}
}

func TestCompareWorkflows_UnresolvedCacheSuppressesDuration(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	a := &Workflow{Calls: map[string][]Call{
		"wf.Mark": {{
			ShardIndex: -1, Attempt: 1, Status: StatusSucceeded,
			Start: base, End: base.Add(40 * time.Minute),
		}},
	}}
	// B cache hit that could NOT be recovered (no Recovery set): the 3s cache
	// copy must not be reported as "13x faster".
	b := &Workflow{Calls: map[string][]Call{
		"wf.Mark": {{
			ShardIndex: -1, Attempt: 1, Status: StatusSucceeded,
			CacheHit: true, CacheResult: "Cache Hit: src:wf.Mark:-1",
			Start: base, End: base.Add(3 * time.Second),
		}},
	}}

	d := CompareWorkflows(a, b)
	// No real difference is comparable → task omitted (no false signal).
	if _, ok := taskDiffByName(d.Tasks, "wf.Mark"); ok {
		t.Error("unresolved cache hit must not produce a false duration diff")
	}

	// Direct check of the suppression flag.
	td := TaskDiff{
		DurationA: 40 * time.Minute, DurationB: 3 * time.Second,
		UnresolvedCacheB: true,
	}
	if td.DurationChangedSignificantly() {
		t.Error("unresolved cache on a side must suppress duration significance")
	}
}
