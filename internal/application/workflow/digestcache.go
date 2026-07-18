package workflow

import (
	"context"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/lmtani/pumbaa/internal/application/ports"
)

// digestPrefetch bounds how many objects are inspected at once. The work is
// entirely latency, so concurrency buys almost linearly; the cap is there to
// stay polite to the storage API rather than because more would not help.
const digestPrefetch = 16

// digestCache memoises content digests over a file provider.
//
// Two properties of the workload make this worth its weight. The same file is
// read by many calls — a reference bundle shared across a dozen tasks — so
// lookups repeat; and each lookup is a network round trip, so repeating one
// costs far more than remembering it. Warming the cache up front also turns a
// sequence of round trips into a single concurrent sweep.
type digestCache struct {
	inner ports.FileProvider

	mu      sync.Mutex
	entries map[string]digestEntry
}

type digestEntry struct {
	digests ports.FileDigests
	err     error
}

func newDigestCache(inner ports.FileProvider) *digestCache {
	return &digestCache{inner: inner, entries: make(map[string]digestEntry)}
}

func (c *digestCache) Read(ctx context.Context, path string) (string, error) {
	return c.inner.Read(ctx, path)
}

func (c *digestCache) ReadBytes(ctx context.Context, path string) ([]byte, error) {
	return c.inner.ReadBytes(ctx, path)
}

func (c *digestCache) GetSize(ctx context.Context, path string) (int64, error) {
	return c.inner.GetSize(ctx, path)
}

// GetContentDigests answers from the cache when it can, including for a path
// that previously failed: a second attempt would fail the same way and cost
// another round trip to prove it.
func (c *digestCache) GetContentDigests(ctx context.Context, path string) (ports.FileDigests, error) {
	c.mu.Lock()
	entry, cached := c.entries[path]
	c.mu.Unlock()
	if cached {
		return entry.digests, entry.err
	}

	digests, err := c.inner.GetContentDigests(ctx, path)

	c.mu.Lock()
	c.entries[path] = digestEntry{digests: digests, err: err}
	c.mu.Unlock()
	return digests, err
}

// warm fetches many paths at once so the comparison that follows finds them
// already in hand. Failures are cached like any other answer — the comparison
// reports them in context, and warming is not the place to decide what a
// missing file means.
func (c *digestCache) warm(ctx context.Context, paths []string) {
	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(digestPrefetch)
	for _, path := range paths {
		group.Go(func() error {
			_, _ = c.GetContentDigests(groupCtx, path)
			return nil
		})
	}
	_ = group.Wait()
}

// pathsWorthPrefetching picks the values in a submission document that look like
// files, since those are the only ones a comparison will hash.
//
// It is a guess, and only a guess: prefetching is an optimisation, so a value
// wrongly included costs one lookup and a value wrongly left out is fetched
// later on demand. Correctness never rests on it.
func pathsWorthPrefetching(params map[string]any) []string {
	seen := make(map[string]bool)
	var out []string
	var consider func(value any)
	consider = func(value any) {
		switch v := value.(type) {
		case string:
			if looksLikePath(v) && !seen[v] {
				seen[v] = true
				out = append(out, v)
			}
		case []any:
			for _, item := range v {
				consider(item)
			}
		case map[string]any:
			for _, item := range v {
				consider(item)
			}
		}
	}
	for _, value := range params {
		consider(value)
	}
	return out
}

// looksLikePath keeps identifiers and flags out of the sweep.
func looksLikePath(value string) bool {
	return strings.Contains(value, "://") || strings.HasPrefix(value, "/")
}

var _ ports.FileProvider = (*digestCache)(nil)
