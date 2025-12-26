package workflow

import (
	"context"
	"os"
	"testing"

	"github.com/lmtani/pumbaa/internal/domain/workflow"
)

type mockWorkflowRepository struct {
	submitFunc      func(ctx context.Context, req workflow.SubmitRequest) (*workflow.SubmitResponse, error)
	abortFunc       func(ctx context.Context, workflowID string) error
	queryFunc       func(ctx context.Context, filter workflow.QueryFilter) (*workflow.QueryResult, error)
	getMetadataFunc func(ctx context.Context, workflowID string) (*workflow.Workflow, error)
	getStatusFunc   func(ctx context.Context, workflowID string) (workflow.Status, error)
}

func (m *mockWorkflowRepository) Submit(ctx context.Context, req workflow.SubmitRequest) (*workflow.SubmitResponse, error) {
	return m.submitFunc(ctx, req)
}

func (m *mockWorkflowRepository) Abort(ctx context.Context, workflowID string) error {
	return m.abortFunc(ctx, workflowID)
}

func (m *mockWorkflowRepository) Query(ctx context.Context, filter workflow.QueryFilter) (*workflow.QueryResult, error) {
	return m.queryFunc(ctx, filter)
}

func (m *mockWorkflowRepository) GetMetadata(ctx context.Context, workflowID string) (*workflow.Workflow, error) {
	return m.getMetadataFunc(ctx, workflowID)
}

// Implement other methods of WorkflowRepository with no-ops or panics if not needed for these tests
func (m *mockWorkflowRepository) GetRawMetadataWithOptions(ctx context.Context, workflowID string, expandSubWorkflows bool) ([]byte, error) {
	return nil, nil
}
func (m *mockWorkflowRepository) GetStatus(ctx context.Context, workflowID string) (workflow.Status, error) {
	if m.getStatusFunc != nil {
		return m.getStatusFunc(ctx, workflowID)
	}
	return workflow.StatusRunning, nil
}
func (m *mockWorkflowRepository) GetOutputs(ctx context.Context, workflowID string) (map[string]interface{}, error) {
	return nil, nil
}
func (m *mockWorkflowRepository) GetLogs(ctx context.Context, workflowID string) (map[string][]workflow.CallLog, error) {
	return nil, nil
}
func (m *mockWorkflowRepository) GetWorkflowCost(ctx context.Context, workflowID string) (float64, string, error) {
	return 0, "", nil
}
func (m *mockWorkflowRepository) GetHealthStatus(ctx context.Context) (*workflow.HealthStatus, error) {
	return nil, nil
}
func (m *mockWorkflowRepository) GetLabels(ctx context.Context, workflowID string) (map[string]string, error) {
	return nil, nil
}
func (m *mockWorkflowRepository) UpdateLabels(ctx context.Context, workflowID string, labels map[string]string) error {
	return nil
}

func TestSubmitUseCase_Execute(t *testing.T) {
	repo := &mockWorkflowRepository{
		submitFunc: func(ctx context.Context, req workflow.SubmitRequest) (*workflow.SubmitResponse, error) {
			return &workflow.SubmitResponse{ID: "test-id", Status: workflow.StatusSubmitted}, nil
		},
	}
	uc := NewSubmitUseCase(repo)

	// Create a temporary file for the workflow
	tmpFile, err := os.CreateTemp("", "workflow-*.wdl")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.WriteString("workflow test {}"); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	input := SubmitInput{
		WorkflowFile: tmpFile.Name(),
	}
	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.WorkflowID != "test-id" {
		t.Errorf("expected WorkflowID test-id, got %s", output.WorkflowID)
	}
}

func TestAbortUseCase_Execute(t *testing.T) {
	repo := &mockWorkflowRepository{
		abortFunc: func(ctx context.Context, workflowID string) error {
			return nil
		},
	}
	uc := NewAbortUseCase(repo)

	input := AbortInput{WorkflowID: "test-id"}
	_, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

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

func TestSubmitUseCase_Execute_Error(t *testing.T) {
	repo := &mockWorkflowRepository{}
	uc := NewSubmitUseCase(repo)

	input := SubmitInput{
		WorkflowFile: "non-existent.wdl",
	}
	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestAbortUseCase_Execute_TerminalState(t *testing.T) {
	// Create a specific mock for this test
	repoWithTerminalStatus := &mockWorkflowRepository{
		getStatusFunc: func(ctx context.Context, workflowID string) (workflow.Status, error) {
			return workflow.StatusSucceeded, nil
		},
	}

	uc := NewAbortUseCase(repoWithTerminalStatus)

	input := AbortInput{WorkflowID: "test-id"}
	_, err := uc.Execute(context.Background(), input)
	if err != workflow.ErrWorkflowAlreadyTerminal {
		t.Errorf("expected ErrWorkflowAlreadyTerminal, got %v", err)
	}
}

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
