// Package metadata contains the use case for retrieving workflow metadata.
package metadata

import (
	"context"
	"fmt"
	"time"

	"github.com/lmtani/pumbaa/internal/domain/ports"
	"github.com/lmtani/pumbaa/internal/domain/workflow"
)

// UseCase handles workflow metadata retrieval.
type UseCase struct {
	repo ports.WorkflowRepository
}

// New creates a new metadata use case.
func New(repo ports.WorkflowRepository) *UseCase {
	return &UseCase{repo: repo}
}

// Input represents the input for the metadata use case.
type Input struct {
	WorkflowID string
}

// Output represents the output of the metadata use case.
type Output struct {
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
func (uc *UseCase) Execute(ctx context.Context, input Input) (*Output, error) {
	if input.WorkflowID == "" {
		return nil, workflow.ErrInvalidWorkflowID
	}

	wf, err := uc.repo.GetMetadata(ctx, input.WorkflowID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow metadata: %w", err)
	}

	output := &Output{
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
