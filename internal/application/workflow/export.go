// export.go turns a remembered run back into the files it was submitted
// with, so it can be submitted again — by the same person, or by whoever the
// directory is handed to.
package workflow

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/lmtani/pumbaa/internal/application"
	"github.com/lmtani/pumbaa/internal/application/ports"
	workflow2 "github.com/lmtani/pumbaa/internal/domain/workflow"
	"github.com/lmtani/pumbaa/pkg/wdl"
)

// Fixed names for the exported files. The originals are recorded in the
// manifest; fixed names are what make the printed submit command always
// right, and keep an inputs and an options file that happened to share a
// basename from colliding.
const (
	ExportWorkflowName     = "workflow.wdl"
	ExportInputsName       = "inputs.json"
	ExportOptionsName      = "options.json"
	ExportDependenciesName = "dependencies.zip"
	ExportLabelsName       = "labels.json"
	ExportManifestName     = "manifest.json"
)

// cromwellIDLabel is the label Cromwell stamps on every run. It is not
// something the submitter gave, and submitting it again would be wrong.
const cromwellIDLabel = "cromwell-workflow-id"

// secretOptionKeys are the workflow options that carry credentials. An
// export meant for someone else should not travel with them.
var secretOptionKeys = []string{"user_service_account_json", "refresh_token", "auth_bearer_token"}

// ErrExportWouldOverwrite reports files already in the target directory.
var ErrExportWouldOverwrite = errors.New("export would overwrite existing files")

// ExportSource says where an exported file came from, which is what decides
// how much to trust it.
type ExportSource string

const (
	// ExportFromHistory is the file kept at submission time: the bytes the
	// server received.
	ExportFromHistory ExportSource = "history"
	// ExportFromServer is the file as the server reports it in
	// submittedFiles, exact for the workflow and inputs, reformatted for the
	// options.
	ExportFromServer ExportSource = "server"
	// ExportFromPath is the file read now from the path recorded at
	// submission time, which may have changed since.
	ExportFromPath ExportSource = "path"
)

// ExportRunUseCase materialises a run as a directory ready for `pumbaa
// workflow submit`. The kept files are the first choice; the server's submittedFiles
// cover runs nothing was kept for, except the dependencies zip, which the
// server discards and only a recorded path can still supply.
type ExportRunUseCase struct {
	store      ports.RunHistoryReader
	server     ports.SubmissionReader
	files      ports.FileProvider
	host       ports.HostProvider
	timeSource func() time.Time
}

// NewExportRunUseCase creates the use case. server may be nil, in which case
// only kept files can be exported; files may be nil, in which case recorded
// paths are never read.
func NewExportRunUseCase(store ports.RunHistoryReader, server ports.SubmissionReader, files ports.FileProvider, host ports.HostProvider) *ExportRunUseCase {
	return &ExportRunUseCase{store: store, server: server, files: files, host: host, timeSource: time.Now}
}

// ExportRunInput says which run and where to.
type ExportRunInput struct {
	WorkflowID string
	// Dir is where the files are written; it is created when missing.
	Dir string
	// Overwrite allows replacing files already in Dir.
	Overwrite bool
	// StripSecrets removes credential-carrying keys from the options, for a
	// directory meant to be handed to someone else.
	StripSecrets bool
}

// ExportedFile is one file written by an export.
type ExportedFile struct {
	Kind ports.RunFileKind `json:"kind"`
	// Name is the file name inside the export directory.
	Name   string       `json:"file"`
	Source ExportSource `json:"source"`
	// OriginalName is the basename the file was submitted from, when known.
	OriginalName string `json:"original_name,omitempty"`
	SHA256       string `json:"sha256"`
	Size         int64  `json:"size"`
	// Warning is a reason to look at the file before submitting it.
	Warning string `json:"warning,omitempty"`
}

// MissingFile is a submitted file the export could not produce.
type MissingFile struct {
	Kind   ports.RunFileKind `json:"kind"`
	Reason string            `json:"reason"`
}

// ExportRunOutput describes what was written. It doubles as the manifest.
type ExportRunOutput struct {
	Format      int               `json:"pumbaa_export"`
	WorkflowID  string            `json:"workflow_id"`
	Host        string            `json:"host"`
	HostAlias   string            `json:"host_alias,omitempty"`
	Name        string            `json:"workflow_name,omitempty"`
	Description string            `json:"description,omitempty"`
	SubmittedAt time.Time         `json:"submitted_at,omitempty"`
	ExportedAt  time.Time         `json:"exported_at"`
	Labels      map[string]string `json:"labels,omitempty"`
	Files       []ExportedFile    `json:"files"`
	// Missing lists what could not be produced and why — typically the
	// dependencies zip of a run whose files were not kept.
	Missing []MissingFile `json:"missing,omitempty"`
	// Secrets names option keys that carry credentials; SecretsStripped
	// says whether they were removed from the exported options.
	Secrets         []string `json:"options_secrets,omitempty"`
	SecretsStripped bool     `json:"secrets_stripped,omitempty"`
	// SubmitCommand reproduces the run from the export directory.
	SubmitCommand string `json:"submit"`
	Dir           string `json:"-"`
}

// Complete reports whether everything needed to submit the run again was
// produced.
func (o *ExportRunOutput) Complete() bool {
	return len(o.Missing) == 0
}

// exportFile is a file on its way to disk.
type exportFile struct {
	ExportedFile
	data []byte
}

// Execute assembles the run's files and writes them, manifest included.
func (uc *ExportRunUseCase) Execute(ctx context.Context, input ExportRunInput) (*ExportRunOutput, error) {
	if input.WorkflowID == "" {
		return nil, application.NewInputValidationError("workflowID", "is required")
	}
	if input.Dir == "" {
		return nil, application.NewInputValidationError("dir", "is required")
	}

	out, files, err := uc.assemble(ctx, input)
	if err != nil {
		return nil, err
	}

	dir, err := filepath.Abs(input.Dir)
	if err != nil {
		return nil, application.NewUseCaseError("export", "invalid directory", err)
	}
	out.Dir = dir

	names := make([]string, 0, len(files)+1)
	for _, f := range files {
		names = append(names, f.Name)
	}
	names = append(names, ExportManifestName)
	if !input.Overwrite {
		if existing := existingFiles(dir, names); len(existing) > 0 {
			return nil, fmt.Errorf("%w in %s: %s", ErrExportWouldOverwrite, dir, strings.Join(existing, ", "))
		}
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, application.NewUseCaseError("export", "failed to create directory", err)
	}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(dir, f.Name), f.data, 0o644); err != nil {
			return nil, application.NewUseCaseError("export", "failed to write "+f.Name, err)
		}
	}
	manifest, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return nil, application.NewUseCaseError("export", "failed to encode the manifest", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ExportManifestName), append(manifest, '\n'), 0o644); err != nil {
		return nil, application.NewUseCaseError("export", "failed to write "+ExportManifestName, err)
	}
	return out, nil
}

// assemble gathers the files without touching the target directory, so a
// refusal to overwrite costs nothing.
func (uc *ExportRunUseCase) assemble(ctx context.Context, input ExportRunInput) (*ExportRunOutput, []exportFile, error) {
	host, alias := uc.host.CurrentHost()
	ref := ports.RunRef{Host: host, WorkflowID: input.WorkflowID}

	out := &ExportRunOutput{
		Format:     1,
		WorkflowID: input.WorkflowID,
		Host:       host,
		HostAlias:  alias,
		ExportedAt: uc.timeSource().UTC(),
	}

	rec, err := uc.store.Get(ctx, ref)
	remembered := err == nil
	if err != nil && !errors.Is(err, ports.ErrRunNotRemembered) {
		return nil, nil, application.NewUseCaseError("export", "failed to read the local history", err)
	}
	if remembered {
		out.Name = rec.Name
		out.Description = rec.Description
		out.SubmittedAt = rec.SubmittedAt
		out.Labels = rec.Labels
	}

	var files []exportFile
	if remembered {
		kept, err := uc.store.Files(ctx, ref)
		if err != nil {
			return nil, nil, application.NewUseCaseError("export", "failed to read the kept files", err)
		}
		files = fromHistory(kept)
	}

	if len(files) == 0 {
		files, err = uc.fromServer(ctx, out, remembered, rec)
		if err != nil {
			return nil, nil, err
		}
	}

	files = uc.completeDependencies(ctx, out, files, remembered, rec)
	files = uc.handleSecrets(out, files, input.StripSecrets)

	if len(out.Labels) > 0 {
		labels, _ := json.MarshalIndent(out.Labels, "", "  ")
		files = append(files, exportFile{
			ExportedFile: ExportedFile{Name: ExportLabelsName, Source: ExportFromHistory},
			data:         append(labels, '\n'),
		})
		if !remembered {
			files[len(files)-1].Source = ExportFromServer
		}
	}

	for i := range files {
		files[i].Size = int64(len(files[i].data))
		files[i].SHA256 = digest(files[i].data)
		out.Files = append(out.Files, files[i].ExportedFile)
	}
	out.SubmitCommand = submitCommand(out)
	return out, files, nil
}

// fromHistory maps the kept files to their export names.
func fromHistory(kept []ports.RunFile) []exportFile {
	var files []exportFile
	for _, f := range kept {
		name, ok := exportName(f.Kind)
		if !ok {
			continue
		}
		files = append(files, exportFile{
			ExportedFile: ExportedFile{Kind: f.Kind, Name: name, Source: ExportFromHistory, OriginalName: f.Name},
			data:         f.Data,
		})
	}
	return files
}

// fromServer reads the submission section of the run's metadata: exact for
// the workflow and inputs, reformatted for the options, and never holding
// the dependencies zip.
func (uc *ExportRunUseCase) fromServer(ctx context.Context, out *ExportRunOutput, remembered bool, rec ports.RunRecord) ([]exportFile, error) {
	if uc.server == nil {
		if remembered {
			return nil, application.NewUseCaseError("export", "no files were kept for this run and no server is configured to ask", ports.ErrRunNotRemembered)
		}
		return nil, application.NewUseCaseError("export", "no local record for this run and no server is configured to ask", ports.ErrRunNotRemembered)
	}

	sub, err := uc.server.GetSubmission(ctx, out.WorkflowID)
	if err != nil {
		return nil, application.NewUseCaseError("export", "no files kept locally, and the server could not supply them", err)
	}
	if sub.Workflow == "" {
		if sub.WorkflowURL != "" {
			return nil, application.NewUseCaseError("export",
				fmt.Sprintf("the run was submitted by URL (%s); the server kept no source to export", sub.WorkflowURL), workflow2.ErrWorkflowNotFound)
		}
		return nil, application.NewUseCaseError("export", "the server kept no workflow source for this run", workflow2.ErrWorkflowNotFound)
	}

	if !remembered {
		out.Name = sub.Name
		out.SubmittedAt = sub.SubmittedAt
		out.Labels = withoutCromwellLabel(sub.Labels)
	} else if out.Name == "" {
		out.Name = sub.Name
	}

	files := []exportFile{{
		ExportedFile: ExportedFile{Kind: ports.RunFileWorkflow, Name: ExportWorkflowName, Source: ExportFromServer, OriginalName: filepath.Base(rec.WorkflowFile)},
		data:         []byte(sub.Workflow),
	}}
	if sub.Inputs != "" && sub.Inputs != "{}" {
		files = append(files, exportFile{
			ExportedFile: ExportedFile{Kind: ports.RunFileInputs, Name: ExportInputsName, Source: ExportFromServer, OriginalName: filepath.Base(rec.InputsFile)},
			data:         []byte(sub.Inputs),
		})
	}
	if sub.Options != "" && strings.TrimSpace(sub.Options) != "{}" {
		files = append(files, exportFile{
			ExportedFile: ExportedFile{Kind: ports.RunFileOptions, Name: ExportOptionsName, Source: ExportFromServer, OriginalName: filepath.Base(rec.OptionsFile),
				Warning: "reformatted by the server; option values it encrypts, if any, are blank"},
			data: []byte(sub.Options),
		})
	}
	for i := range files {
		if files[i].OriginalName == "." {
			files[i].OriginalName = ""
		}
	}
	return files, nil
}

// completeDependencies makes sure a workflow with local imports leaves with
// a zip, or with a reason it could not. A zip from the history is trusted; a
// zip read from the recorded path is checked against the workflow's imports
// and marked as the best effort it is.
func (uc *ExportRunUseCase) completeDependencies(ctx context.Context, out *ExportRunOutput, files []exportFile, remembered bool, rec ports.RunRecord) []exportFile {
	var source []byte
	for _, f := range files {
		switch f.Kind {
		case ports.RunFileDependencies:
			return files
		case ports.RunFileWorkflow:
			source = f.data
		}
	}

	imports := localImports(source)
	if len(imports) == 0 {
		return files
	}

	if remembered && rec.DependenciesFile != "" && uc.files != nil {
		data, err := uc.files.ReadBytes(ctx, rec.DependenciesFile)
		if err == nil && len(data) > 0 {
			warning := "read now from " + rec.DependenciesFile + ", which may have changed since the submission"
			if report := wdl.CheckDependencies(source, data); report.HasErrors() {
				warning += "; it does not resolve every import of the workflow"
			}
			return append(files, exportFile{
				ExportedFile: ExportedFile{Kind: ports.RunFileDependencies, Name: ExportDependenciesName, Source: ExportFromPath,
					OriginalName: filepath.Base(rec.DependenciesFile), Warning: warning},
				data: data,
			})
		}
	}

	reason := fmt.Sprintf("the workflow imports %s, and Cromwell does not keep the dependencies zip; ", strings.Join(imports, ", "))
	switch {
	case !remembered:
		reason += "only the machine that submitted the run through pumbaa has it"
	case rec.DependenciesFile != "":
		reason += "the one recorded at " + rec.DependenciesFile + " could not be read"
	default:
		reason += "the run was recorded without one"
	}
	out.Missing = append(out.Missing, MissingFile{Kind: ports.RunFileDependencies, Reason: reason})
	return files
}

// localImports lists the imports a dependencies zip would have to satisfy.
// Remote imports are Cromwell's to fetch.
func localImports(source []byte) []string {
	return wdl.LocalImports(source)
}

// handleSecrets reports the credential-carrying option keys, and removes
// them when asked. Options that are not a JSON object are left as they are.
func (uc *ExportRunUseCase) handleSecrets(out *ExportRunOutput, files []exportFile, strip bool) []exportFile {
	for i := range files {
		if files[i].Kind != ports.RunFileOptions {
			continue
		}
		var options map[string]json.RawMessage
		if err := json.Unmarshal(files[i].data, &options); err != nil {
			return files
		}
		for _, key := range secretOptionKeys {
			if _, ok := options[key]; ok {
				out.Secrets = append(out.Secrets, key)
				if strip {
					delete(options, key)
				}
			}
		}
		if strip && len(out.Secrets) > 0 {
			out.SecretsStripped = true
			data, err := json.MarshalIndent(options, "", "  ")
			if err == nil {
				files[i].data = append(data, '\n')
				files[i].Warning = "credential options removed: " + strings.Join(out.Secrets, ", ")
			}
		}
	}
	return files
}

// submitCommand is the invocation that reproduces the run from the export
// directory. Labels are passed as flags: labels.json is for the reader.
func submitCommand(out *ExportRunOutput) string {
	parts := []string{"pumbaa workflow submit"}
	flags := map[ports.RunFileKind]string{
		ports.RunFileWorkflow:     "-w",
		ports.RunFileInputs:       "-i",
		ports.RunFileOptions:      "-o",
		ports.RunFileDependencies: "-d",
	}
	for _, kind := range ports.RunFileKinds {
		for _, f := range out.Files {
			if f.Kind == kind {
				parts = append(parts, flags[kind]+" "+f.Name)
			}
		}
	}
	keys := make([]string, 0, len(out.Labels))
	for key := range out.Labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("-l %s=%s", key, out.Labels[key]))
	}
	if out.Description != "" {
		parts = append(parts, fmt.Sprintf("-D %q", out.Description))
	}
	return strings.Join(parts, " ")
}

func exportName(kind ports.RunFileKind) (string, bool) {
	switch kind {
	case ports.RunFileWorkflow:
		return ExportWorkflowName, true
	case ports.RunFileInputs:
		return ExportInputsName, true
	case ports.RunFileOptions:
		return ExportOptionsName, true
	case ports.RunFileDependencies:
		return ExportDependenciesName, true
	}
	return "", false
}

func withoutCromwellLabel(labels map[string]string) map[string]string {
	if len(labels) == 0 {
		return nil
	}
	out := make(map[string]string, len(labels))
	for k, v := range labels {
		if k != cromwellIDLabel {
			out[k] = v
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func existingFiles(dir string, names []string) []string {
	var existing []string
	for _, name := range names {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			existing = append(existing, name)
		}
	}
	return existing
}
