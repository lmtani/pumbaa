package workflow

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lmtani/pumbaa/internal/application/ports"
	"github.com/lmtani/pumbaa/internal/domain/workflow"
)

// stubRunHistory is an in-memory stand-in for the local store.
type stubRunHistory struct {
	ports.RunHistory
	records   []ports.RunRecord
	statusSet map[string]string
	lastQuery ports.RunHistoryFilter
}

func newStubRunHistory(records ...ports.RunRecord) *stubRunHistory {
	return &stubRunHistory{records: records, statusSet: map[string]string{}}
}

func (s *stubRunHistory) List(_ context.Context, filter ports.RunHistoryFilter) ([]ports.RunRecord, error) {
	s.lastQuery = filter
	if filter.Host == "" {
		return s.records, nil
	}
	var out []ports.RunRecord
	for _, rec := range s.records {
		if rec.Host == filter.Host {
			out = append(out, rec)
		}
	}
	return out, nil
}

func (s *stubRunHistory) Get(_ context.Context, ref ports.RunRef) (ports.RunRecord, error) {
	for _, rec := range s.records {
		if rec.Host == ref.Host && rec.WorkflowID == ref.WorkflowID {
			return rec, nil
		}
	}
	return ports.RunRecord{}, ports.ErrRunNotRemembered
}

func (s *stubRunHistory) SetDescription(_ context.Context, ref ports.RunRef, description string) error {
	for i := range s.records {
		if s.records[i].Host == ref.Host && s.records[i].WorkflowID == ref.WorkflowID {
			s.records[i].Description = description
			return nil
		}
	}
	return ports.ErrRunNotRemembered
}

func (s *stubRunHistory) SetStatus(_ context.Context, ref ports.RunRef, status string, _ time.Time) error {
	s.statusSet[ref.WorkflowID] = status
	return nil
}

func (s *stubRunHistory) Forget(_ context.Context, ref ports.RunRef) error {
	for i := range s.records {
		if s.records[i].Host == ref.Host && s.records[i].WorkflowID == ref.WorkflowID {
			s.records = append(s.records[:i], s.records[i+1:]...)
			return nil
		}
	}
	return ports.ErrRunNotRemembered
}

func (s *stubRunHistory) Record(_ context.Context, rec ports.RunRecord) error {
	s.records = append(s.records, rec)
	return nil
}

// historyQuerier answers only about the runs it was given.
type historyQuerier struct {
	known     map[string]workflow.Workflow
	err       error
	lastQuery workflow.QueryFilter
	calls     int
}

func (s *historyQuerier) Query(_ context.Context, filter workflow.QueryFilter) (*workflow.QueryResult, error) {
	s.calls++
	s.lastQuery = filter
	if s.err != nil {
		return nil, s.err
	}
	result := &workflow.QueryResult{}
	for _, id := range filter.IDs {
		if wf, ok := s.known[id]; ok {
			result.Workflows = append(result.Workflows, wf)
		}
	}
	result.TotalCount = len(result.Workflows)
	return result, nil
}

func historyRecord(host, id, status string) ports.RunRecord {
	return ports.RunRecord{
		Host:        host,
		HostAlias:   "local",
		WorkflowID:  id,
		Name:        "HelloWorld",
		Description: "note for " + id,
		Origin:      ports.OriginSubmit,
		SubmittedAt: time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC),
		LastStatus:  status,
	}
}

func TestListScopesToTheActiveHost(t *testing.T) {
	store := newStubRunHistory(
		historyRecord("http://localhost:8000", "here", "Running"),
		historyRecord("https://other", "there", "Running"),
	)
	uc := NewRunHistoryUseCase(store, nil, fakeHostProvider{url: "http://localhost:8000"})

	out, err := uc.List(context.Background(), ListRunHistoryInput{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(out.Entries) != 1 || out.Entries[0].Record.WorkflowID != "here" {
		t.Errorf("got %+v, want only the active host's run", out.Entries)
	}

	all, err := uc.List(context.Background(), ListRunHistoryInput{AllHosts: true})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all.Entries) != 2 {
		t.Errorf("got %d runs, want every host's", len(all.Entries))
	}
	if store.lastQuery.Host != "" {
		t.Errorf("host filter = %q, want it unset when listing every host", store.lastQuery.Host)
	}
}

func TestListRefreshesStatusesInOneQuery(t *testing.T) {
	store := newStubRunHistory(
		historyRecord("http://localhost:8000", "run-1", "Running"),
		historyRecord("http://localhost:8000", "run-2", "Running"),
	)
	querier := &historyQuerier{known: map[string]workflow.Workflow{
		"run-1": {ID: "run-1", Status: workflow.StatusSucceeded},
		"run-2": {ID: "run-2", Status: workflow.StatusRunning},
	}}
	uc := NewRunHistoryUseCase(store, querier, fakeHostProvider{url: "http://localhost:8000"})

	out, err := uc.List(context.Background(), ListRunHistoryInput{Refresh: true})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if querier.calls != 1 {
		t.Errorf("queried the server %d times, want a single batched call", querier.calls)
	}
	if len(querier.lastQuery.IDs) != 2 {
		t.Errorf("query IDs = %v, want the listed runs so the server does not dump its whole database", querier.lastQuery.IDs)
	}
	if out.Entries[0].Status != string(workflow.StatusSucceeded) || out.Entries[0].Stale {
		t.Errorf("got %+v, want the refreshed status", out.Entries[0])
	}
	if store.statusSet["run-1"] != string(workflow.StatusSucceeded) {
		t.Errorf("stored status = %q, want the refresh written back for offline reads", store.statusSet["run-1"])
	}
	if _, written := store.statusSet["run-2"]; written {
		t.Error("wrote back an unchanged status, want the write skipped")
	}
}

func TestListMarksRunsTheServerForgot(t *testing.T) {
	store := newStubRunHistory(historyRecord("http://localhost:8000", "ghost", "Succeeded"))
	querier := &historyQuerier{known: map[string]workflow.Workflow{}}
	uc := NewRunHistoryUseCase(store, querier, fakeHostProvider{url: "http://localhost:8000"})

	out, err := uc.List(context.Background(), ListRunHistoryInput{Refresh: true})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if !out.Entries[0].Forgotten {
		t.Error("Forgotten = false, want the run marked once the server denies knowing it")
	}
	if out.Entries[0].Status != "Succeeded" {
		t.Errorf("status = %q, want the last known one kept", out.Entries[0].Status)
	}
}

func TestListWaitsBeforeCallingAFreshRunForgotten(t *testing.T) {
	fresh := historyRecord("http://localhost:8000", "just-submitted", "Submitted")
	fresh.SubmittedAt = time.Now().Add(-30 * time.Second)
	store := newStubRunHistory(fresh)
	// Cromwell answers /query from a summary table a background job fills, so
	// a run submitted seconds ago is routinely missing from it.
	querier := &historyQuerier{known: map[string]workflow.Workflow{}}
	uc := NewRunHistoryUseCase(store, querier, fakeHostProvider{url: "http://localhost:8000"})

	out, err := uc.List(context.Background(), ListRunHistoryInput{Refresh: true})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if out.Entries[0].Forgotten {
		t.Error("a run submitted seconds ago was declared forgotten; the query index simply lags")
	}
	if !out.Entries[0].Stale {
		t.Error("Stale = false, want the unconfirmed status flagged as the last known one")
	}
	if out.Entries[0].Status != "Submitted" {
		t.Errorf("status = %q, want the last known one kept", out.Entries[0].Status)
	}
}

func TestListSurvivesAnUnreachableServer(t *testing.T) {
	store := newStubRunHistory(historyRecord("http://localhost:8000", "run-1", "Running"))
	querier := &historyQuerier{err: workflow.ErrConnectionFailed}
	uc := NewRunHistoryUseCase(store, querier, fakeHostProvider{url: "http://localhost:8000"})

	out, err := uc.List(context.Background(), ListRunHistoryInput{Refresh: true})
	if err != nil {
		t.Fatalf("List returned %v; an unreachable server is when local memory matters most", err)
	}
	if len(out.Entries) != 1 || !out.Entries[0].Stale {
		t.Errorf("got %+v, want the remembered status flagged as stale", out.Entries)
	}
	if out.Entries[0].Forgotten {
		t.Error("Forgotten = true, want a connection failure not to be read as deletion")
	}
	if out.RefreshError == nil {
		t.Error("RefreshError = nil, want the reason reported")
	}
}

// slowQuerier stands in for a server that accepted the connection and then
// went quiet, which is what a dropped VPN looks like.
type slowQuerier struct{}

func (slowQuerier) Query(ctx context.Context, _ workflow.QueryFilter) (*workflow.QueryResult, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestListDoesNotWaitOnASilentServer(t *testing.T) {
	store := newStubRunHistory(historyRecord("http://localhost:8000", "run-1", "Running"))
	uc := NewRunHistoryUseCase(store, slowQuerier{}, fakeHostProvider{url: "http://localhost:8000"})

	start := time.Now()
	out, err := uc.List(context.Background(), ListRunHistoryInput{Refresh: true})
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if elapsed > 2*refreshTimeout {
		t.Errorf("took %v to answer from disk; the refresh deadline (%v) did not apply", elapsed, refreshTimeout)
	}
	if len(out.Entries) != 1 || out.Entries[0].Status != "Running" {
		t.Errorf("got %+v, want the remembered record returned anyway", out.Entries)
	}
	if out.RefreshError == nil {
		t.Error("RefreshError = nil, want the timeout reported")
	}
	if out.Entries[0].Forgotten {
		t.Error("Forgotten = true, want a silent server not to be read as deletion")
	}
}

func TestAnnotateEditsARememberedRun(t *testing.T) {
	store := newStubRunHistory(historyRecord("http://localhost:8000", "run-1", "Running"))
	querier := &historyQuerier{}
	uc := NewRunHistoryUseCase(store, querier, fakeHostProvider{url: "http://localhost:8000"})

	if err := uc.Annotate(context.Background(), "run-1", "the rerun with the fixed reference"); err != nil {
		t.Fatalf("Annotate: %v", err)
	}
	if store.records[0].Description != "the rerun with the fixed reference" {
		t.Errorf("description = %q, want the new note", store.records[0].Description)
	}
	if querier.calls != 0 {
		t.Error("queried the server to edit a run already remembered")
	}
}

func TestAnnotateAdoptsARunFromTheServer(t *testing.T) {
	store := newStubRunHistory()
	submitted := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
	querier := &historyQuerier{known: map[string]workflow.Workflow{
		"someone-elses": {
			ID:          "someone-elses",
			Name:        "TSO500",
			Status:      workflow.StatusSucceeded,
			SubmittedAt: submitted,
			Labels:      map[string]string{"owner": "someone"},
		},
	}}
	uc := NewRunHistoryUseCase(store, querier, fakeHostProvider{url: "http://localhost:8000", alias: "prod"})

	if err := uc.Annotate(context.Background(), "someone-elses", "run that produced the reference panel"); err != nil {
		t.Fatalf("Annotate: %v", err)
	}
	if len(store.records) != 1 {
		t.Fatalf("stored %d records, want the run adopted", len(store.records))
	}

	rec := store.records[0]
	if rec.Origin != ports.OriginNote {
		t.Errorf("origin = %q, want %q for a run not submitted here", rec.Origin, ports.OriginNote)
	}
	if rec.Name != "TSO500" || !rec.SubmittedAt.Equal(submitted) {
		t.Errorf("got %+v, want the server's name and submission time rather than invented ones", rec)
	}
	if rec.HostAlias != "prod" {
		t.Errorf("host alias = %q, want the active host's", rec.HostAlias)
	}
	if rec.LastStatus != string(workflow.StatusSucceeded) {
		t.Errorf("status = %q, want the server's", rec.LastStatus)
	}
}

func TestAnnotateRejectsARunNobodyKnows(t *testing.T) {
	store := newStubRunHistory()
	uc := NewRunHistoryUseCase(store, &historyQuerier{known: map[string]workflow.Workflow{}}, fakeHostProvider{url: "http://localhost:8000"})

	err := uc.Annotate(context.Background(), "ghost", "note")
	if err == nil {
		t.Fatal("Annotate succeeded, want a refusal to invent a run")
	}
	if !errors.Is(err, workflow.ErrWorkflowNotFound) {
		t.Errorf("got %v, want it to report the run does not exist", err)
	}
	if len(store.records) != 0 {
		t.Error("stored a record for a run that does not exist")
	}
}

func TestAnnotateEmptyDropsAnAnnotationOnlyRun(t *testing.T) {
	annotated := historyRecord("http://localhost:8000", "annotated", "Succeeded")
	annotated.Origin = ports.OriginNote
	submitted := historyRecord("http://localhost:8000", "submitted", "Succeeded")
	store := newStubRunHistory(annotated, submitted)
	uc := NewRunHistoryUseCase(store, &historyQuerier{}, fakeHostProvider{url: "http://localhost:8000"})

	if err := uc.Annotate(context.Background(), "annotated", ""); err != nil {
		t.Fatalf("Annotate: %v", err)
	}
	if _, err := store.Get(context.Background(), ports.RunRef{Host: "http://localhost:8000", WorkflowID: "annotated"}); !errors.Is(err, ports.ErrRunNotRemembered) {
		t.Error("a run remembered only for its note survived the note being cleared")
	}

	if err := uc.Annotate(context.Background(), "submitted", ""); err != nil {
		t.Fatalf("Annotate: %v", err)
	}
	rec, err := store.Get(context.Background(), ports.RunRef{Host: "http://localhost:8000", WorkflowID: "submitted"})
	if err != nil {
		t.Fatalf("a submitted run lost its record when its note was cleared: %v", err)
	}
	if rec.Description != "" {
		t.Errorf("description = %q, want it cleared", rec.Description)
	}
}
