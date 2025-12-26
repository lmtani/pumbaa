package workflow

import (
	"context"
	"fmt"
	"os"

	"github.com/lmtani/pumbaa/internal/domain/ports"
	workflow2 "github.com/lmtani/pumbaa/internal/domain/workflow"
)

// NewSubmitUseCase creates a new submit use case.
func NewSubmitUseCase(repo ports.WorkflowRepository) *SubmitUseCase {
	return &SubmitUseCase{repo: repo}
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

	req := workflow2.SubmitRequest{
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

	return &SubmitOutput{
		WorkflowID: resp.ID,
		Status:     string(resp.Status),
	}, nil
}

// SubmitUseCase handles workflow submission.
type SubmitUseCase struct {
	repo ports.WorkflowRepository
}
