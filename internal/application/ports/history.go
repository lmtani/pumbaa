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
// the files a submission was assembled from — rather than duplicating live
// state, with one exception: LastStatus, kept so history reads still say
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
	// Files are the submitted files kept verbatim, which is what lets a run
	// be exported and submitted again after the paths above have rotted.
	// Reads fill in everything but the bytes — a listing needs sizes, not
	// contents — and RunHistoryReader.Files returns them. On a Record, nil
	// leaves whatever is already kept alone; anything else replaces it.
	Files []RunFile
}

// RunFileKind names one of the files a submission is assembled from.
type RunFileKind string

const (
	RunFileWorkflow     RunFileKind = "workflow"
	RunFileInputs       RunFileKind = "inputs"
	RunFileOptions      RunFileKind = "options"
	RunFileDependencies RunFileKind = "dependencies"
)

// RunFileKinds is every kind, in the order a submission names them, which is
// the order anything listing kept files should use.
var RunFileKinds = []RunFileKind{RunFileWorkflow, RunFileInputs, RunFileOptions, RunFileDependencies}

// RunFile is one submitted file, as the server received it. Name is the
// basename it was read from, kept for recognition only: an export writes
// fixed names so the printed submit command is always right.
type RunFile struct {
	Kind   RunFileKind
	Name   string
	SHA256 string
	Size   int64
	Data   []byte `json:"Data,omitempty"`
}

// PruneResult says what a prune removed. Bytes counts the kept files that
// went with the runs, which is what the person asking to free space wants
// to hear.
type PruneResult struct {
	Runs  int
	Bytes int64
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
	// Search matches keywords case-insensitively: every whitespace-separated
	// term must appear somewhere in the record.
	Search string
	// Status keeps only runs whose last known status matches. That status is
	// the one recorded locally, which can lag the server.
	Status string
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
	// Files returns the kept files of a run with their bytes, or an empty
	// slice when nothing was kept. It reports ErrRunNotRemembered when the
	// run itself is unknown.
	Files(ctx context.Context, ref RunRef) ([]RunFile, error)
}

// RunHistoryWriter records and edits the local run history.
type RunHistoryWriter interface {
	// Record stores a run, preserving an existing description when the
	// incoming one is empty and the kept files when rec.Files is nil:
	// re-recording a run must never silently erase the note someone wrote
	// about it, nor the files only this machine still has.
	Record(ctx context.Context, rec RunRecord) error
	// SetDescription edits the note of a run already remembered, reporting
	// ErrRunNotRemembered otherwise.
	SetDescription(ctx context.Context, ref RunRef, description string) error
	// SetStatus refreshes the last known status of a remembered run.
	SetStatus(ctx context.Context, ref RunRef, status string, at time.Time) error
	Forget(ctx context.Context, ref RunRef) error
	// Prune drops records submitted before a cutoff, and their files with
	// them, returning what went.
	Prune(ctx context.Context, before time.Time) (PruneResult, error)
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
