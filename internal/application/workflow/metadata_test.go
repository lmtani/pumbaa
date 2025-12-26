package workflow

import (
	"context"
	"testing"

	"github.com/lmtani/pumbaa/internal/domain/workflow"
)

func TestMetadataUseCase_Execute_WithFailures(t *testing.T) {
	repo := &mockWorkflowRepository{
		getMetadataFunc: func(ctx context.Context, workflowID string) (*workflow.Workflow, error) {
			return &workflow.Workflow{
				ID:   "test-id",
				Name: "test-wf",
				Failures: []workflow.Failure{
					{Message: "root failure", CausedBy: []workflow.Failure{{Message: "cause"}}},
				},
				Calls: map[string][]workflow.Call{
					"task1": {{Name: "task1", Status: workflow.StatusFailed, Failures: []workflow.Failure{{Message: "task failure"}}}},
				},
			}, nil
		},
	}
	uc := NewMetadataUseCase(repo)

	input := MetadataInput{WorkflowID: "test-id"}
	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(output.Failures) == 0 {
		t.Error("expected failures in output")
	}
}
