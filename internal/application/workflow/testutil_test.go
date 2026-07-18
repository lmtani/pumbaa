package workflow

import (
	"context"
	"sync"

	"github.com/lmtani/pumbaa/internal/domain/workflow"
)

// =============================================================================
// Mock WorkflowRepository
// =============================================================================

// mockWorkflowRepository is a test double for workflow ports used by use cases.
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

// =============================================================================
// Mock FileProvider
// =============================================================================

// mockFileProvider is a test double for ports.FileProvider.
// Configure the *Func fields to control behavior in tests.
type mockFileProvider struct {
	readFunc           func(ctx context.Context, path string) (string, error)
	readBytesFunc      func(ctx context.Context, path string) ([]byte, error)
	getSizeFunc        func(ctx context.Context, path string) (int64, error)
	getContentHashFunc func(ctx context.Context, path string) (string, error)
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

func (m *mockFileProvider) GetSize(ctx context.Context, path string) (int64, error) {
	if m.getSizeFunc != nil {
		return m.getSizeFunc(ctx, path)
	}
	return 0, nil
}

func (m *mockFileProvider) GetContentHash(ctx context.Context, path string) (string, error) {
	if m.getContentHashFunc != nil {
		return m.getContentHashFunc(ctx, path)
	}
	return "", nil
}

// =============================================================================
// Mock TaskMetricsWriter
// =============================================================================

// mockTaskMetricsWriter is a test double for ports.TaskMetricsWriter.
// Configure the writeFunc to control behavior in tests.
type mockTaskMetricsWriter struct {
	writeFunc func(filename string, metrics []workflow.TaskMetrics) error
}

func (m *mockTaskMetricsWriter) WriteToFile(filename string, metrics []workflow.TaskMetrics) error {
	if m.writeFunc != nil {
		return m.writeFunc(filename, metrics)
	}
	return nil
}

// =============================================================================
// Mock FileSizeCache
// =============================================================================

// mockFileSizeCache is a test double for ports.FileSizeCache.
// Configure the *Func fields to control behavior in tests.
type mockFileSizeCache struct {
	mu       sync.RWMutex
	sizes    map[string]int64
	loadFunc func() error
	saveFunc func() error
}

func (m *mockFileSizeCache) Load() error {
	if m.loadFunc != nil {
		return m.loadFunc()
	}
	return nil
}

func (m *mockFileSizeCache) Save() error {
	if m.saveFunc != nil {
		return m.saveFunc()
	}
	return nil
}

func (m *mockFileSizeCache) Get(path string) (int64, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.sizes == nil {
		return 0, false
	}
	size, ok := m.sizes[path]
	return size, ok
}

func (m *mockFileSizeCache) Set(path string, size int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sizes == nil {
		m.sizes = make(map[string]int64)
	}
	m.sizes[path] = size
}
