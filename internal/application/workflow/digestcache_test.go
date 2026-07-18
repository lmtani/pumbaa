package workflow

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/lmtani/pumbaa/internal/application/ports"
)

// countingProvider records how many times each path was actually fetched.
type countingProvider struct {
	mu     sync.Mutex
	counts map[string]int
	fail   map[string]bool
}

func (p *countingProvider) Read(context.Context, string) (string, error) { return "", nil }
func (p *countingProvider) ReadBytes(context.Context, string) ([]byte, error) {
	return nil, nil
}
func (p *countingProvider) GetSize(context.Context, string) (int64, error) { return 0, nil }
func (p *countingProvider) GetContentDigests(_ context.Context, path string) (ports.FileDigests, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.counts[path]++
	if p.fail[path] {
		return ports.FileDigests{}, errors.New("unreachable")
	}
	return ports.FileDigests{MD5: "d41d8cd98f00b204e9800998ecf8427e"}, nil
}

// The same file is read by many calls, and each read is a network round trip.
// Fetching it once is the difference between a command that takes seconds and
// one that takes a minute.
func TestDigestCacheFetchesEachPathOnce(t *testing.T) {
	inner := &countingProvider{counts: map[string]int{}}
	cache := newDigestCache(inner)

	for range 5 {
		if _, err := cache.GetContentDigests(context.Background(), "gs://b/ref.fa"); err != nil {
			t.Fatalf("GetContentDigests() error: %v", err)
		}
	}

	if got := inner.counts["gs://b/ref.fa"]; got != 1 {
		t.Errorf("fetched %d times, want 1", got)
	}
}

// A failure is an answer too: retrying it costs another round trip to learn the
// same thing.
func TestDigestCacheRemembersFailures(t *testing.T) {
	inner := &countingProvider{counts: map[string]int{}, fail: map[string]bool{"gs://b/gone": true}}
	cache := newDigestCache(inner)

	for range 3 {
		if _, err := cache.GetContentDigests(context.Background(), "gs://b/gone"); err == nil {
			t.Fatal("expected the failure to be reported every time")
		}
	}
	if got := inner.counts["gs://b/gone"]; got != 1 {
		t.Errorf("fetched %d times, want 1", got)
	}
}

func TestDigestCacheWarmLeavesNothingToFetch(t *testing.T) {
	inner := &countingProvider{counts: map[string]int{}}
	cache := newDigestCache(inner)
	paths := []string{"gs://b/a", "gs://b/b", "gs://b/c"}

	cache.warm(context.Background(), paths)
	for _, p := range paths {
		if _, err := cache.GetContentDigests(context.Background(), p); err != nil {
			t.Fatalf("GetContentDigests(%s) error: %v", p, err)
		}
	}

	for _, p := range paths {
		if got := inner.counts[p]; got != 1 {
			t.Errorf("%s fetched %d times, want 1", p, got)
		}
	}
}

// Prefetching is a guess and must stay one: it picks what looks like a file and
// never decides anything the comparison depends on.
func TestPathsWorthPrefetchingPicksFileLikeValues(t *testing.T) {
	got := pathsWorthPrefetching(map[string]any{
		"W.sample": "NA12878",
		"W.ref":    "gs://bucket/ref.fa",
		"W.local":  "/data/in.bam",
		"W.count":  float64(4),
		"W.bams":   []any{"gs://bucket/a.bam", "gs://bucket/b.bam"},
		"W.nested": map[string]any{"index": "/data/in.bai"},
		"W.flag":   true,
		"W.dupe":   "gs://bucket/ref.fa",
	})

	want := map[string]bool{
		"gs://bucket/ref.fa": true,
		"/data/in.bam":       true,
		"gs://bucket/a.bam":  true,
		"gs://bucket/b.bam":  true,
		"/data/in.bai":       true,
	}
	if len(got) != len(want) {
		t.Fatalf("got %v, want the %d file-like values once each", got, len(want))
	}
	for _, p := range got {
		if !want[p] {
			t.Errorf("%q is not a file-like value", p)
		}
	}
}
