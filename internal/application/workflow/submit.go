package workflow

import (
	"context"

	"github.com/lmtani/pumbaa/internal/application"
	"github.com/lmtani/pumbaa/internal/application/ports"
	workflow2 "github.com/lmtani/pumbaa/internal/domain/workflow"
)

// SubmitUseCase handles workflow submission.
type SubmitUseCase struct {
	submitter    ports.WorkflowSubmitter
	fileProvider ports.FileProvider
	preflight    *PreflightUseCase
}

// NewSubmitUseCase creates a new submit use case. preflight may be nil, in
// which case submissions are not checked before being sent.
func NewSubmitUseCase(submitter ports.WorkflowSubmitter, fileProvider ports.FileProvider, preflight *PreflightUseCase) *SubmitUseCase {
	return &SubmitUseCase{submitter: submitter, fileProvider: fileProvider, preflight: preflight}
}

// SubmitInput represents the input for workflow submission.
type SubmitInput struct {
	WorkflowFile     string
	InputsFile       string
	OptionsFile      string
	DependenciesFile string
	Labels           map[string]string
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

	return &SubmitOutput{
		WorkflowID: resp.ID,
		Status:     string(resp.Status),
		Preflight:  report,
	}, nil
}
