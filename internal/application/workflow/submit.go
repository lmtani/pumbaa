package workflow

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/lmtani/pumbaa/internal/application"
	"github.com/lmtani/pumbaa/internal/application/ports"
	workflow2 "github.com/lmtani/pumbaa/internal/domain/workflow"
)

// SubmitHistory is the optional local memory of submissions: the store to
// write to, and the server the submission is going to, which records are
// keyed by.
type SubmitHistory struct {
	Store ports.RunHistory
	Host  ports.HostProvider
	// SkipFiles records the paths only, as the history did before it kept
	// the files themselves. Off by default: the kept files are what make a
	// run exportable after the paths have rotted.
	SkipFiles bool
	// MaxFileBytes caps the total size of the files kept for one run; above
	// it the run is recorded without them. Zero means DefaultMaxKeptBytes.
	MaxFileBytes int64
}

// DefaultMaxKeptBytes is the size above which a submission's files are not
// kept. A WDL bundle is tens of kilobytes; something this large is an inputs
// JSON with data inlined, not worth a database that grows by it on every run.
const DefaultMaxKeptBytes = 50 << 20

// SubmitUseCase handles workflow submission.
type SubmitUseCase struct {
	submitter    ports.WorkflowSubmitter
	fileProvider ports.FileProvider
	preflight    *PreflightUseCase
	history      *SubmitHistory
}

// NewSubmitUseCase creates a new submit use case. preflight may be nil, in
// which case submissions are not checked before being sent; history may be
// nil, in which case they are not remembered locally.
func NewSubmitUseCase(submitter ports.WorkflowSubmitter, fileProvider ports.FileProvider, preflight *PreflightUseCase, history *SubmitHistory) *SubmitUseCase {
	return &SubmitUseCase{submitter: submitter, fileProvider: fileProvider, preflight: preflight, history: history}
}

// SubmitInput represents the input for workflow submission.
type SubmitInput struct {
	WorkflowFile     string
	InputsFile       string
	OptionsFile      string
	DependenciesFile string
	Labels           map[string]string
	// Description is the free-text note kept in the local run history. It is
	// not sent to Cromwell: labels there are constrained, and this is meant
	// to be a sentence, not a tag.
	Description string
	// SkipPreflight submits without checking the workflow and its inputs
	// first.
	SkipPreflight bool
}

// SubmitOutput represents the output of workflow submission.
type SubmitOutput struct {
	WorkflowID string
	Status     string
	// Preflight is the report of the checks run before submitting, so the
	// caller can confirm they happened (and surface any warnings). Nil when
	// preflight was skipped.
	Preflight *PreflightReport
	// Remembered reports whether the submission made it into the local run
	// history, and HistoryError why it did not. A failure here never fails
	// the submission: the workflow is already running.
	Remembered   bool
	HistoryError error
	// FilesKept reports that the submitted files were kept verbatim with the
	// record, which is what lets `pumbaa history export` reproduce the run.
	// FilesSkipped says why they were not when keeping was expected; it is
	// empty when keeping is simply off.
	FilesKept    bool
	FilesSkipped string
}

// Execute submits a workflow to Cromwell.
func (uc *SubmitUseCase) Execute(ctx context.Context, input SubmitInput) (*SubmitOutput, error) {
	if input.WorkflowFile == "" {
		return nil, application.NewInputValidationError("workflowFile", "is required")
	}

	// Read workflow source
	workflowSource, err := uc.fileProvider.ReadBytes(ctx, input.WorkflowFile)
	if err != nil {
		return nil, application.NewUseCaseError("submit", "failed to read workflow file", err)
	}

	// Read optional files
	var inputsData, optionsData, depsData []byte

	if input.InputsFile != "" {
		inputsData, err = uc.fileProvider.ReadBytes(ctx, input.InputsFile)
		if err != nil {
			return nil, application.NewUseCaseError("submit", "failed to read inputs file", err)
		}
	}

	if input.OptionsFile != "" {
		optionsData, err = uc.fileProvider.ReadBytes(ctx, input.OptionsFile)
		if err != nil {
			return nil, application.NewUseCaseError("submit", "failed to read options file", err)
		}
	}

	if input.DependenciesFile != "" {
		depsData, err = uc.fileProvider.ReadBytes(ctx, input.DependenciesFile)
		if err != nil {
			return nil, application.NewUseCaseError("submit", "failed to read dependencies file", err)
		}
	}

	// Catch what Cromwell would only tell us minutes (and dollars) later.
	// The server check is skipped: submitting is about to contact it anyway.
	var report *PreflightReport
	if uc.preflight != nil && !input.SkipPreflight {
		report = uc.preflight.check(ctx, workflowSource, inputsData, depsData, true, false)
		if report.HasErrors() {
			return nil, &PreflightFailedError{Report: report}
		}
	}

	req := workflow2.SubmitRequest{
		WorkflowSource:       workflowSource,
		WorkflowInputs:       inputsData,
		WorkflowOptions:      optionsData,
		WorkflowDependencies: depsData,
		Labels:               input.Labels,
	}

	resp, err := uc.submitter.Submit(ctx, req)
	if err != nil {
		return nil, application.NewUseCaseError("submit", "failed to submit workflow", err)
	}

	out := &SubmitOutput{
		WorkflowID: resp.ID,
		Status:     string(resp.Status),
		Preflight:  report,
	}
	uc.remember(ctx, out, input, req, resp, report)
	return out, nil
}

// recordablePath makes a local path absolute, since a record saying
// "hello.wdl" is worthless read from another directory a month later. Remote
// URIs (gs://, s3://) are already absolute and must be left alone.
func recordablePath(path string) string {
	if path == "" || strings.Contains(path, "://") {
		return path
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}

// remember writes the submission to the local run history, files included.
// The workflow is already running by the time this is called, so a failure
// is reported on the output as a warning rather than returned as the outcome
// of submitting.
func (uc *SubmitUseCase) remember(ctx context.Context, out *SubmitOutput, input SubmitInput, req workflow2.SubmitRequest, resp *workflow2.SubmitResponse, report *PreflightReport) {
	if uc.history == nil || uc.history.Store == nil || uc.history.Host == nil {
		return
	}

	host, alias := uc.history.Host.CurrentHost()
	name := ""
	if report != nil {
		// Preflight already parsed the WDL; skipping it costs the name, not
		// the record.
		name = report.WorkflowName
	}

	files, skipped := uc.history.keptFiles(input, req)
	err := uc.history.Store.Record(ctx, ports.RunRecord{
		Host:             host,
		HostAlias:        alias,
		WorkflowID:       resp.ID,
		Name:             name,
		Description:      input.Description,
		WorkflowFile:     recordablePath(input.WorkflowFile),
		InputsFile:       recordablePath(input.InputsFile),
		OptionsFile:      recordablePath(input.OptionsFile),
		DependenciesFile: recordablePath(input.DependenciesFile),
		Labels:           input.Labels,
		Origin:           ports.OriginSubmit,
		SubmittedAt:      time.Now().UTC(),
		LastStatus:       string(resp.Status),
		StatusSyncedAt:   time.Now().UTC(),
		Files:            files,
	})
	if err != nil {
		out.HistoryError = err
		return
	}
	out.Remembered = true
	out.FilesKept = files != nil
	out.FilesSkipped = skipped
}

// keptFiles turns the request into the files to keep: the very bytes the
// server received, not a second read of the paths, so what is kept is what
// ran. It returns nil — keep nothing — when keeping is off or the files are
// over the cap, with the reason in the second value for the latter.
func (h *SubmitHistory) keptFiles(input SubmitInput, req workflow2.SubmitRequest) ([]ports.RunFile, string) {
	if h.SkipFiles {
		return nil, ""
	}
	limit := h.MaxFileBytes
	if limit <= 0 {
		limit = DefaultMaxKeptBytes
	}

	var files []ports.RunFile
	var total int64
	keep := func(kind ports.RunFileKind, path string, data []byte) {
		if len(data) == 0 {
			return
		}
		files = append(files, ports.RunFile{Kind: kind, Name: filepath.Base(path), Data: data})
		total += int64(len(data))
	}
	keep(ports.RunFileWorkflow, input.WorkflowFile, req.WorkflowSource)
	keep(ports.RunFileInputs, input.InputsFile, req.WorkflowInputs)
	keep(ports.RunFileOptions, input.OptionsFile, req.WorkflowOptions)
	keep(ports.RunFileDependencies, input.DependenciesFile, req.WorkflowDependencies)

	if total > limit {
		return nil, fmt.Sprintf("the submitted files total %s, above the %s kept per run", FormatBytes(total), FormatBytes(limit))
	}
	return files, ""
}

// FormatBytes renders a size the way a person reads one: 512 B, 1.5 KiB,
// 50.0 MiB.
func FormatBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}
