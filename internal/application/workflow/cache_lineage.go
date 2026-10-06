package workflow

import (
	"context"
	"fmt"
	"sync"

	"github.com/lmtani/pumbaa/internal/application/ports"
	workflow2 "github.com/lmtani/pumbaa/internal/domain/workflow"
)

// CacheLineageUseCase follows a cache hit through every run it was copied
// from until the run that actually executed the call, so the user lands on
// the producing run instead of walking the chain by hand.
//
// Source metadata is memoized by workflow ID for the lifetime of the use
// case: a cached call's source never changes, and in a fully cached run
// most hits point at the same few workflows. It is safe for concurrent use.
type CacheLineageUseCase struct {
	reader ports.WorkflowMetadataReader

	mu        sync.Mutex
	workflows map[string]*workflow2.Workflow
}

// NewCacheLineageUseCase creates a lineage resolver backed by reader.
func NewCacheLineageUseCase(reader ports.WorkflowMetadataReader) *CacheLineageUseCase {
	return &CacheLineageUseCase{
		reader:    reader,
		workflows: make(map[string]*workflow2.Workflow),
	}
}

// Execute resolves the lineage starting at src. It always returns the hops it
// could reach; a chain that stops short of the producing run carries the
// reason in CacheLineage.Err instead of failing, since the hops already
// reached are still worth showing.
func (uc *CacheLineageUseCase) Execute(ctx context.Context, src workflow2.CacheSource) workflow2.CacheLineage {
	var lineage workflow2.CacheLineage
	visited := make(map[workflow2.CacheSource]bool)

	for {
		if len(lineage.Hops) >= maxCacheChainDepth {
			lineage.Err = fmt.Errorf("cache chain longer than %d runs", maxCacheChainDepth)
			return lineage
		}
		if visited[src] {
			lineage.Err = fmt.Errorf("cache chain loops back to workflow %s", src.WorkflowID)
			return lineage
		}
		visited[src] = true

		wf, err := uc.get(ctx, src.WorkflowID)
		if err != nil {
			lineage.Err = fmt.Errorf("workflow %s: %w", src.WorkflowID, err)
			return lineage
		}
		call, ok := wf.FindCall(src.CallName, src.ShardIndex)
		if !ok {
			lineage.Err = fmt.Errorf("call %s (shard %d) not found in workflow %s", src.CallName, src.ShardIndex, src.WorkflowID)
			return lineage
		}

		lineage.Hops = append(lineage.Hops, workflow2.CacheHop{Workflow: wf, Call: call, Source: src})
		if !call.CacheHit {
			return lineage
		}

		next, ok := workflow2.ParseCacheResult(call.CacheResult)
		if !ok {
			lineage.Err = fmt.Errorf("unreadable cache pointer %q in workflow %s", call.CacheResult, src.WorkflowID)
			return lineage
		}
		src = next
	}
}

func (uc *CacheLineageUseCase) get(ctx context.Context, id string) (*workflow2.Workflow, error) {
	uc.mu.Lock()
	wf, ok := uc.workflows[id]
	uc.mu.Unlock()
	if ok {
		return wf, nil
	}

	// Fetch outside the lock: a duplicate concurrent fetch is harmless, a
	// screen frozen behind another fetch is not.
	wf, err := uc.reader.GetMetadata(ctx, id)
	if err != nil {
		return nil, err
	}
	uc.mu.Lock()
	uc.workflows[id] = wf
	uc.mu.Unlock()
	return wf, nil
}
