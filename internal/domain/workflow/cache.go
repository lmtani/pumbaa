package workflow

import (
	"strconv"
	"strings"
	"time"
)

// cacheHitPrefix is the leading text of a Cromwell call-cache hit result, e.g.
// "Cache Hit: 84e93adf-...:Workflow.Task:-1". A miss is reported as "Cache Miss".
const cacheHitPrefix = "Cache Hit: "

// CacheSource identifies the original call a cache hit was copied from, parsed
// from a call's CacheResult string.
type CacheSource struct {
	WorkflowID string
	CallName   string // fully-qualified call name, e.g. "Workflow.Task"
	ShardIndex int
}

// ParseCacheResult extracts the source of a cache hit from a Cromwell
// callCaching result string. It returns false for misses or any string that
// does not carry a resolvable "<workflowId>:<callFqn>:<shard>" pointer.
func ParseCacheResult(result string) (CacheSource, bool) {
	if !strings.HasPrefix(result, cacheHitPrefix) {
		return CacheSource{}, false
	}
	rest := strings.TrimPrefix(result, cacheHitPrefix)

	// WDL identifiers and UUIDs contain no ':', so a resolvable pointer splits
	// into exactly workflowId, callFqn and shardIndex.
	parts := strings.Split(rest, ":")
	if len(parts) != 3 {
		return CacheSource{}, false
	}
	shard, err := strconv.Atoi(strings.TrimSpace(parts[2]))
	if err != nil {
		return CacheSource{}, false
	}
	workflowID := strings.TrimSpace(parts[0])
	callName := strings.TrimSpace(parts[1])
	if workflowID == "" || callName == "" {
		return CacheSource{}, false
	}
	return CacheSource{WorkflowID: workflowID, CallName: callName, ShardIndex: shard}, true
}

// CacheRecovery holds the real execution metrics recovered for a cache-hit
// call by following its provenance to the workflow that actually ran it. The
// metrics come from the terminal real execution (chains of cache hits are
// followed through), while SourceWorkflowID is the immediate source the call
// pointed at.
type CacheRecovery struct {
	SourceWorkflowID string
	Start            time.Time
	End              time.Time
	DockerImage      string
	Status           Status
	Attempt          int // attempt count of the terminal real execution
	Depth            int // number of cache hops followed to reach the real run
}

// FindCall returns the call with the given fully-qualified name and shard
// index, choosing the latest attempt when several exist.
func (w *Workflow) FindCall(name string, shard int) (Call, bool) {
	calls, ok := w.Calls[name]
	if !ok {
		return Call{}, false
	}
	var best Call
	found := false
	for _, c := range calls {
		if c.ShardIndex != shard {
			continue
		}
		if !found || c.Attempt > best.Attempt {
			best = c
			found = true
		}
	}
	return best, found
}
