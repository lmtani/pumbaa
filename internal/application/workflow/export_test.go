package workflow

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lmtani/pumbaa/internal/application/ports"
	"github.com/lmtani/pumbaa/internal/domain/workflow"
)

const exportHost = "http://localhost:8000"

type stubSubmissionReader struct {
	sub   *workflow.Submission
	err   error
	calls int
}

func (s *stubSubmissionReader) GetSubmission(_ context.Context, _ string) (*workflow.Submission, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	return s.sub, nil
}

const importingWDL = "version 1.0\nimport \"lib/tasks.wdl\" as lib\nworkflow HelloWorld {\n  call lib.hello\n}\n"

func zipWith(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("zip create: %v", err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatalf("zip write: %v", err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}
	return buf.Bytes()
}

func exportRecord(files ...ports.RunFile) (*stubRunHistory, ports.RunRecord) {
	rec := historyRecord(exportHost, "run-1", "Succeeded")
	rec.Description = "smoke test"
	rec.Labels = map[string]string{"owner": "lucas"}
	store := newStubRunHistory(rec)
	if files != nil {
		store.files = map[string][]ports.RunFile{"run-1": files}
	}
	return store, rec
}

func keptSet(t *testing.T) []ports.RunFile {
	t.Helper()
	return []ports.RunFile{
		{Kind: ports.RunFileWorkflow, Name: "hello.wdl", Data: []byte(importingWDL)},
		{Kind: ports.RunFileInputs, Name: "hello.inputs.json", Data: []byte(`{"HelloWorld.name": "x"}`)},
		{Kind: ports.RunFileOptions, Name: "prod.options.json", Data: []byte(`{"jes_gcs_root": "gs://b/x", "user_service_account_json": "{\"k\":1}"}`)},
		{Kind: ports.RunFileDependencies, Name: "deps.zip", Data: zipWith(t, map[string]string{"lib/tasks.wdl": "version 1.0\ntask hello { command {} }\n"})},
	}
}

func newExport(store ports.RunHistoryReader, server ports.SubmissionReader, files ports.FileProvider) *ExportRunUseCase {
	uc := NewExportRunUseCase(store, server, files, fakeHostProvider{url: exportHost, alias: "local"})
	uc.timeSource = func() time.Time { return time.Date(2026, 11, 5, 10, 0, 0, 0, time.UTC) }
	return uc
}

func readExported(t *testing.T, dir, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return data
}

func TestExportWritesTheKeptFilesVerbatim(t *testing.T) {
	store, _ := exportRecord(keptSet(t)...)
	server := &stubSubmissionReader{}
	dir := filepath.Join(t.TempDir(), "out")

	out, err := newExport(store, server, nil).Execute(context.Background(), ExportRunInput{WorkflowID: "run-1", Dir: dir})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if server.calls != 0 {
		t.Error("the server was asked although every file was kept locally")
	}
	if !out.Complete() {
		t.Errorf("Missing = %v, want a complete export", out.Missing)
	}

	kept := keptSet(t)
	for i, name := range []string{ExportWorkflowName, ExportInputsName, ExportOptionsName, ExportDependenciesName} {
		if got := readExported(t, dir, name); !bytes.Equal(got, kept[i].Data) {
			t.Errorf("%s differs from the kept bytes", name)
		}
	}
	if len(out.Files) != 5 {
		t.Fatalf("manifest lists %d files, want 4 submitted ones plus labels", len(out.Files))
	}
	if out.Files[0].Source != ExportFromHistory || out.Files[0].OriginalName != "hello.wdl" {
		t.Errorf("workflow entry = %+v, want source history under its original name", out.Files[0])
	}

	var labels map[string]string
	if err := json.Unmarshal(readExported(t, dir, ExportLabelsName), &labels); err != nil || labels["owner"] != "lucas" {
		t.Errorf("labels.json = %v (%v), want the submitted labels", labels, err)
	}

	var manifest ExportRunOutput
	if err := json.Unmarshal(readExported(t, dir, ExportManifestName), &manifest); err != nil {
		t.Fatalf("manifest.json: %v", err)
	}
	if manifest.WorkflowID != "run-1" || manifest.Host != exportHost || manifest.Description != "smoke test" {
		t.Errorf("manifest = %+v, want the run's identity and note", manifest)
	}

	want := `pumbaa workflow submit -w workflow.wdl -i inputs.json -o options.json -d dependencies.zip -l owner=lucas -D "smoke test"`
	if out.SubmitCommand != want {
		t.Errorf("submit command = %q, want %q", out.SubmitCommand, want)
	}

	if len(out.Secrets) != 1 || out.Secrets[0] != "user_service_account_json" || out.SecretsStripped {
		t.Errorf("Secrets = %v stripped=%v, want the credential key reported and left in place", out.Secrets, out.SecretsStripped)
	}
}

func TestExportStripsSecretsOnRequest(t *testing.T) {
	store, _ := exportRecord(keptSet(t)...)
	dir := filepath.Join(t.TempDir(), "out")

	out, err := newExport(store, nil, nil).Execute(context.Background(), ExportRunInput{WorkflowID: "run-1", Dir: dir, StripSecrets: true})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !out.SecretsStripped {
		t.Error("SecretsStripped = false, want the credential removed")
	}
	var options map[string]any
	if err := json.Unmarshal(readExported(t, dir, ExportOptionsName), &options); err != nil {
		t.Fatalf("options.json: %v", err)
	}
	if _, ok := options["user_service_account_json"]; ok {
		t.Error("options.json still carries user_service_account_json")
	}
	if options["jes_gcs_root"] != "gs://b/x" {
		t.Errorf("options.json = %v, want the other options untouched", options)
	}
}

func TestExportFallsBackToTheServer(t *testing.T) {
	server := &stubSubmissionReader{sub: &workflow.Submission{
		ID:          "run-9",
		Name:        "HelloWorld",
		Labels:      map[string]string{"cromwell-workflow-id": "cromwell-run-9", "sample": "S1"},
		SubmittedAt: time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC),
		Workflow:    "version 1.0\nworkflow HelloWorld {}\n",
		Inputs:      `{"HelloWorld.name":"x"}`,
		Options:     "{\n  \"read_from_cache\": false\n}",
	}}
	dir := filepath.Join(t.TempDir(), "out")

	out, err := newExport(newStubRunHistory(), server, nil).Execute(context.Background(), ExportRunInput{WorkflowID: "run-9", Dir: dir})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !out.Complete() {
		t.Errorf("Missing = %v; a workflow without imports needs no zip", out.Missing)
	}
	if out.Name != "HelloWorld" || !out.SubmittedAt.Equal(server.sub.SubmittedAt) {
		t.Errorf("name/submitted = %q/%v, want them from the server", out.Name, out.SubmittedAt)
	}
	if _, ok := out.Labels["cromwell-workflow-id"]; ok || out.Labels["sample"] != "S1" {
		t.Errorf("labels = %v, want Cromwell's own label dropped and the rest kept", out.Labels)
	}
	for _, f := range out.Files {
		if f.Source != ExportFromServer {
			t.Errorf("%s came from %s, want the server", f.Name, f.Source)
		}
	}
	if got := readExported(t, dir, ExportInputsName); string(got) != `{"HelloWorld.name":"x"}` {
		t.Errorf("inputs.json = %q, want the server's exact string", got)
	}
	if !strings.Contains(out.SubmitCommand, "-l sample=S1") || strings.Contains(out.SubmitCommand, "-d ") {
		t.Errorf("submit command = %q", out.SubmitCommand)
	}
}

func TestExportReportsAMissingDependenciesZip(t *testing.T) {
	server := &stubSubmissionReader{sub: &workflow.Submission{ID: "run-9", Name: "HelloWorld", Workflow: importingWDL}}
	dir := filepath.Join(t.TempDir(), "out")

	out, err := newExport(newStubRunHistory(), server, nil).Execute(context.Background(), ExportRunInput{WorkflowID: "run-9", Dir: dir})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.Complete() || len(out.Missing) != 1 || out.Missing[0].Kind != ports.RunFileDependencies {
		t.Fatalf("Missing = %v, want the dependencies zip reported", out.Missing)
	}
	if !strings.Contains(out.Missing[0].Reason, "lib/tasks.wdl") || !strings.Contains(out.Missing[0].Reason, "submitted the run") {
		t.Errorf("reason = %q, want the import named and who still has the zip", out.Missing[0].Reason)
	}
	if _, err := os.Stat(filepath.Join(dir, ExportDependenciesName)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("dependencies.zip exists (err %v), want nothing written for a missing file", err)
	}
	if strings.Contains(out.SubmitCommand, "-d ") {
		t.Errorf("submit command %q names a zip that was not written", out.SubmitCommand)
	}
}

func TestExportReadsTheRecordedDependenciesPath(t *testing.T) {
	store, rec := exportRecord()
	rec.DependenciesFile = "/wf/deps.zip"
	rec.WorkflowFile = "/wf/hello.wdl"
	store.records[0] = rec
	server := &stubSubmissionReader{sub: &workflow.Submission{ID: "run-1", Name: "HelloWorld", Workflow: importingWDL}}
	zipData := zipWith(t, map[string]string{"lib/tasks.wdl": "version 1.0\ntask hello { command {} }\n"})
	fp := &mockFileProvider{readBytesFunc: func(_ context.Context, path string) ([]byte, error) {
		if path == "/wf/deps.zip" {
			return zipData, nil
		}
		return nil, errors.New("unexpected path " + path)
	}}
	dir := filepath.Join(t.TempDir(), "out")

	out, err := newExport(store, server, fp).Execute(context.Background(), ExportRunInput{WorkflowID: "run-1", Dir: dir})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !out.Complete() {
		t.Fatalf("Missing = %v, want the zip read from the recorded path", out.Missing)
	}
	var deps *ExportedFile
	for i := range out.Files {
		if out.Files[i].Kind == ports.RunFileDependencies {
			deps = &out.Files[i]
		}
	}
	if deps == nil {
		t.Fatal("no dependencies entry in the manifest")
	}
	if deps.Source != ExportFromPath || !strings.Contains(deps.Warning, "/wf/deps.zip") || strings.Contains(deps.Warning, "does not resolve") {
		t.Errorf("dependencies entry = %+v, want source path with a warning about the path, and no complaint about imports", deps)
	}
	if got := readExported(t, dir, ExportDependenciesName); !bytes.Equal(got, zipData) {
		t.Error("dependencies.zip differs from the file at the recorded path")
	}
	if out.Files[0].Source != ExportFromServer || out.Files[0].OriginalName != "hello.wdl" {
		t.Errorf("workflow entry = %+v, want the server's source under the recorded basename", out.Files[0])
	}
}

func TestExportRefusesToOverwrite(t *testing.T) {
	store, _ := exportRecord(keptSet(t)...)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ExportWorkflowName), []byte("mine"), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}

	_, err := newExport(store, nil, nil).Execute(context.Background(), ExportRunInput{WorkflowID: "run-1", Dir: dir})
	if !errors.Is(err, ErrExportWouldOverwrite) || !strings.Contains(err.Error(), ExportWorkflowName) {
		t.Fatalf("err = %v, want ErrExportWouldOverwrite naming workflow.wdl", err)
	}
	if got := readExported(t, dir, ExportWorkflowName); string(got) != "mine" {
		t.Error("the existing file was touched")
	}
	if _, err := os.Stat(filepath.Join(dir, ExportInputsName)); !errors.Is(err, os.ErrNotExist) {
		t.Error("other files were written although the export was refused")
	}

	if _, err := newExport(store, nil, nil).Execute(context.Background(), ExportRunInput{WorkflowID: "run-1", Dir: dir, Overwrite: true}); err != nil {
		t.Fatalf("Execute with Overwrite: %v", err)
	}
	if got := readExported(t, dir, ExportWorkflowName); string(got) != importingWDL {
		t.Error("Overwrite did not replace the file")
	}
}

func TestExportExplainsWhenNothingCanSupplyTheFiles(t *testing.T) {
	_, err := newExport(newStubRunHistory(), nil, nil).Execute(context.Background(), ExportRunInput{WorkflowID: "ghost", Dir: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "no local record") {
		t.Errorf("err = %v, want the absence of a record and a server explained", err)
	}

	store, _ := exportRecord()
	_, err = newExport(store, nil, nil).Execute(context.Background(), ExportRunInput{WorkflowID: "run-1", Dir: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "no files were kept") {
		t.Errorf("err = %v, want the absence of kept files explained", err)
	}

	server := &stubSubmissionReader{sub: &workflow.Submission{ID: "run-1", WorkflowURL: "https://example.org/wf.wdl"}}
	_, err = newExport(store, server, nil).Execute(context.Background(), ExportRunInput{WorkflowID: "run-1", Dir: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "https://example.org/wf.wdl") {
		t.Errorf("err = %v, want the URL submission explained", err)
	}

	server = &stubSubmissionReader{err: errors.New("connection refused")}
	_, err = newExport(store, server, nil).Execute(context.Background(), ExportRunInput{WorkflowID: "run-1", Dir: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "connection refused") {
		t.Errorf("err = %v, want the server failure surfaced", err)
	}
}
