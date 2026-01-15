package workflow

import (
	"context"

	"github.com/lmtani/pumbaa/internal/application"
	"github.com/lmtani/pumbaa/internal/domain/ports"
	workflow2 "github.com/lmtani/pumbaa/internal/domain/workflow"
)

// SubmitUseCase handles workflow submission.
type SubmitUseCase struct {
	submitter    ports.WorkflowSubmitter
	fileProvider ports.FileProvider
}

// NewSubmitUseCase creates a new submit use case.
func NewSubmitUseCase(submitter ports.WorkflowSubmitter, fileProvider ports.FileProvider) *SubmitUseCase {
	return &SubmitUseCase{submitter: submitter, fileProvider: fileProvider}
}

// SubmitInput represents the input for workflow submission.
type SubmitInput struct {
	WorkflowFile     string
	InputsFile       string
	OptionsFile      string
	DependenciesFile string
	Labels           map[string]string
}

// SubmitOutput represents the output of workflow submission.
type SubmitOutput struct {
	WorkflowID string
	Status     string
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
	}, nil
}
