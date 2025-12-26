package workflow

import (
	"context"
	"fmt"
	"time"

	"github.com/lmtani/pumbaa/internal/domain/ports"
	workflow2 "github.com/lmtani/pumbaa/internal/domain/workflow"
)

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
		return nil, workflow2.ErrInvalidWorkflowID
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
func extractFailureMessages(failure workflow2.Failure) []string {
	messages := []string{failure.Message}
	for _, cause := range failure.CausedBy {
		messages = append(messages, extractFailureMessages(cause)...)
	}
	return messages
}
