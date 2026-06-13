package workflow

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lmtani/pumbaa/internal/domain/workflow"
)

func TestCacheResolver_RecoversRealMetrics(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// Source run that actually executed wf.Mark for 30 minutes.
	source := &workflow.Workflow{
		ID: "src-1",
		Calls: map[string][]workflow.Call{
			"wf.Mark": {{
				Name: "wf.Mark", ShardIndex: -1, Attempt: 1, Status: workflow.StatusSucceeded,
				DockerImage: "img:1", Start: base, End: base.Add(30 * time.Minute),
			}},
		},
	}

	repo := &mockWorkflowRepository{
		getMetadataFunc: func(ctx context.Context, id string) (*workflow.Workflow, error) {
			if id == "src-1" {
				return source, nil
			}
			return nil, errors.New("unknown id")
		},
	}

	// The run under inspection: wf.Mark was a cache hit copying from src-1.
	w := &workflow.Workflow{
		Calls: map[string][]workflow.Call{
			"wf.Mark": {{
				Name: "wf.Mark", ShardIndex: -1, Attempt: 1, Status: workflow.StatusSucceeded,
				CacheHit: true, CacheResult: "Cache Hit: src-1:wf.Mark:-1",
				Start: base, End: base.Add(2 * time.Second),
			}},
		},
	}

	newCacheResolver(repo).resolve(context.Background(), w)

	rec := w.Calls["wf.Mark"][0].Recovery
	if rec == nil {
		t.Fatal("expected Recovery to be set")
	}
	if rec.SourceWorkflowID != "src-1" {
		t.Errorf("source = %q, want src-1", rec.SourceWorkflowID)
	}
	if got := rec.End.Sub(rec.Start); got != 30*time.Minute {
		t.Errorf("recovered duration = %s, want 30m", got)
	}
}

func TestCacheResolver_FollowsChain(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// terminal real execution lives in src-2; src-1 is itself a cache hit of it.
	src1 := &workflow.Workflow{
		ID: "src-1",
		Calls: map[string][]workflow.Call{
			"wf.Mark": {{
				Name: "wf.Mark", ShardIndex: -1, Attempt: 1, Status: workflow.StatusSucceeded,
				CacheHit: true, CacheResult: "Cache Hit: src-2:wf.Mark:-1",
				Start: base, End: base.Add(2 * time.Second),
			}},
		},
	}
	src2 := &workflow.Workflow{
		ID: "src-2",
		Calls: map[string][]workflow.Call{
			"wf.Mark": {{
				Name: "wf.Mark", ShardIndex: -1, Attempt: 1, Status: workflow.StatusSucceeded,
				Start: base, End: base.Add(45 * time.Minute),
			}},
		},
	}

	repo := &mockWorkflowRepository{
		getMetadataFunc: func(ctx context.Context, id string) (*workflow.Workflow, error) {
			switch id {
			case "src-1":
				return src1, nil
			case "src-2":
				return src2, nil
			}
			return nil, errors.New("unknown id")
		},
	}

	w := &workflow.Workflow{
		Calls: map[string][]workflow.Call{
			"wf.Mark": {{
				Name: "wf.Mark", ShardIndex: -1, Attempt: 1, Status: workflow.StatusSucceeded,
				CacheHit: true, CacheResult: "Cache Hit: src-1:wf.Mark:-1",
			}},
		},
	}

	newCacheResolver(repo).resolve(context.Background(), w)

	rec := w.Calls["wf.Mark"][0].Recovery
	if rec == nil {
		t.Fatal("expected Recovery via chain")
	}
	// Immediate source is what the run pointed at; metrics come from the terminal.
	if rec.SourceWorkflowID != "src-1" {
		t.Errorf("immediate source = %q, want src-1", rec.SourceWorkflowID)
	}
	if got := rec.End.Sub(rec.Start); got != 45*time.Minute {
		t.Errorf("recovered duration = %s, want 45m (terminal real run)", got)
	}
	if rec.Depth != 2 {
		t.Errorf("chain depth = %d, want 2", rec.Depth)
	}
}

func TestCacheResolver_SubworkflowCacheServed(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// Subworkflow whose two leaf children were both cache hits.
	subAllCached := &workflow.Workflow{
		ID: "sub-cached",
		Calls: map[string][]workflow.Call{
			"Align.Fastp":   {{Name: "Align.Fastp", ShardIndex: -1, Attempt: 1, CacheHit: true, CacheResult: "Cache Hit: x:Align.Fastp:-1"}},
			"Align.Bwamem2": {{Name: "Align.Bwamem2", ShardIndex: -1, Attempt: 1, CacheHit: true, CacheResult: "Cache Hit: x:Align.Bwamem2:-1"}},
		},
	}
	// Subworkflow where one child ran fresh.
	subFresh := &workflow.Workflow{
		ID: "sub-fresh",
		Calls: map[string][]workflow.Call{
			"Align.Fastp":   {{Name: "Align.Fastp", ShardIndex: -1, Attempt: 1, CacheHit: true, CacheResult: "Cache Hit: x:Align.Fastp:-1"}},
			"Align.Bwamem2": {{Name: "Align.Bwamem2", ShardIndex: -1, Attempt: 1, CacheHit: false, Start: base, End: base.Add(2 * time.Hour)}},
		},
	}
	repo := &mockWorkflowRepository{
		getMetadataFunc: func(ctx context.Context, id string) (*workflow.Workflow, error) {
			switch id {
			case "sub-cached":
				return subAllCached, nil
			case "sub-fresh":
				return subFresh, nil
			}
			return nil, errors.New("unknown id")
		},
	}

	w := &workflow.Workflow{
		Calls: map[string][]workflow.Call{
			"wf.AlignCached": {{Name: "wf.AlignCached", ShardIndex: -1, Attempt: 1, Status: workflow.StatusSucceeded, SubWorkflowID: "sub-cached"}},
			"wf.AlignFresh":  {{Name: "wf.AlignFresh", ShardIndex: -1, Attempt: 1, Status: workflow.StatusSucceeded, SubWorkflowID: "sub-fresh"}},
		},
	}

	newCacheResolver(repo).resolve(context.Background(), w)

	if !w.Calls["wf.AlignCached"][0].SubworkflowCacheServed {
		t.Error("subworkflow with all children cached should be flagged SubworkflowCacheServed")
	}
	if w.Calls["wf.AlignFresh"][0].SubworkflowCacheServed {
		t.Error("subworkflow with a fresh child must NOT be flagged")
	}
}

func TestCacheResolver_SubworkflowInlinedMetadataNoFetch(t *testing.T) {
	calls := 0
	repo := &mockWorkflowRepository{
		getMetadataFunc: func(ctx context.Context, id string) (*workflow.Workflow, error) {
			calls++
			return nil, errors.New("should not fetch when metadata is inlined")
		},
	}

	w := &workflow.Workflow{
		Calls: map[string][]workflow.Call{
			"wf.Align": {{
				Name: "wf.Align", ShardIndex: -1, Attempt: 1, Status: workflow.StatusSucceeded,
				SubWorkflowMetadata: &workflow.Workflow{
					Calls: map[string][]workflow.Call{
						"Align.T": {{Name: "Align.T", ShardIndex: -1, Attempt: 1, CacheHit: true, CacheResult: "Cache Hit: x:Align.T:-1"}},
					},
				},
			}},
		},
	}

	newCacheResolver(repo).resolve(context.Background(), w)

	if calls != 0 {
		t.Errorf("inlined subworkflow metadata should not trigger a fetch, got %d fetches", calls)
	}
	if !w.Calls["wf.Align"][0].SubworkflowCacheServed {
		t.Error("inlined all-cached subworkflow should be flagged")
	}
}

func TestCacheResolver_RecoversTerminalAttemptCount(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// Terminal real execution that took 3 attempts (FindCall returns the
	// latest); the intermediate cache-hit hop has only attempt 1.
	terminal := &workflow.Workflow{
		ID: "src-2",
		Calls: map[string][]workflow.Call{
			"wf.Mark": {
				{Name: "wf.Mark", ShardIndex: -1, Attempt: 1, Status: workflow.StatusFailed, Start: base, End: base.Add(time.Minute)},
				{Name: "wf.Mark", ShardIndex: -1, Attempt: 2, Status: workflow.StatusFailed, Start: base, End: base.Add(2 * time.Minute)},
				{Name: "wf.Mark", ShardIndex: -1, Attempt: 3, Status: workflow.StatusSucceeded, Start: base, End: base.Add(50 * time.Minute)},
			},
		},
	}
	intermediate := &workflow.Workflow{
		ID: "src-1",
		Calls: map[string][]workflow.Call{
			"wf.Mark": {{
				Name: "wf.Mark", ShardIndex: -1, Attempt: 1, Status: workflow.StatusSucceeded,
				CacheHit: true, CacheResult: "Cache Hit: src-2:wf.Mark:-1",
			}},
		},
	}
	repo := &mockWorkflowRepository{
		getMetadataFunc: func(ctx context.Context, id string) (*workflow.Workflow, error) {
			switch id {
			case "src-1":
				return intermediate, nil
			case "src-2":
				return terminal, nil
			}
			return nil, errors.New("unknown id")
		},
	}

	w := &workflow.Workflow{
		Calls: map[string][]workflow.Call{
			"wf.Mark": {{
				Name: "wf.Mark", ShardIndex: -1, Attempt: 1, Status: workflow.StatusSucceeded,
				CacheHit: true, CacheResult: "Cache Hit: src-1:wf.Mark:-1",
			}},
		},
	}

	newCacheResolver(repo).resolve(context.Background(), w)

	rec := w.Calls["wf.Mark"][0].Recovery
	if rec == nil {
		t.Fatal("expected Recovery via chain")
	}
	if rec.Attempt != 3 {
		t.Errorf("recovered attempt = %d, want 3 (terminal real run's latest attempt)", rec.Attempt)
	}
}

func TestCacheResolver_UnresolvableLeavesNil(t *testing.T) {
	repo := &mockWorkflowRepository{
		getMetadataFunc: func(ctx context.Context, id string) (*workflow.Workflow, error) {
			return nil, errors.New("source archived")
		},
	}

	w := &workflow.Workflow{
		Calls: map[string][]workflow.Call{
			"wf.Mark": {{
				Name: "wf.Mark", ShardIndex: -1, Attempt: 1, Status: workflow.StatusSucceeded,
				CacheHit: true, CacheResult: "Cache Hit: gone:wf.Mark:-1",
			}},
		},
	}

	newCacheResolver(repo).resolve(context.Background(), w)

	if w.Calls["wf.Mark"][0].Recovery != nil {
		t.Error("unresolvable source should leave Recovery nil (fallback)")
	}
}

func TestCacheResolver_MemoizesSourceFetches(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	calls := 0
	source := &workflow.Workflow{
		ID: "src-1",
		Calls: map[string][]workflow.Call{
			"wf.A": {{Name: "wf.A", ShardIndex: -1, Attempt: 1, Status: workflow.StatusSucceeded, Start: base, End: base.Add(time.Minute)}},
			"wf.B": {{Name: "wf.B", ShardIndex: -1, Attempt: 1, Status: workflow.StatusSucceeded, Start: base, End: base.Add(time.Minute)}},
		},
	}
	repo := &mockWorkflowRepository{
		getMetadataFunc: func(ctx context.Context, id string) (*workflow.Workflow, error) {
			calls++
			return source, nil
		},
	}

	// Two cache hits pointing at the same source workflow.
	w := &workflow.Workflow{
		Calls: map[string][]workflow.Call{
			"wf.A": {{Name: "wf.A", ShardIndex: -1, Attempt: 1, CacheHit: true, CacheResult: "Cache Hit: src-1:wf.A:-1"}},
			"wf.B": {{Name: "wf.B", ShardIndex: -1, Attempt: 1, CacheHit: true, CacheResult: "Cache Hit: src-1:wf.B:-1"}},
		},
	}

	newCacheResolver(repo).resolve(context.Background(), w)

	if calls != 1 {
		t.Errorf("source fetched %d times, want 1 (memoized)", calls)
	}
}
