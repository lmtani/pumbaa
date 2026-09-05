package ports

import (
	"context"
	"errors"
	"time"
)

// ErrRunNotRemembered reports that a run has no local record. It is a normal
// answer, not a failure: most runs on a server were never submitted from this
// machine.
var ErrRunNotRemembered = errors.New("no local record for this run")

// RunOrigin says how a run came to be remembered, which is what separates
// "I started this" from "I wrote a note about someone else's run".
type RunOrigin string

const (
	// OriginSubmit marks a run submitted through pumbaa on this machine.
	OriginSubmit RunOrigin = "submit"
	// OriginNote marks a run remembered only because someone annotated it.
	OriginNote RunOrigin = "note"
)

// RunRef identifies a remembered run. A workflow ID is only unique within a
// server, so the host is part of the identity.
type RunRef struct {
	Host       string
	WorkflowID string
}

// RunRecord is what the local store remembers about a run. It deliberately
// holds what the server cannot answer later — the free-text description and
// the local files a submission was assembled from — rather than duplicating
// live state, with one exception: LastStatus, kept so history reads still say
// something useful when the server is unreachable or has forgotten the run.
type RunRecord struct {
	Host             string
	HostAlias        string
	WorkflowID       string
	Name             string
	Description      string
	WorkflowFile     string
	InputsFile       string
	OptionsFile      string
	DependenciesFile string
	Labels           map[string]string
	Origin           RunOrigin
	SubmittedAt      time.Time
	LastStatus       string
	StatusSyncedAt   time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// Ref returns the record's identity.
func (r RunRecord) Ref() RunRef {
	return RunRef{Host: r.Host, WorkflowID: r.WorkflowID}
}

// RunHistoryFilter selects records for listing. The zero value lists the most
// recent runs of every host.
type RunHistoryFilter struct {
	// Host restricts the listing to one server; empty means all of them.
	Host string
	// Search matches, case-insensitively, against the description, the
	// workflow name and the workflow ID.
	Search string
	Since  time.Time
	Limit  int
}

// RunHistoryReader reads the local run history.
type RunHistoryReader interface {
	Get(ctx context.Context, ref RunRef) (RunRecord, error)
	List(ctx context.Context, filter RunHistoryFilter) ([]RunRecord, error)
	// Lookup answers "which of these runs do we remember" in one query, so a
	// listing can be annotated without a round trip per row.
	Lookup(ctx context.Context, host string, workflowIDs []string) (map[string]RunRecord, error)
}

// RunHistoryWriter records and edits the local run history.
type RunHistoryWriter interface {
	// Record stores a run, preserving an existing description when the
	// incoming one is empty: re-recording a run must never silently erase
	// the note someone wrote about it.
	Record(ctx context.Context, rec RunRecord) error
	// SetDescription edits the note of a run already remembered, reporting
	// ErrRunNotRemembered otherwise.
	SetDescription(ctx context.Context, ref RunRef, description string) error
	// SetStatus refreshes the last known status of a remembered run.
	SetStatus(ctx context.Context, ref RunRef, status string, at time.Time) error
	Forget(ctx context.Context, ref RunRef) error
	// Prune drops records submitted before a cutoff, returning how many went.
	Prune(ctx context.Context, before time.Time) (int, error)
}

// RunHistory is the local, server-independent memory of what was submitted.
type RunHistory interface {
	RunHistoryReader
	RunHistoryWriter
	Close() error
}

// HostProvider reports the Cromwell server the current invocation is pointed
// at, which the history needs to key records by.
type HostProvider interface {
	CurrentHost() (url, alias string)
}
