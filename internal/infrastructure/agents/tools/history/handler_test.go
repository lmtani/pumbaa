package history

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/lmtani/pumbaa/internal/application/ports"
	"github.com/lmtani/pumbaa/internal/infrastructure/agents/tools/types"
)

type stubStore struct {
	records    []ports.RunRecord
	lastFilter ports.RunHistoryFilter
}

func (s *stubStore) Get(_ context.Context, ref ports.RunRef) (ports.RunRecord, error) {
	for _, rec := range s.records {
		if rec.Host == ref.Host && rec.WorkflowID == ref.WorkflowID {
			return rec, nil
		}
	}
	return ports.RunRecord{}, ports.ErrRunNotRemembered
}

func (s *stubStore) List(_ context.Context, filter ports.RunHistoryFilter) ([]ports.RunRecord, error) {
	s.lastFilter = filter
	return s.records, nil
}

func (s *stubStore) Lookup(context.Context, string, []string) (map[string]ports.RunRecord, error) {
	return nil, nil
}

type stubHost struct{}

func (stubHost) CurrentHost() (string, string) { return "http://localhost:8000", "local" }

func record(id string) ports.RunRecord {
	return ports.RunRecord{
		Host:         "http://localhost:8000",
		WorkflowID:   id,
		Name:         "Tso500",
		Description:  "rerun with the rebuilt reference panel",
		WorkflowFile: "/wdl/tso500.wdl",
		Origin:       ports.OriginSubmit,
		SubmittedAt:  time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC),
		LastStatus:   "Succeeded",
	}
}

func data(t *testing.T, out types.Output) map[string]any {
	t.Helper()
	if !out.Success {
		t.Fatalf("action failed: %s", out.Error)
	}
	m, ok := out.Data.(map[string]any)
	if !ok {
		t.Fatalf("data is %T, want a map", out.Data)
	}
	return m
}

func TestHandleReturnsOneRun(t *testing.T) {
	h := NewHandler(&stubStore{records: []ports.RunRecord{record("run-1")}}, stubHost{})

	out, err := h.Handle(context.Background(), types.Input{Action: action, WorkflowID: "run-1"})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}

	run, ok := data(t, out)["run"].(map[string]any)
	if !ok {
		t.Fatal("no run in the answer")
	}
	if run["description"] != "rerun with the rebuilt reference panel" {
		t.Errorf("description = %v, want the local note", run["description"])
	}
	if run["workflow_file"] != "/wdl/tso500.wdl" {
		t.Errorf("workflow_file = %v, want the recorded path", run["workflow_file"])
	}
	if data(t, out)["host"] != "local" {
		t.Errorf("host = %v, want the alias", data(t, out)["host"])
	}
}

func TestHandleExplainsAnUnrememberedRun(t *testing.T) {
	h := NewHandler(&stubStore{}, stubHost{})

	out, err := h.Handle(context.Background(), types.Input{Action: action, WorkflowID: "ghost"})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if out.Success {
		t.Fatal("Success = true, want the absence reported")
	}
	if !strings.Contains(out.Error, "pumbaa history note") {
		t.Errorf("error %q does not say how a run comes to be remembered", out.Error)
	}
}

func TestHandleListsScopedToTheActiveHost(t *testing.T) {
	store := &stubStore{records: []ports.RunRecord{record("run-1"), record("run-2")}}
	h := NewHandler(store, stubHost{})

	out, err := h.Handle(context.Background(), types.Input{Action: action, Query: "reference"})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}

	if store.lastFilter.Host != "http://localhost:8000" {
		t.Errorf("host filter = %q, want the active host", store.lastFilter.Host)
	}
	if store.lastFilter.Search != "reference" {
		t.Errorf("search = %q, want the query passed through", store.lastFilter.Search)
	}
	if store.lastFilter.Limit != defaultLimit {
		t.Errorf("limit = %d, want the default cap on an unbounded question", store.lastFilter.Limit)
	}

	runs, ok := data(t, out)["runs"].([]map[string]any)
	if !ok || len(runs) != 2 {
		t.Fatalf("runs = %v, want both records", data(t, out)["runs"])
	}
	if data(t, out)["note"] == nil {
		t.Error("no note warning that the statuses may lag")
	}
}

func TestViewDropsEmptyFields(t *testing.T) {
	rec := ports.RunRecord{WorkflowID: "run-1", Origin: ports.OriginNote}

	got := view(rec)
	for _, key := range []string{"name", "description", "inputs_file", "options_file", "labels"} {
		if _, present := got[key]; present {
			t.Errorf("%q is in the answer despite being empty", key)
		}
	}
	if got["remembered"] != string(ports.OriginNote) {
		t.Errorf("remembered = %v, want how the run came to be known", got["remembered"])
	}
}
