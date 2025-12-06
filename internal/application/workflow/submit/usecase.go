// Package submit contains the use case for submitting workflows.
package submit

import (
	"context"
	"fmt"
	"os"

	"github.com/lmtani/pumbaa/internal/domain/workflow"
)

// UseCase handles workflow submission.
type UseCase struct {
	repo workflow.Repository
}

// New creates a new submit use case.
func New(repo workflow.Repository) *UseCase {
	return &UseCase{repo: repo}
}

// Input represents the input for the submit use case.
type Input struct {
	WorkflowFile     string
	InputsFile       string
	OptionsFile      string
	DependenciesFile string
	Labels           map[string]string
}

// Output represents the output of the submit use case.
type Output struct {
	WorkflowID string
	Status     string
}

// Execute submits a workflow to Cromwell.
func (uc *UseCase) Execute(ctx context.Context, input Input) (*Output, error) {
	// Read workflow source
	workflowSource, err := os.ReadFile(input.WorkflowFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow file: %w", err)
	}

	// Read optional files
	var inputsData, optionsData, depsData []byte

	if input.InputsFile != "" {
		inputsData, err = os.ReadFile(input.InputsFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read inputs file: %w", err)
		}
	}

	if input.OptionsFile != "" {
		optionsData, err = os.ReadFile(input.OptionsFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read options file: %w", err)
		}
	}

	if input.DependenciesFile != "" {
		depsData, err = os.ReadFile(input.DependenciesFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read dependencies file: %w", err)
		}
	}

	req := workflow.SubmitRequest{
		WorkflowSource:       workflowSource,
		WorkflowInputs:       inputsData,
		WorkflowOptions:      optionsData,
		WorkflowDependencies: depsData,
		Labels:               input.Labels,
		WorkflowType:         "WDL",
		WorkflowTypeVersion:  "1.0",
	}

	resp, err := uc.repo.Submit(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to submit workflow: %w", err)
	}

	return &Output{
		WorkflowID: resp.ID,
		Status:     string(resp.Status),
	}, nil
}
