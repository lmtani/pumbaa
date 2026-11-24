package history

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

func TestSearchMatchesEveryTermAcrossTheRecord(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	tso := sampleRecord("tso-run")
	tso.Name = "Tso500SomaticAnalysis"
	tso.Description = "rerun do painel com a referência reconstruída"
	tso.WorkflowFile = "/wdl/tso500.wdl"
	tso.InputsFile = "/inputs/hg38_v2.json"
	tso.Labels = map[string]string{"sample": "S001"}
	germline := sampleRecord("germline-run")
	germline.Name = "GermlineVariantCalling"
	germline.Description = "validação da coorte de janeiro"
	germline.WorkflowFile = "/wdl/germline.wdl"
	germline.InputsFile = "/inputs/cohort.json"
	germline.Labels = map[string]string{"project": "genomics"}
	for _, rec := range []ports.RunRecord{tso, germline} {
		if err := store.Record(ctx, rec); err != nil {
			t.Fatalf("Record: %v", err)
		}
	}

	tests := []struct {
		search string
		want   string
	}{
		{"tso500", "tso-run"},
		{"painel referência", "tso-run"},     // two terms, not adjacent in the text
		{"REFERÊNCIA PAINEL", "tso-run"},     // order and case do not matter
		{"hg38", "tso-run"},                  // an inputs path
		{"hg38_v2", "tso-run"},               // _ is text, not a wildcard
		{"s001", "tso-run"},                  // a label value
		{"genomics", "germline-run"},         // a label key's value
		{"germline janeiro", "germline-run"}, // name plus description
		{"/wdl/tso500.wdl", "tso-run"},       // a full path
	}
	for _, tt := range tests {
		t.Run(tt.search, func(t *testing.T) {
			got, err := store.List(ctx, ports.RunHistoryFilter{Search: tt.search})
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			if len(got) != 1 || got[0].WorkflowID != tt.want {
				t.Errorf("search %q matched %d runs %v, want just %s", tt.search, len(got), ids(got), tt.want)
			}
		})
	}
}

func TestListFiltersByLastKnownStatus(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	failed := sampleRecord("failed-run")
	failed.LastStatus = "Failed"
	succeeded := sampleRecord("succeeded-run")
	succeeded.LastStatus = "Succeeded"
	for _, rec := range []ports.RunRecord{failed, succeeded} {
		if err := store.Record(ctx, rec); err != nil {
			t.Fatalf("Record: %v", err)
		}
	}

	// Case-insensitive: an agent may well send "failed".
	for _, status := range []string{"Failed", "failed"} {
		got, err := store.List(ctx, ports.RunHistoryFilter{Status: status})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(got) != 1 || got[0].WorkflowID != "failed-run" {
			t.Errorf("status %q matched %v, want just failed-run", status, ids(got))
		}
	}
}

func TestSearchRequiresEveryTerm(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	rec := sampleRecord("tso-run")
	rec.Description = "rerun do painel"
	if err := store.Record(ctx, rec); err != nil {
		t.Fatalf("Record: %v", err)
	}

	// "painel" matches, "germline" does not: an AND, not an OR.
	got, err := store.List(ctx, ports.RunHistoryFilter{Search: "painel germline"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("matched %v, want nothing when one of the terms is absent", ids(got))
	}
}

func TestSearchTreatsWildcardsAsText(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	if err := store.Record(ctx, sampleRecord("abc-123")); err != nil {
		t.Fatalf("Record: %v", err)
	}

	// A bare wildcard must not match everything.
	for _, search := range []string{"%", "_"} {
		got, err := store.List(ctx, ports.RunHistoryFilter{Search: search})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("search %q matched %v, want it treated as literal text", search, ids(got))
		}
	}
}

func ids(records []ports.RunRecord) []string {
	out := make([]string, 0, len(records))
	for _, rec := range records {
		out = append(out, rec.WorkflowID)
	}
	return out
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
	if removed.Runs != 1 {
		t.Errorf("pruned %d runs, want 1", removed.Runs)
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

func TestTheDatabaseIsReadableByItsOwnerOnly(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permissions")
	}
	ctx := context.Background()
	dir := filepath.Join(t.TempDir(), "pumbaa")
	path := filepath.Join(dir, "history.db")

	store := New(path)
	if err := store.Record(ctx, sampleRecord("abc-123")); err != nil {
		t.Fatalf("Record: %v", err)
	}
	_ = store.Close()
	for p, want := range map[string]os.FileMode{dir: 0o700, path: 0o600} {
		info, err := os.Stat(p)
		if err != nil {
			t.Fatalf("stat %s: %v", p, err)
		}
		if got := info.Mode().Perm(); got != want {
			t.Errorf("%s mode = %o, want %o", filepath.Base(p), got, want)
		}
	}

	// A database created before files were kept is tightened on open.
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	reopened := New(path)
	defer func() { _ = reopened.Close() }()
	if _, err := reopened.Get(ctx, ports.RunRef{Host: testHost, WorkflowID: "abc-123"}); err != nil {
		t.Fatalf("Get: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("mode after reopening = %o, want 600", got)
	}
}

// Reads return the kept files with sizes but no bytes. Recording such a
// record back — read, edit, record — must not replace the only copy of the
// submitted files with empty ones.
func TestRecordRefusesFilesReadBackWithoutBytes(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	rec := sampleRecord("abc-123")
	rec.Files = []ports.RunFile{{Kind: ports.RunFileWorkflow, Name: "hello.wdl", Data: []byte("version 1.0\n")}}
	if err := store.Record(ctx, rec); err != nil {
		t.Fatalf("Record: %v", err)
	}

	got, err := store.Get(ctx, rec.Ref())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	got.Name = "Renamed"
	if err := store.Record(ctx, got); err == nil {
		t.Fatal("Record accepted kept files without their bytes")
	}

	kept, err := store.Files(ctx, rec.Ref())
	if err != nil {
		t.Fatalf("Files: %v", err)
	}
	if len(kept) != 1 || string(kept[0].Data) != "version 1.0\n" {
		t.Errorf("kept files = %+v, want the original bytes untouched", kept)
	}
	if after, _ := store.Get(ctx, rec.Ref()); after.Name == "Renamed" {
		t.Error("the refused record was partly written")
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

// keptFiles are files out of submission order on purpose: reads must put
// them back in it.
func keptFiles() []ports.RunFile {
	return []ports.RunFile{
		{Kind: ports.RunFileDependencies, Name: "deps.zip", Data: []byte("PK\x03\x04 not really a zip")},
		{Kind: ports.RunFileWorkflow, Name: "hello.wdl", Data: []byte("version 1.0\nworkflow HelloWorld {}\n")},
		{Kind: ports.RunFileInputs, Name: "hello.inputs.json", Data: []byte(`{"HelloWorld.name": "x"}`)},
	}
}

func hexDigest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func TestRecordKeepsFilesVerbatim(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	rec := sampleRecord("abc-123")
	rec.Files = keptFiles()

	if err := store.Record(ctx, rec); err != nil {
		t.Fatalf("Record: %v", err)
	}

	got, err := store.Get(ctx, rec.Ref())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	wantOrder := []ports.RunFileKind{ports.RunFileWorkflow, ports.RunFileInputs, ports.RunFileDependencies}
	if len(got.Files) != len(wantOrder) {
		t.Fatalf("Get returned %d files, want %d", len(got.Files), len(wantOrder))
	}
	for i, f := range got.Files {
		if f.Kind != wantOrder[i] {
			t.Errorf("files[%d] = %s, want %s: reads must follow submission order", i, f.Kind, wantOrder[i])
		}
		if f.Data != nil {
			t.Errorf("Get returned the bytes of %s; a listing wants sizes, not contents", f.Kind)
		}
	}

	files, err := store.Files(ctx, rec.Ref())
	if err != nil {
		t.Fatalf("Files: %v", err)
	}
	if len(files) != 3 {
		t.Fatalf("Files returned %d files, want 3", len(files))
	}
	wdl := files[0]
	if wdl.Kind != ports.RunFileWorkflow || wdl.Name != "hello.wdl" {
		t.Errorf("files[0] = %s %q, want the workflow hello.wdl", wdl.Kind, wdl.Name)
	}
	if !bytes.Equal(wdl.Data, keptFiles()[1].Data) {
		t.Errorf("workflow bytes = %q, want them verbatim", wdl.Data)
	}
	if wdl.Size != int64(len(wdl.Data)) || wdl.SHA256 != hexDigest(wdl.Data) {
		t.Errorf("size/sha = %d/%s, want the store to compute them from the bytes", wdl.Size, wdl.SHA256)
	}
	if !bytes.Equal(files[2].Data, keptFiles()[0].Data) {
		t.Errorf("dependencies bytes = %q, want them verbatim", files[2].Data)
	}
}

func TestRecordWithoutFilesKeepsTheExistingOnes(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	rec := sampleRecord("abc-123")
	rec.Files = keptFiles()
	if err := store.Record(ctx, rec); err != nil {
		t.Fatalf("Record: %v", err)
	}

	// A re-record with nil files — what a note or status refresh sends —
	// must not erase what only this machine still has.
	again := sampleRecord("abc-123")
	again.Files = nil
	if err := store.Record(ctx, again); err != nil {
		t.Fatalf("Record again: %v", err)
	}
	files, err := store.Files(ctx, rec.Ref())
	if err != nil {
		t.Fatalf("Files: %v", err)
	}
	if len(files) != 3 {
		t.Fatalf("%d files after a nil re-record, want the 3 kept ones", len(files))
	}

	// An explicit empty slice is a decision: keep nothing.
	again.Files = []ports.RunFile{}
	if err := store.Record(ctx, again); err != nil {
		t.Fatalf("Record with no files: %v", err)
	}
	files, err = store.Files(ctx, rec.Ref())
	if err != nil {
		t.Fatalf("Files: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("%d files after an empty re-record, want none", len(files))
	}
}

func TestFilesOfARunWithoutAny(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	if _, err := store.Files(ctx, ports.RunRef{Host: testHost, WorkflowID: "ghost"}); !errors.Is(err, ports.ErrRunNotRemembered) {
		t.Errorf("Files of an unknown run = %v, want ErrRunNotRemembered", err)
	}

	if err := store.Record(ctx, sampleRecord("abc-123")); err != nil {
		t.Fatalf("Record: %v", err)
	}
	files, err := store.Files(ctx, ports.RunRef{Host: testHost, WorkflowID: "abc-123"})
	if err != nil {
		t.Fatalf("Files of a run without kept files: %v, want an empty answer", err)
	}
	if len(files) != 0 {
		t.Errorf("Files = %v, want none", files)
	}
}

func TestListAndLookupAttachKeptFilesPerRun(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	with := sampleRecord("with-files")
	with.Files = keptFiles()
	without := sampleRecord("without-files")
	for _, rec := range []ports.RunRecord{with, without} {
		if err := store.Record(ctx, rec); err != nil {
			t.Fatalf("Record: %v", err)
		}
	}

	listed, err := store.List(ctx, ports.RunHistoryFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	counts := map[string]int{}
	for _, rec := range listed {
		counts[rec.WorkflowID] = len(rec.Files)
	}
	if counts["with-files"] != 3 || counts["without-files"] != 0 {
		t.Errorf("kept files per run = %v, want 3 for with-files and 0 for without-files", counts)
	}

	found, err := store.Lookup(ctx, testHost, []string{"with-files", "without-files"})
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if len(found["with-files"].Files) != 3 || len(found["without-files"].Files) != 0 {
		t.Errorf("Lookup kept files = %d/%d, want 3/0", len(found["with-files"].Files), len(found["without-files"].Files))
	}
}

func TestForgetDropsTheKeptFiles(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	rec := sampleRecord("abc-123")
	rec.Files = keptFiles()
	if err := store.Record(ctx, rec); err != nil {
		t.Fatalf("Record: %v", err)
	}
	if err := store.Forget(ctx, rec.Ref()); err != nil {
		t.Fatalf("Forget: %v", err)
	}

	// The run coming back with nil files must find nothing left behind.
	if err := store.Record(ctx, sampleRecord("abc-123")); err != nil {
		t.Fatalf("Record: %v", err)
	}
	files, err := store.Files(ctx, rec.Ref())
	if err != nil {
		t.Fatalf("Files: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("%d files survived Forget, want none", len(files))
	}
}

func TestPruneCountsTheBytesFreed(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	old := sampleRecord("old-run")
	old.SubmittedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	old.Files = keptFiles()
	recent := sampleRecord("recent-run")
	recent.SubmittedAt = time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	recent.Files = keptFiles()
	for _, rec := range []ports.RunRecord{old, recent} {
		if err := store.Record(ctx, rec); err != nil {
			t.Fatalf("Record: %v", err)
		}
	}
	var wantBytes int64
	for _, f := range keptFiles() {
		wantBytes += int64(len(f.Data))
	}

	removed, err := store.Prune(ctx, time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if removed.Runs != 1 || removed.Bytes != wantBytes {
		t.Errorf("Prune = %+v, want 1 run and %d bytes", removed, wantBytes)
	}
	files, err := store.Files(ctx, recent.Ref())
	if err != nil || len(files) != 3 {
		t.Errorf("recent run has %d files after prune (err %v), want its 3 untouched", len(files), err)
	}
}

func TestMigratesAVersionOneDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")
	ctx := context.Background()

	// A database as the first release left it: runs only, version 1.
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	legacySchema(t, raw, 1)
	if _, err := raw.Exec(`INSERT INTO runs (host, workflow_id, submitted_at, created_at, updated_at, description)
		VALUES (?, ?, ?, ?, ?, ?)`, testHost, "old-run", "2026-01-01T00:00:00Z", "2026-01-01T00:00:00Z", "2026-01-01T00:00:00Z", "from before files were kept"); err != nil {
		t.Fatalf("v1 row: %v", err)
	}
	if err := raw.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	store := New(path)
	defer func() { _ = store.Close() }()
	got, err := store.Get(ctx, ports.RunRef{Host: testHost, WorkflowID: "old-run"})
	if err != nil {
		t.Fatalf("Get after migration: %v", err)
	}
	if got.Description != "from before files were kept" || len(got.Files) != 0 {
		t.Errorf("got %+v, want the v1 row intact with no files", got)
	}

	rec := sampleRecord("new-run")
	rec.Files = keptFiles()
	if err := store.Record(ctx, rec); err != nil {
		t.Fatalf("Record with files after migration: %v", err)
	}
	db, err := store.open()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	var version, gooseVersion int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		t.Fatalf("user_version: %v", err)
	}
	if err := db.QueryRow(`SELECT MAX(version_id) FROM goose_db_version WHERE is_applied`).Scan(&gooseVersion); err != nil {
		t.Fatalf("goose version: %v", err)
	}
	// user_version stays in step so a binary from before goose finds the
	// database up to date rather than replaying its steps.
	if gooseVersion < legacyVersion || version != legacyVersion {
		t.Errorf("goose version = %d, user_version = %d; want at least %d and %d", gooseVersion, version, legacyVersion, legacyVersion)
	}
}

// legacySchema builds a database as the runner before goose left it at
// version: the same SQL steps, recorded in user_version only. The goose
// annotations in the files are SQL comments, so they run as they are.
func legacySchema(t *testing.T, db *sql.DB, version int) {
	t.Helper()
	files := []string{"migrations/00001_runs.sql", "migrations/00002_run_files.sql"}
	for _, name := range files[:min(version, len(files))] {
		step, err := migrationFiles.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if _, err := db.Exec(string(step)); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	if version >= 3 {
		if _, err := db.Exec(`ALTER TABLE run_files ADD COLUMN withheld TEXT NOT NULL DEFAULT ''`); err != nil {
			t.Fatalf("v3 schema: %v", err)
		}
	}
	if _, err := db.Exec(fmt.Sprintf("PRAGMA user_version = %d", version)); err != nil {
		t.Fatalf("user_version: %v", err)
	}
}

// The release before goose migrated to user_version 3 with its own runner.
// Opening that database must adopt the three versions — running v3's ALTER
// TABLE again would fail and lock the user out of their history.
func TestAdoptsADatabaseTheOldRunnerMigrated(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")
	ctx := context.Background()
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	legacySchema(t, raw, 3)
	if _, err := raw.Exec(`INSERT INTO runs (host, workflow_id, submitted_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		testHost, "old-run", "2026-01-01T00:00:00Z", "2026-01-01T00:00:00Z", "2026-01-01T00:00:00Z"); err != nil {
		t.Fatalf("run row: %v", err)
	}
	if _, err := raw.Exec(`INSERT INTO run_files (host, workflow_id, kind, name, sha256, size, data, withheld) VALUES (?, ?, 'options', 'o.json', '', 2, '{}', 'refresh_token')`,
		testHost, "old-run"); err != nil {
		t.Fatalf("file row: %v", err)
	}
	_ = raw.Close()

	store := New(path)
	defer func() { _ = store.Close() }()
	files, err := store.Files(ctx, ports.RunRef{Host: testHost, WorkflowID: "old-run"})
	if err != nil {
		t.Fatalf("Files after adopting: %v", err)
	}
	if len(files) != 1 || len(files[0].Withheld) != 1 {
		t.Errorf("files = %+v, want the row the old runner wrote", files)
	}
}

func TestWithheldCredentialNamesRoundTrip(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	rec := sampleRecord("abc-123")
	rec.Files = []ports.RunFile{{Kind: ports.RunFileOptions, Name: "o.json", Data: []byte(`{"jes_gcs_root": "gs://b"}`), Withheld: []string{"user_service_account_json", "refresh_token"}}}
	if err := store.Record(ctx, rec); err != nil {
		t.Fatalf("Record: %v", err)
	}

	files, err := store.Files(ctx, rec.Ref())
	if err != nil || len(files) != 1 {
		t.Fatalf("Files = %+v (%v)", files, err)
	}
	if got := files[0].Withheld; len(got) != 2 || got[0] != "user_service_account_json" || got[1] != "refresh_token" {
		t.Errorf("Files withheld = %v, want both names back", got)
	}
	got, err := store.Get(ctx, rec.Ref())
	if err != nil || len(got.Files) != 1 || len(got.Files[0].Withheld) != 2 {
		t.Errorf("Get files = %+v (%v), want the withheld names in a read without bytes too", got.Files, err)
	}
}

// A database kept by a build that stored the options verbatim holds the
// credentials themselves. The migration must take them out, record their
// names, and leave no copy of the bytes anywhere in the file.
func TestMigrationScrubsCredentialsKeptBefore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")
	ctx := context.Background()
	const marker = "SECRET-MARKER-7f3a"

	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	legacySchema(t, raw, 2)
	// Large enough to spill into an overflow page, which is what a freed
	// page left unzeroed would keep.
	secret := `{"jes_gcs_root": "gs://b/x", "user_service_account_json": "` + marker + strings.Repeat("k", 6000) + `"}`
	for _, row := range []struct{ id, data string }{{"with-secret", secret}, {"unreadable", "not json " + marker}, {"clean", `{"read_from_cache": false}`}} {
		if _, err := raw.Exec(`INSERT INTO runs (host, workflow_id, submitted_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
			testHost, row.id, "2026-01-01T00:00:00Z", "2026-01-01T00:00:00Z", "2026-01-01T00:00:00Z"); err != nil {
			t.Fatalf("run row: %v", err)
		}
		if _, err := raw.Exec(`INSERT INTO run_files (host, workflow_id, kind, name, sha256, size, data) VALUES (?, ?, 'options', 'o.json', ?, ?, ?)`,
			testHost, row.id, hexDigest([]byte(row.data)), len(row.data), []byte(row.data)); err != nil {
			t.Fatalf("file row: %v", err)
		}
	}
	if err := raw.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	store := New(path)
	files, err := store.Files(ctx, ports.RunRef{Host: testHost, WorkflowID: "with-secret"})
	if err != nil || len(files) != 1 {
		t.Fatalf("Files = %+v (%v)", files, err)
	}
	f := files[0]
	if strings.Contains(string(f.Data), marker) || !strings.Contains(string(f.Data), "gs://b/x") {
		t.Errorf("options = %s, want the credential gone and the rest kept", f.Data)
	}
	if len(f.Withheld) != 1 || f.Withheld[0] != "user_service_account_json" {
		t.Errorf("Withheld = %v, want the credential's name", f.Withheld)
	}
	if f.SHA256 != hexDigest(f.Data) || f.Size != int64(len(f.Data)) {
		t.Errorf("digest/size = %s/%d, want them recomputed for the scrubbed bytes", f.SHA256, f.Size)
	}
	if files, _ := store.Files(ctx, ports.RunRef{Host: testHost, WorkflowID: "unreadable"}); len(files) != 0 {
		t.Errorf("options that cannot be checked were kept: %+v", files)
	}
	if files, _ := store.Files(ctx, ports.RunRef{Host: testHost, WorkflowID: "clean"}); len(files) != 1 || string(files[0].Data) != `{"read_from_cache": false}` {
		t.Errorf("options without credentials = %+v, want them untouched", files)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	matches, _ := filepath.Glob(path + "*")
	for _, p := range matches {
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("read %s: %v", p, err)
		}
		if bytes.Contains(data, []byte(marker)) {
			t.Errorf("%s still holds the credential's bytes", filepath.Base(p))
		}
	}
}
