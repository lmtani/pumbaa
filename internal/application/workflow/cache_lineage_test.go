package workflow

import (
	"context"
	"strings"
	"testing"

	"github.com/lmtani/pumbaa/internal/domain/workflow"
)

// lineageRepo serves workflows by ID and counts fetches.
func lineageRepo(wfs map[string]*workflow.Workflow, fetches map[string]int) *mockWorkflowRepository {
	return &mockWorkflowRepository{
		getMetadataFunc: func(_ context.Context, id string) (*workflow.Workflow, error) {
			fetches[id]++
			if wf, ok := wfs[id]; ok {
				return wf, nil
			}
			return nil, workflow.ErrWorkflowNotFound
		},
	}
}

// hitRun is a run whose wf.Amber call was copied from pointsTo.
func hitRun(id, pointsTo string) *workflow.Workflow {
	return &workflow.Workflow{ID: id, Calls: map[string][]workflow.Call{
		"wf.Amber": {{Name: "wf.Amber", ShardIndex: -1, Attempt: 1, CacheHit: true,
			CacheResult: "Cache Hit: " + pointsTo + ":wf.Amber:-1"}},
	}}
}

func ranRun(id string) *workflow.Workflow {
	return &workflow.Workflow{ID: id, Calls: map[string][]workflow.Call{
		"wf.Amber": {{Name: "wf.Amber", ShardIndex: -1, Attempt: 1, Status: workflow.StatusSucceeded}},
	}}
}

func amberIn(id string) workflow.CacheSource {
	return workflow.CacheSource{WorkflowID: id, CallName: "wf.Amber", ShardIndex: -1}
}

func TestCacheLineage_FollowsChainToProducingRun(t *testing.T) {
	fetches := map[string]int{}
	uc := NewCacheLineageUseCase(lineageRepo(map[string]*workflow.Workflow{
		"b": hitRun("b", "c"),
		"c": ranRun("c"),
	}, fetches))

	lineage := uc.Resolve(context.Background(), amberIn("b"))

	if lineage.Err != nil {
		t.Fatalf("unexpected error: %v", lineage.Err)
	}
	if len(lineage.Hops) != 2 || lineage.Hops[0].Workflow.ID != "b" || lineage.Hops[1].Workflow.ID != "c" {
		t.Fatalf("hops = %+v, want b then c", lineage.Hops)
	}
	orig, ok := lineage.Original()
	if !ok || orig.Workflow.ID != "c" {
		t.Errorf("Original = %v, %v; want c", orig.Workflow, ok)
	}

	// Sources are memoized: a second resolution fetches nothing
	uc.Resolve(context.Background(), amberIn("b"))
	if fetches["b"] != 1 || fetches["c"] != 1 {
		t.Errorf("fetches = %v, want one each", fetches)
	}
}

func TestCacheLineage_KeepsHopsReachedWhenSourceIsGone(t *testing.T) {
	uc := NewCacheLineageUseCase(lineageRepo(map[string]*workflow.Workflow{
		"b": hitRun("b", "gone"),
	}, map[string]int{}))

	lineage := uc.Resolve(context.Background(), amberIn("b"))

	if lineage.Err == nil || !strings.Contains(lineage.Err.Error(), "gone") {
		t.Fatalf("Err = %v, want it to name the missing workflow", lineage.Err)
	}
	if len(lineage.Hops) != 1 {
		t.Fatalf("hops = %d, want the one reached", len(lineage.Hops))
	}
	if _, ok := lineage.Original(); ok {
		t.Error("a broken chain has no original")
	}
	if far, ok := lineage.Furthest(); !ok || far.Workflow.ID != "b" {
		t.Error("Furthest should be the last hop reached")
	}
}

func TestCacheLineage_StopsOnLoop(t *testing.T) {
	uc := NewCacheLineageUseCase(lineageRepo(map[string]*workflow.Workflow{
		"b": hitRun("b", "c"),
		"c": hitRun("c", "b"),
	}, map[string]int{}))

	lineage := uc.Resolve(context.Background(), amberIn("b"))

	if lineage.Err == nil || !strings.Contains(lineage.Err.Error(), "loops") {
		t.Fatalf("Err = %v, want a loop error", lineage.Err)
	}
	if len(lineage.Hops) != 2 {
		t.Errorf("hops = %d, want 2", len(lineage.Hops))
	}
}

func TestCacheLineage_MissingCall(t *testing.T) {
	uc := NewCacheLineageUseCase(lineageRepo(map[string]*workflow.Workflow{
		"b": {ID: "b", Calls: map[string][]workflow.Call{}},
	}, map[string]int{}))

	lineage := uc.Resolve(context.Background(), amberIn("b"))

	if lineage.Err == nil || len(lineage.Hops) != 0 {
		t.Fatalf("lineage = %+v, want no hops and an error", lineage)
	}
}

func TestMemoReader_RetriesFailures(t *testing.T) {
	calls := 0
	memo := newMemoReader(&mockWorkflowRepository{
		getMetadataFunc: func(_ context.Context, id string) (*workflow.Workflow, error) {
			calls++
			if calls == 1 {
				return nil, workflow.ErrConnectionFailed
			}
			return &workflow.Workflow{ID: id}, nil
		},
	})

	if _, err := memo.GetMetadata(context.Background(), "a"); err == nil {
		t.Fatal("first fetch should fail")
	}
	for i := 0; i < 2; i++ {
		if wf, err := memo.GetMetadata(context.Background(), "a"); err != nil || wf.ID != "a" {
			t.Fatalf("fetch %d = %v, %v", i, wf, err)
		}
	}
	if calls != 2 {
		t.Errorf("reader called %d times, want 2 (failure retried, success memoized)", calls)
	}
}
