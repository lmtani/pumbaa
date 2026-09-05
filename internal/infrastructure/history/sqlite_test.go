package history

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/lmtani/pumbaa/internal/application/ports"
)

const testHost = "http://localhost:8000"

func newTestStore(t *testing.T) *Store {
	t.Helper()
	store := New(filepath.Join(t.TempDir(), "history.db"))
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func sampleRecord(id string) ports.RunRecord {
	return ports.RunRecord{
		Host:         testHost,
		HostAlias:    "local",
		WorkflowID:   id,
		Name:         "HelloWorld",
		Description:  "smoke test",
		WorkflowFile: "/wf/hello.wdl",
		InputsFile:   "/wf/hello.inputs.json",
		Labels:       map[string]string{"owner": "lucas"},
		Origin:       ports.OriginSubmit,
		SubmittedAt:  time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC),
		LastStatus:   "Submitted",
	}
}

func TestRecordAndGetRoundTrip(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	want := sampleRecord("abc-123")

	if err := store.Record(ctx, want); err != nil {
		t.Fatalf("Record: %v", err)
	}

	got, err := store.Get(ctx, want.Ref())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Description != want.Description || got.Name != want.Name {
		t.Errorf("got %+v, want description/name preserved", got)
	}
	if !got.SubmittedAt.Equal(want.SubmittedAt) {
		t.Errorf("SubmittedAt = %v, want %v", got.SubmittedAt, want.SubmittedAt)
	}
	if got.Labels["owner"] != "lucas" {
		t.Errorf("labels = %v, want the submitted labels", got.Labels)
	}
	if got.Origin != ports.OriginSubmit {
		t.Errorf("origin = %q, want %q", got.Origin, ports.OriginSubmit)
	}
	if got.WorkflowFile != want.WorkflowFile || got.InputsFile != want.InputsFile {
		t.Errorf("got %+v, want the local file paths preserved", got)
	}
}

func TestGetUnknownRunIsNotAnError(t *testing.T) {
	store := newTestStore(t)

	_, err := store.Get(context.Background(), ports.RunRef{Host: testHost, WorkflowID: "nope"})
	if !errors.Is(err, ports.ErrRunNotRemembered) {
		t.Fatalf("got %v, want ErrRunNotRemembered", err)
	}
}

func TestRecordKeepsExistingDescription(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	rec := sampleRecord("abc-123")
	if err := store.Record(ctx, rec); err != nil {
		t.Fatalf("Record: %v", err)
	}

	// A later record without a description must not erase the note.
	rec.Description = ""
	rec.LastStatus = "Running"
	if err := store.Record(ctx, rec); err != nil {
		t.Fatalf("Record: %v", err)
	}

	got, err := store.Get(ctx, rec.Ref())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Description != "smoke test" {
		t.Errorf("description = %q, want the original note preserved", got.Description)
	}
	if got.LastStatus != "Running" {
		t.Errorf("last status = %q, want the new one", got.LastStatus)
	}
}

func TestRunsAreScopedByHost(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	local := sampleRecord("same-id")
	remote := sampleRecord("same-id")
	remote.Host = "https://cromwell.example.com"
	remote.Description = "the other server"
	for _, rec := range []ports.RunRecord{local, remote} {
		if err := store.Record(ctx, rec); err != nil {
			t.Fatalf("Record: %v", err)
		}
	}

	got, err := store.Get(ctx, remote.Ref())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Description != "the other server" {
		t.Errorf("description = %q, want the record of the host asked for", got.Description)
	}

	all, err := store.List(ctx, ports.RunHistoryFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("listed %d runs, want both hosts kept apart", len(all))
	}
}

func TestSetDescriptionRequiresAnExistingRun(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	ref := ports.RunRef{Host: testHost, WorkflowID: "abc-123"}

	if err := store.SetDescription(ctx, ref, "note"); !errors.Is(err, ports.ErrRunNotRemembered) {
		t.Fatalf("got %v, want ErrRunNotRemembered", err)
	}

	if err := store.Record(ctx, sampleRecord("abc-123")); err != nil {
		t.Fatalf("Record: %v", err)
	}
	if err := store.SetDescription(ctx, ref, "rerun with the fixed reference"); err != nil {
		t.Fatalf("SetDescription: %v", err)
	}
	got, _ := store.Get(ctx, ref)
	if got.Description != "rerun with the fixed reference" {
		t.Errorf("description = %q, want the edited note", got.Description)
	}
}

func TestSetStatusStampsSyncTime(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	rec := sampleRecord("abc-123")
	if err := store.Record(ctx, rec); err != nil {
		t.Fatalf("Record: %v", err)
	}

	syncedAt := time.Date(2026, 7, 2, 8, 30, 0, 0, time.UTC)
	if err := store.SetStatus(ctx, rec.Ref(), "Succeeded", syncedAt); err != nil {
		t.Fatalf("SetStatus: %v", err)
	}

	got, _ := store.Get(ctx, rec.Ref())
	if got.LastStatus != "Succeeded" {
		t.Errorf("status = %q, want Succeeded", got.LastStatus)
	}
	if !got.StatusSyncedAt.Equal(syncedAt) {
		t.Errorf("StatusSyncedAt = %v, want %v", got.StatusSyncedAt, syncedAt)
	}
}

func TestListFilters(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	old := sampleRecord("old-run")
	old.Description = "reference build"
	old.SubmittedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	recent := sampleRecord("recent-run")
	recent.Description = "TSO500 rerun"
	recent.SubmittedAt = time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	other := sampleRecord("other-host-run")
	other.Host = "https://cromwell.example.com"
	other.SubmittedAt = time.Date(2026, 7, 21, 0, 0, 0, 0, time.UTC)
	for _, rec := range []ports.RunRecord{old, recent, other} {
		if err := store.Record(ctx, rec); err != nil {
			t.Fatalf("Record: %v", err)
		}
	}

	all, err := store.List(ctx, ports.RunHistoryFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 3 || all[0].WorkflowID != "other-host-run" {
		t.Errorf("got %d runs starting with %q, want 3 newest-first", len(all), all[0].WorkflowID)
	}

	byHost, err := store.List(ctx, ports.RunHistoryFilter{Host: testHost})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(byHost) != 2 {
		t.Errorf("host filter returned %d runs, want 2", len(byHost))
	}

	bySearch, err := store.List(ctx, ports.RunHistoryFilter{Search: "tso500"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(bySearch) != 1 || bySearch[0].WorkflowID != "recent-run" {
		t.Errorf("search returned %+v, want the case-insensitive description match", bySearch)
	}

	bySince, err := store.List(ctx, ports.RunHistoryFilter{Since: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(bySince) != 2 {
		t.Errorf("since filter returned %d runs, want 2", len(bySince))
	}

	byLimit, err := store.List(ctx, ports.RunHistoryFilter{Limit: 1})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(byLimit) != 1 {
		t.Errorf("limit returned %d runs, want 1", len(byLimit))
	}
}

func TestLookupBatchesByHost(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	if err := store.Record(ctx, sampleRecord("known-1")); err != nil {
		t.Fatalf("Record: %v", err)
	}
	if err := store.Record(ctx, sampleRecord("known-2")); err != nil {
		t.Fatalf("Record: %v", err)
	}

	found, err := store.Lookup(ctx, testHost, []string{"known-1", "unknown", "known-2"})
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if len(found) != 2 {
		t.Errorf("found %d records, want only the remembered ones", len(found))
	}
	if _, ok := found["unknown"]; ok {
		t.Error("Lookup invented a record for an unknown run")
	}

	empty, err := store.Lookup(ctx, testHost, nil)
	if err != nil || len(empty) != 0 {
		t.Errorf("Lookup(nil) = %v, %v; want an empty map and no error", empty, err)
	}
}

func TestForgetAndPrune(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	old := sampleRecord("old-run")
	old.SubmittedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	recent := sampleRecord("recent-run")
	recent.SubmittedAt = time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	for _, rec := range []ports.RunRecord{old, recent} {
		if err := store.Record(ctx, rec); err != nil {
			t.Fatalf("Record: %v", err)
		}
	}

	if err := store.Forget(ctx, ports.RunRef{Host: testHost, WorkflowID: "ghost"}); !errors.Is(err, ports.ErrRunNotRemembered) {
		t.Errorf("Forget of an unknown run = %v, want ErrRunNotRemembered", err)
	}

	removed, err := store.Prune(ctx, time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if removed != 1 {
		t.Errorf("pruned %d runs, want 1", removed)
	}

	if err := store.Forget(ctx, recent.Ref()); err != nil {
		t.Fatalf("Forget: %v", err)
	}
	all, _ := store.List(ctx, ports.RunHistoryFilter{})
	if len(all) != 0 {
		t.Errorf("%d runs left, want none", len(all))
	}
}

func TestMigrationIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")
	ctx := context.Background()

	first := New(path)
	if err := first.Record(ctx, sampleRecord("abc-123")); err != nil {
		t.Fatalf("Record: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	second := New(path)
	defer func() { _ = second.Close() }()
	got, err := second.Get(ctx, ports.RunRef{Host: testHost, WorkflowID: "abc-123"})
	if err != nil {
		t.Fatalf("Get after reopen: %v", err)
	}
	if got.Description != "smoke test" {
		t.Errorf("description = %q, want the record to survive reopening", got.Description)
	}
}

func TestCloseWithoutUseDoesNotCreateTheDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")
	store := New(path)

	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, err := filepath.Glob(path); err != nil {
		t.Fatalf("Glob: %v", err)
	}
	entries, err := filepath.Glob(filepath.Join(filepath.Dir(path), "*"))
	if err != nil {
		t.Fatalf("Glob: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("found %v, want no database created by an unused store", entries)
	}
}
