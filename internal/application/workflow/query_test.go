package workflow

import (
	"context"
	"testing"

	"github.com/lmtani/pumbaa/internal/domain/workflow"
)

func TestQueryUseCase_Execute(t *testing.T) {
	repo := &mockWorkflowRepository{
		queryFunc: func(ctx context.Context, filter workflow.QueryFilter) (*workflow.QueryResult, error) {
			return &workflow.QueryResult{
				Workflows: []workflow.Workflow{
					{ID: "test-id", Name: "test-wf", Status: workflow.StatusSucceeded},
				},
				TotalCount: 1,
			}, nil
		},
	}
	uc := NewQueryUseCase(repo)

	input := QueryInput{Name: "test-wf"}
	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(output.Workflows) != 1 {
		t.Errorf("expected 1 workflow, got %d", len(output.Workflows))
	}
}
