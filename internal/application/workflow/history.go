// history.go reads and edits the local memory of runs: what was submitted
// from this machine, and what someone wrote down about it.
package workflow

import (
	"context"
	"errors"
	"time"

	"github.com/lmtani/pumbaa/internal/application"
	"github.com/lmtani/pumbaa/internal/application/ports"
	workflow2 "github.com/lmtani/pumbaa/internal/domain/workflow"
)

// forgottenGrace is how long a run is given before its absence from a query
// is read as deletion. Cromwell answers /query from a summary table filled by
// a background summarizer, so a run submitted seconds ago is routinely missing
// from it while very much existing — claiming it was forgotten would be a
// confident lie at exactly the moment the user is watching.
const forgottenGrace = 10 * time.Minute

// RunHistoryUseCase serves the local run history. The server is consulted
// only to refresh what the local record cannot know — the current status, and
// whether the run still exists there at all.
type RunHistoryUseCase struct {
	store   ports.RunHistory
	querier ports.WorkflowQuerier
	host    ports.HostProvider
}

// NewRunHistoryUseCase creates the use case. querier may be nil, in which
// case the history is read without ever contacting a server.
func NewRunHistoryUseCase(store ports.RunHistory, querier ports.WorkflowQuerier, host ports.HostProvider) *RunHistoryUseCase {
	return &RunHistoryUseCase{store: store, querier: querier, host: host}
}

// RunHistoryEntry is a remembered run plus what the server says about it now.
type RunHistoryEntry struct {
	Record ports.RunRecord
	// Status is the run's status: refreshed from the server when it could be
	// reached, otherwise the last one seen.
	Status string
	// Stale reports that Status is the remembered one because the server was
	// not consulted or could not be reached.
	Stale bool
	// Forgotten reports that the server answered and does not know this run
	// any more — the case only a local history can cover.
	Forgotten bool
}

// ListRunHistoryInput selects what to list.
type ListRunHistoryInput struct {
	// AllHosts lists every server's runs instead of only the active one.
	// Statuses are never refreshed in this mode: the other servers are not
	// the one this invocation is connected to.
	AllHosts bool
	Search   string
	Since    time.Time
	Limit    int
	// Refresh asks the server for the current status of the listed runs.
	Refresh bool
}

// ListRunHistoryOutput is the listing plus the context needed to render it.
type ListRunHistoryOutput struct {
	Entries []RunHistoryEntry
	Host    string
	// RefreshError says why statuses could not be refreshed. It is not a
	// failure of the listing: an unreachable server is exactly when local
	// memory earns its keep.
	RefreshError error
}

// List returns remembered runs, newest first.
func (uc *RunHistoryUseCase) List(ctx context.Context, input ListRunHistoryInput) (*ListRunHistoryOutput, error) {
	host, _ := uc.host.CurrentHost()

	filter := ports.RunHistoryFilter{
		Search: input.Search,
		Since:  input.Since,
		Limit:  input.Limit,
	}
	if !input.AllHosts {
		filter.Host = host
	}

	records, err := uc.store.List(ctx, filter)
	if err != nil {
		return nil, application.NewUseCaseError("history", "failed to read the local run history", err)
	}

	out := &ListRunHistoryOutput{Host: host, Entries: make([]RunHistoryEntry, 0, len(records))}
	for _, rec := range records {
		out.Entries = append(out.Entries, RunHistoryEntry{Record: rec, Status: rec.LastStatus, Stale: true})
	}

	if input.Refresh && !input.AllHosts {
		out.RefreshError = uc.refresh(ctx, host, out.Entries)
	}
	return out, nil
}

// ShowRunHistoryOutput is a single remembered run.
type ShowRunHistoryOutput struct {
	Entry        RunHistoryEntry
	RefreshError error
}

// Show returns one remembered run, reporting ports.ErrRunNotRemembered when
// there is nothing local about it.
func (uc *RunHistoryUseCase) Show(ctx context.Context, workflowID string, refresh bool) (*ShowRunHistoryOutput, error) {
	host, _ := uc.host.CurrentHost()

	rec, err := uc.store.Get(ctx, ports.RunRef{Host: host, WorkflowID: workflowID})
	if err != nil {
		return nil, err
	}

	out := &ShowRunHistoryOutput{Entry: RunHistoryEntry{Record: rec, Status: rec.LastStatus, Stale: true}}
	if refresh {
		entries := []RunHistoryEntry{out.Entry}
		out.RefreshError = uc.refresh(ctx, host, entries)
		out.Entry = entries[0]
	}
	return out, nil
}

// Annotate writes the description of a run. A run that is not remembered yet
// is looked up on the server first, so a note can be attached to any run —
// including one someone else started.
func (uc *RunHistoryUseCase) Annotate(ctx context.Context, workflowID, description string) error {
	host, alias := uc.host.CurrentHost()
	ref := ports.RunRef{Host: host, WorkflowID: workflowID}

	// A run remembered only for its note has nothing left once the note is
	// cleared; keeping an empty row would put a marker on a listing for no
	// reason. A submitted run keeps its record: the files it ran with are
	// worth remembering on their own.
	if description == "" {
		if rec, getErr := uc.store.Get(ctx, ref); getErr == nil && rec.Origin == ports.OriginNote {
			return uc.store.Forget(ctx, ref)
		}
	}

	err := uc.store.SetDescription(ctx, ref, description)
	if err == nil {
		return nil
	}
	if !errors.Is(err, ports.ErrRunNotRemembered) {
		return application.NewUseCaseError("history", "failed to save the description", err)
	}

	if uc.querier == nil {
		return err
	}
	found, queryErr := uc.lookupOnServer(ctx, []string{workflowID})
	if queryErr != nil {
		return application.NewUseCaseError("history", "run is not in the local history and the server could not be reached", queryErr)
	}
	wf, ok := found[workflowID]
	if !ok {
		return application.NewUseCaseError("history", "no such run on this server", workflow2.ErrWorkflowNotFound)
	}

	return uc.store.Record(ctx, ports.RunRecord{
		Host:        host,
		HostAlias:   alias,
		WorkflowID:  workflowID,
		Name:        wf.Name,
		Description: description,
		Labels:      wf.Labels,
		Origin:      ports.OriginNote,
		SubmittedAt: wf.SubmittedAt,
		LastStatus:  string(wf.Status),
		// The status came from the server just now, so it is as fresh as the
		// record itself.
		StatusSyncedAt: time.Now().UTC(),
	})
}

// Forget drops a run from the local history.
func (uc *RunHistoryUseCase) Forget(ctx context.Context, workflowID string) error {
	host, _ := uc.host.CurrentHost()
	return uc.store.Forget(ctx, ports.RunRef{Host: host, WorkflowID: workflowID})
}

// Prune drops every run submitted before the cutoff, across all hosts.
func (uc *RunHistoryUseCase) Prune(ctx context.Context, before time.Time) (int, error) {
	return uc.store.Prune(ctx, before)
}

// Marks looks up which of the given runs are remembered on the active host,
// so a listing can be annotated in a single query.
func (uc *RunHistoryUseCase) Marks(ctx context.Context, workflowIDs []string) (map[string]ports.RunRecord, error) {
	host, _ := uc.host.CurrentHost()
	return uc.store.Lookup(ctx, host, workflowIDs)
}

// refresh replaces remembered statuses with current ones, in a single query,
// and writes what it learns back so the next offline read is closer to the
// truth. Entries are updated in place.
func (uc *RunHistoryUseCase) refresh(ctx context.Context, host string, entries []RunHistoryEntry) error {
	if uc.querier == nil || len(entries) == 0 {
		return nil
	}

	ids := make([]string, 0, len(entries))
	for _, e := range entries {
		ids = append(ids, e.Record.WorkflowID)
	}
	live, err := uc.lookupOnServer(ctx, ids)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	for i := range entries {
		wf, ok := live[entries[i].Record.WorkflowID]
		if !ok {
			// The server answered without this run. Old enough, that means it
			// was forgotten there — precisely what the local record survives.
			// Recent enough, it just has not been summarized yet, and the
			// honest answer is that the status is the last one seen.
			if now.Sub(entries[i].Record.SubmittedAt) > forgottenGrace {
				entries[i].Forgotten = true
				entries[i].Stale = false
			}
			continue
		}
		entries[i].Status = string(wf.Status)
		entries[i].Stale = false
		if entries[i].Record.LastStatus != string(wf.Status) {
			ref := ports.RunRef{Host: host, WorkflowID: entries[i].Record.WorkflowID}
			// A failed write only costs freshness on the next offline read.
			_ = uc.store.SetStatus(ctx, ref, string(wf.Status), now)
		}
	}
	return nil
}

// lookupOnServer asks the server about specific runs in one query.
func (uc *RunHistoryUseCase) lookupOnServer(ctx context.Context, ids []string) (map[string]workflow2.Workflow, error) {
	result, err := uc.querier.Query(ctx, workflow2.QueryFilter{IDs: ids, PageSize: len(ids)})
	if err != nil {
		return nil, err
	}
	found := make(map[string]workflow2.Workflow, len(result.Workflows))
	for _, wf := range result.Workflows {
		found[wf.ID] = wf
	}
	return found, nil
}
