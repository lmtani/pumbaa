// Package workflow contains use cases for workflow management operations.
// This package consolidates common workflow operations (submit, abort, query, metadata)
// following Clean Architecture principles.
package workflow

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/lmtani/pumbaa/internal/domain/ports"
	"github.com/lmtani/pumbaa/internal/domain/workflow"
)

// SubmitUseCase handles workflow submission.
type SubmitUseCase struct {
	repo ports.WorkflowRepository
}

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

	return &SubmitOutput{
		WorkflowID: resp.ID,
		Status:     string(resp.Status),
	}, nil
}

// AbortUseCase handles workflow abortion.
type AbortUseCase struct {
	repo ports.WorkflowRepository
}

// NewAbortUseCase creates a new abort use case.
func NewAbortUseCase(repo ports.WorkflowRepository) *AbortUseCase {
	return &AbortUseCase{repo: repo}
}

// AbortInput represents the input for workflow abortion.
type AbortInput struct {
	WorkflowID string
}

// AbortOutput represents the output of workflow abortion.
type AbortOutput struct {
	WorkflowID string
	Message    string
}

// Execute aborts a running workflow.
func (uc *AbortUseCase) Execute(ctx context.Context, input AbortInput) (*AbortOutput, error) {
	if input.WorkflowID == "" {
		return nil, workflow.ErrInvalidWorkflowID
	}

	// Check current status before aborting
	status, err := uc.repo.GetStatus(ctx, input.WorkflowID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow status: %w", err)
	}

	// Check if workflow is already in terminal state
	switch status {
	case workflow.StatusSucceeded, workflow.StatusFailed, workflow.StatusAborted:
		return nil, workflow.ErrWorkflowAlreadyTerminal
	}

	// Abort the workflow
	if err := uc.repo.Abort(ctx, input.WorkflowID); err != nil {
		return nil, fmt.Errorf("failed to abort workflow: %w", err)
	}

	return &AbortOutput{
		WorkflowID: input.WorkflowID,
		Message:    "Workflow abort requested successfully",
	}, nil
}

// QueryUseCase handles workflow queries.
type QueryUseCase struct {
	repo ports.WorkflowRepository
}

// NewQueryUseCase creates a new query use case.
func NewQueryUseCase(repo ports.WorkflowRepository) *QueryUseCase {
	return &QueryUseCase{repo: repo}
}

// QueryInput represents the input for workflow queries.
type QueryInput struct {
	Name     string
	Status   []string
	Labels   map[string]string
	Page     int
	PageSize int
}

// QueryOutput represents the output of workflow queries.
type QueryOutput struct {
	Workflows  []WorkflowSummary
	TotalCount int
	Page       int
	PageSize   int
}

// WorkflowSummary represents a summary of a workflow for listing.
type WorkflowSummary struct {
	ID          string
	Name        string
	Status      string
	SubmittedAt time.Time
	Start       time.Time
	End         time.Time
}

// Execute queries workflows based on filters.
func (uc *QueryUseCase) Execute(ctx context.Context, input QueryInput) (*QueryOutput, error) {
	// Convert string statuses to domain Status
	statuses := make([]workflow.Status, 0, len(input.Status))
	for _, s := range input.Status {
		statuses = append(statuses, workflow.Status(s))
	}

	filter := workflow.QueryFilter{
		Name:     input.Name,
		Status:   statuses,
		Labels:   input.Labels,
		Page:     input.Page,
		PageSize: input.PageSize,
	}

	result, err := uc.repo.Query(ctx, filter)
	if err != nil {
		return nil, err
	}

	output := &QueryOutput{
		Workflows:  make([]WorkflowSummary, 0, len(result.Workflows)),
		TotalCount: result.TotalCount,
		Page:       input.Page,
		PageSize:   input.PageSize,
	}

	for _, wf := range result.Workflows {
		output.Workflows = append(output.Workflows, WorkflowSummary{
			ID:          wf.ID,
			Name:        wf.Name,
			Status:      string(wf.Status),
			SubmittedAt: wf.SubmittedAt,
			Start:       wf.Start,
			End:         wf.End,
		})
	}

	return output, nil
}

// MetadataUseCase handles workflow metadata retrieval.
type MetadataUseCase struct {
	repo ports.WorkflowRepository
}

// NewMetadataUseCase creates a new metadata use case.
func NewMetadataUseCase(repo ports.WorkflowRepository) *MetadataUseCase {
	return &MetadataUseCase{repo: repo}
}

// MetadataInput represents the input for metadata retrieval.
type MetadataInput struct {
	WorkflowID string
}

// MetadataOutput represents the output of metadata retrieval.
type MetadataOutput struct {
	ID       string
	Name     string
	Status   string
	Start    time.Time
	End      time.Time
	Duration time.Duration
	Labels   map[string]string
	Calls    []CallSummary
	Failures []string
}

// CallSummary represents a summary of a call execution.
type CallSummary struct {
	Name       string
	Status     string
	Start      time.Time
	End        time.Time
	Duration   time.Duration
	Attempt    int
	ShardIndex int
	ReturnCode *int
}

// Execute retrieves metadata for a workflow.
func (uc *MetadataUseCase) Execute(ctx context.Context, input MetadataInput) (*MetadataOutput, error) {
	if input.WorkflowID == "" {
		return nil, workflow.ErrInvalidWorkflowID
	}

	wf, err := uc.repo.GetMetadata(ctx, input.WorkflowID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow metadata: %w", err)
	}

	output := &MetadataOutput{
		ID:       wf.ID,
		Name:     wf.Name,
		Status:   string(wf.Status),
		Start:    wf.Start,
		End:      wf.End,
		Duration: wf.Duration(),
		Labels:   wf.Labels,
		Calls:    make([]CallSummary, 0),
		Failures: make([]string, 0),
	}

	// Extract call summaries
	for callName, calls := range wf.Calls {
		for _, call := range calls {
			duration := time.Duration(0)
			if !call.Start.IsZero() {
				end := call.End
				if end.IsZero() {
					end = time.Now()
				}
				duration = end.Sub(call.Start)
			}

			output.Calls = append(output.Calls, CallSummary{
				Name:       callName,
				Status:     string(call.Status),
				Start:      call.Start,
				End:        call.End,
				Duration:   duration,
				Attempt:    call.Attempt,
				ShardIndex: call.ShardIndex,
				ReturnCode: call.ReturnCode,
			})
		}
	}

	// Extract failure messages
	for _, failure := range wf.Failures {
		output.Failures = append(output.Failures, extractFailureMessages(failure)...)
	}

	return output, nil
}

// extractFailureMessages recursively extracts failure messages.
func extractFailureMessages(failure workflow.Failure) []string {
	messages := []string{failure.Message}
	for _, cause := range failure.CausedBy {
		messages = append(messages, extractFailureMessages(cause)...)
	}
	return messages
}
