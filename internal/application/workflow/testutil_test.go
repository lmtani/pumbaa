package workflow

import (
	"context"

	"github.com/lmtani/pumbaa/internal/domain/workflow"
)

// =============================================================================
// Mock WorkflowRepository
// =============================================================================

// mockWorkflowRepository is a test double for ports.WorkflowRepository.
// Configure the *Func fields to control behavior in tests.
type mockWorkflowRepository struct {
	submitFunc      func(ctx context.Context, req workflow.SubmitRequest) (*workflow.SubmitResponse, error)
	abortFunc       func(ctx context.Context, workflowID string) error
	queryFunc       func(ctx context.Context, filter workflow.QueryFilter) (*workflow.QueryResult, error)
	getMetadataFunc func(ctx context.Context, workflowID string) (*workflow.Workflow, error)
	getStatusFunc   func(ctx context.Context, workflowID string) (workflow.Status, error)
}

func (m *mockWorkflowRepository) Submit(ctx context.Context, req workflow.SubmitRequest) (*workflow.SubmitResponse, error) {
	if m.submitFunc != nil {
		return m.submitFunc(ctx, req)
	}
	return nil, nil
}

func (m *mockWorkflowRepository) Abort(ctx context.Context, workflowID string) error {
	if m.abortFunc != nil {
		return m.abortFunc(ctx, workflowID)
	}
	return nil
}

func (m *mockWorkflowRepository) Query(ctx context.Context, filter workflow.QueryFilter) (*workflow.QueryResult, error) {
	if m.queryFunc != nil {
		return m.queryFunc(ctx, filter)
	}
	return nil, nil
}

func (m *mockWorkflowRepository) GetMetadata(ctx context.Context, workflowID string) (*workflow.Workflow, error) {
	if m.getMetadataFunc != nil {
		return m.getMetadataFunc(ctx, workflowID)
	}
	return nil, nil
}

func (m *mockWorkflowRepository) GetStatus(ctx context.Context, workflowID string) (workflow.Status, error) {
	if m.getStatusFunc != nil {
		return m.getStatusFunc(ctx, workflowID)
	}
	return workflow.StatusRunning, nil
}

// Stub implementations for interface compliance
func (m *mockWorkflowRepository) GetRawMetadataWithOptions(ctx context.Context, workflowID string, expandSubWorkflows bool) ([]byte, error) {
	return nil, nil
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

// =============================================================================
// Mock FileProvider
// =============================================================================

// mockFileProvider is a test double for ports.FileProvider.
// Configure the *Func fields to control behavior in tests.
type mockFileProvider struct {
	readFunc      func(ctx context.Context, path string) (string, error)
	readBytesFunc func(ctx context.Context, path string) ([]byte, error)
}

func (m *mockFileProvider) Read(ctx context.Context, path string) (string, error) {
	if m.readFunc != nil {
		return m.readFunc(ctx, path)
	}
	return "", nil
}

func (m *mockFileProvider) ReadBytes(ctx context.Context, path string) ([]byte, error) {
	if m.readBytesFunc != nil {
		return m.readBytesFunc(ctx, path)
	}
	return nil, nil
}
