package workflow

import (
	"context"
	"sync"

	"github.com/lmtani/pumbaa/internal/application/ports"
	workflow2 "github.com/lmtani/pumbaa/internal/domain/workflow"
)

// memoReader memoizes workflow metadata by ID. Following cache hits fetches
// the same few source workflows over and over, and a source's metadata no
// longer changes for the calls that were copied from it. It is safe for
// concurrent use; entries live as long as the memoReader does.
type memoReader struct {
	reader ports.WorkflowMetadataReader

	mu   sync.Mutex
	byID map[string]*workflow2.Workflow
}

var _ ports.WorkflowMetadataReader = (*memoReader)(nil)

func newMemoReader(reader ports.WorkflowMetadataReader) *memoReader {
	return &memoReader{reader: reader, byID: make(map[string]*workflow2.Workflow)}
}

// GetMetadata returns the memoized metadata, fetching it on first use.
// Failures are not memoized, so a later call retries.
func (r *memoReader) GetMetadata(ctx context.Context, id string) (*workflow2.Workflow, error) {
	r.mu.Lock()
	wf, ok := r.byID[id]
	r.mu.Unlock()
	if ok {
		return wf, nil
	}

	// Fetch outside the lock: a duplicate concurrent fetch is harmless, a
	// caller blocked behind someone else's fetch is not.
	wf, err := r.reader.GetMetadata(ctx, id)
	if err != nil {
		return nil, err
	}
	r.mu.Lock()
	r.byID[id] = wf
	r.mu.Unlock()
	return wf, nil
}
