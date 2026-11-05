package handler

import (
	"encoding/json"
	"testing"

	"github.com/lmtani/pumbaa/internal/application/ports"
	"github.com/lmtani/pumbaa/internal/application/workflow"
)

// A script reading the listing must not have to infer "can I export this"
// from the shape of Files, and must still find the fields it found before.
func TestEntriesJSONSayWhichRunsAreExportable(t *testing.T) {
	entries := []workflow.RunHistoryEntry{
		{Record: ports.RunRecord{WorkflowID: "kept", Files: []ports.RunFile{{Kind: ports.RunFileWorkflow, Size: 10}}}, Status: "Succeeded"},
		{Record: ports.RunRecord{WorkflowID: "paths-only"}, Status: "Failed", Forgotten: true},
	}

	raw, err := json.Marshal(entriesJSON(entries))
	if err != nil {
		t.Fatal(err)
	}
	var got []map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d entries, want 2", len(got))
	}
	if got[0]["Exportable"] != true || got[1]["Exportable"] != false {
		t.Errorf("Exportable = %v, %v; want true, false", got[0]["Exportable"], got[1]["Exportable"])
	}
	for _, key := range []string{"Record", "Status", "Stale", "Forgotten"} {
		if _, ok := got[0][key]; !ok {
			t.Errorf("entry lost its %s field", key)
		}
	}
	if got[1]["Forgotten"] != true {
		t.Errorf("Forgotten = %v, want true", got[1]["Forgotten"])
	}
}

// The export's JSON is the manifest, so a script reads one shape whether it
// takes it from stdout or from the directory, plus where that directory is.
func TestExportJSONIsTheManifestPlusWhereItWent(t *testing.T) {
	out := &workflow.ExportRunOutput{
		WorkflowID:    "abc",
		Dir:           "/tmp/abc",
		Files:         []workflow.ExportedFile{{Kind: ports.RunFileWorkflow, Name: workflow.ExportWorkflowName}},
		Missing:       []workflow.MissingFile{{Kind: ports.RunFileDependencies, Reason: "not kept"}},
		SubmitCommand: "pumbaa workflow submit -w workflow.wdl",
	}

	raw, err := json.Marshal(exportJSON{Dir: out.Dir, Complete: out.Complete(), ExportRunOutput: out})
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got["dir"] != "/tmp/abc" {
		t.Errorf("dir = %v, want /tmp/abc", got["dir"])
	}
	if got["complete"] != false {
		t.Errorf("complete = %v, want false", got["complete"])
	}
	for _, key := range []string{"workflow_id", "files", "missing", "submit"} {
		if _, ok := got[key]; !ok {
			t.Errorf("manifest field %s is missing from the JSON", key)
		}
	}
}
