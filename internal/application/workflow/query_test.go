package workflow

import (
	"context"
	"errors"
	"testing"

	"github.com/lmtani/pumbaa/internal/application"
	"github.com/lmtani/pumbaa/internal/domain/workflow"
)

func TestQueryUseCase_Execute(t *testing.T) {
	repo := &mockWorkflowRepository{
		queryFunc: func(ctx context.Context, filter workflow.QueryFilter) (*workflow.QueryResult, error) {
			if filter.Name != "test-wf" {
				t.Errorf("expected name filter test-wf, got %s", filter.Name)
			}
			if filter.Page != 2 || filter.PageSize != 10 {
				t.Errorf("unexpected paging: page=%d size=%d", filter.Page, filter.PageSize)
			}
			if len(filter.Status) != 2 || filter.Status[0] != workflow.StatusRunning || filter.Status[1] != workflow.StatusFailed {
				t.Errorf("unexpected status filter: %v", filter.Status)
			}
			if filter.Labels["team"] != "bio" {
				t.Errorf("unexpected labels: %v", filter.Labels)
			}
			return &workflow.QueryResult{
				Workflows: []workflow.Workflow{
					{ID: "test-id", Name: "test-wf", Status: workflow.StatusSucceeded},
				},
				TotalCount: 1,
			}, nil
		},
	}
	uc := NewQueryUseCase(repo)

	input := QueryInput{
		Name:     "test-wf",
		Status:   []string{string(workflow.StatusRunning), string(workflow.StatusFailed)},
		Labels:   map[string]string{"team": "bio"},
		Page:     2,
		PageSize: 10,
	}
	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(output.Workflows) != 1 {
		t.Errorf("expected 1 workflow, got %d", len(output.Workflows))
	}
}

func TestQueryUseCase_Execute_Error(t *testing.T) {
	repo := &mockWorkflowRepository{
		queryFunc: func(ctx context.Context, filter workflow.QueryFilter) (*workflow.QueryResult, error) {
			return nil, errors.New("query failed")
		},
	}
	uc := NewQueryUseCase(repo)

	_, err := uc.Execute(context.Background(), QueryInput{Name: "test"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, application.ErrOperationFailed) {
		t.Errorf("expected ErrOperationFailed, got %v", err)
	}
	var ucErr *application.UseCaseError
	if !errors.As(err, &ucErr) {
		t.Fatalf("expected UseCaseError, got %T", err)
	}
	if ucErr.Operation != "query" {
		t.Errorf("expected operation query, got %s", ucErr.Operation)
	}
}
