package workflow

import (
	"context"
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
}

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
	out.Remembered, out.HistoryError = uc.remember(ctx, input, resp, report)
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

// remember writes the submission to the local run history. The workflow is
// already running by the time this is called, so its failure is reported to
// the caller as a warning rather than returned as the outcome of submitting.
func (uc *SubmitUseCase) remember(ctx context.Context, input SubmitInput, resp *workflow2.SubmitResponse, report *PreflightReport) (bool, error) {
	if uc.history == nil || uc.history.Store == nil || uc.history.Host == nil {
		return false, nil
	}

	host, alias := uc.history.Host.CurrentHost()
	name := ""
	if report != nil {
		// Preflight already parsed the WDL; skipping it costs the name, not
		// the record.
		name = report.WorkflowName
	}

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
	})
	if err != nil {
		return false, err
	}
	return true, nil
}
