package workflow

import (
	"context"

	"github.com/lmtani/pumbaa/internal/application/ports"
	workflow2 "github.com/lmtani/pumbaa/internal/domain/workflow"
)

// maxCacheChainDepth bounds how many cache-hit hops are followed before giving
// up, so a pathological or cyclic chain cannot loop forever.
const maxCacheChainDepth = 10

// cacheResolver recovers the real execution metrics of cache-hit calls by
// following their provenance through the workflows that actually ran them.
// It performs I/O (fetching source metadata) and therefore lives in the
// application layer; the domain only consumes the recovered values.
type cacheResolver struct {
	reader ports.WorkflowMetadataReader
	cache  map[string]*workflow2.Workflow // memoized source metadata by workflow ID
}

func newCacheResolver(reader ports.WorkflowMetadataReader) *cacheResolver {
	return &cacheResolver{
		reader: reader,
		cache:  make(map[string]*workflow2.Workflow),
	}
}

// resolve annotates the first-level calls of w with cache provenance: leaf
// cache hits get their real metrics recovered (CacheRecovery), and subworkflow
// calls whose work was entirely served from cache are flagged so the diff does
// not treat their cache-artifact wall-clock as real time. Calls whose sources
// cannot be reached are left untouched (the diff falls back to non-comparable).
func (r *cacheResolver) resolve(ctx context.Context, w *workflow2.Workflow) {
	if w == nil {
		return
	}
	for name := range w.Calls {
		for i := range w.Calls[name] {
			r.resolveCall(ctx, &w.Calls[name][i])
		}
	}
}

func (r *cacheResolver) resolveCall(ctx context.Context, call *workflow2.Call) {
	if call.CacheHit {
		src, ok := workflow2.ParseCacheResult(call.CacheResult)
		if !ok {
			return
		}
		if rec := r.follow(ctx, src, 1); rec != nil {
			call.Recovery = rec
		}
		return
	}

	if call.SubWorkflowID != "" || call.SubWorkflowMetadata != nil {
		subCalls := r.subCalls(ctx, call)
		if subCalls == nil {
			return
		}
		if served, leaves := r.cacheServed(ctx, subCalls, 1); served && leaves > 0 {
			call.SubworkflowCacheServed = true
		}
	}
}

// subCalls returns a subworkflow call's children, preferring inlined metadata
// and falling back to fetching by ID.
func (r *cacheResolver) subCalls(ctx context.Context, call *workflow2.Call) map[string][]workflow2.Call {
	if call.SubWorkflowMetadata != nil {
		return call.SubWorkflowMetadata.Calls
	}
	sw, err := r.get(ctx, call.SubWorkflowID)
	if err != nil {
		return nil
	}
	return sw.Calls
}

// cacheServed reports whether every real-work leaf reachable from calls was a
// cache hit (and there is at least one leaf), fetching nested subworkflows as
// needed and bounded by depth. It short-circuits on the first fresh leaf, so a
// workflow that did real work is cheap to reject.
func (r *cacheResolver) cacheServed(ctx context.Context, calls map[string][]workflow2.Call, depth int) (allCached bool, leaves int) {
	if depth > maxCacheChainDepth {
		return false, 0
	}
	total := 0
	for name := range calls {
		for _, c := range calls[name] {
			switch {
			case c.SubWorkflowMetadata != nil:
				ok, n := r.cacheServed(ctx, c.SubWorkflowMetadata.Calls, depth+1)
				if !ok {
					return false, 0
				}
				total += n
			case c.SubWorkflowID != "":
				sw, err := r.get(ctx, c.SubWorkflowID)
				if err != nil {
					return false, 0
				}
				ok, n := r.cacheServed(ctx, sw.Calls, depth+1)
				if !ok {
					return false, 0
				}
				total += n
			default:
				total++
				if !c.CacheHit {
					return false, 0
				}
			}
		}
	}
	return total > 0, total
}

// follow resolves a cache source to the metrics of the terminal real execution,
// following chained cache hits up to maxCacheChainDepth. It returns nil when the
// chain cannot be resolved (source missing, call missing, depth exceeded), which
// signals the diff to fall back.
func (r *cacheResolver) follow(ctx context.Context, src workflow2.CacheSource, depth int) *workflow2.CacheRecovery {
	if depth > maxCacheChainDepth {
		return nil
	}

	source, err := r.get(ctx, src.WorkflowID)
	if err != nil {
		return nil
	}

	call, ok := source.FindCall(src.CallName, src.ShardIndex)
	if !ok {
		return nil
	}

	// The source call may itself be a cache hit: follow the chain to the real
	// execution, but keep the immediate source ID this run pointed at.
	if call.CacheHit {
		next, ok := workflow2.ParseCacheResult(call.CacheResult)
		if !ok {
			return nil
		}
		rec := r.follow(ctx, next, depth+1)
		if rec == nil {
			return nil
		}
		rec.SourceWorkflowID = src.WorkflowID
		return rec
	}

	return &workflow2.CacheRecovery{
		SourceWorkflowID: src.WorkflowID,
		Start:            call.Start,
		End:              call.End,
		DockerImage:      call.DockerImage,
		Status:           call.Status,
		Attempt:          call.Attempt,
		Depth:            depth,
	}
}

// get fetches a source workflow's metadata, memoizing by ID to avoid refetching
// the same source across many cache-hit calls.
func (r *cacheResolver) get(ctx context.Context, id string) (*workflow2.Workflow, error) {
	if w, ok := r.cache[id]; ok {
		return w, nil
	}
	w, err := r.reader.GetMetadata(ctx, id)
	if err != nil {
		return nil, err
	}
	r.cache[id] = w
	return w, nil
}
