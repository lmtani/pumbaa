package workflow

import (
	"context"
	"errors"
	"testing"

	"github.com/lmtani/pumbaa/internal/domain/workflow"
)

// mockSubmitFileProvider is a test double for FileProvider
type mockSubmitFileProvider struct {
	readBytesFunc func(ctx context.Context, path string) ([]byte, error)
}

func (m *mockSubmitFileProvider) Read(ctx context.Context, path string) (string, error) {
	return "", nil
}

func (m *mockSubmitFileProvider) ReadBytes(ctx context.Context, path string) ([]byte, error) {
	if m.readBytesFunc != nil {
		return m.readBytesFunc(ctx, path)
	}
	return nil, nil
}

func TestSubmitUseCase_Execute(t *testing.T) {
	repo := &mockWorkflowRepository{
		submitFunc: func(ctx context.Context, req workflow.SubmitRequest) (*workflow.SubmitResponse, error) {
			return &workflow.SubmitResponse{ID: "test-id", Status: workflow.StatusSubmitted}, nil
		},
	}
	fp := &mockSubmitFileProvider{
		readBytesFunc: func(ctx context.Context, path string) ([]byte, error) {
			return []byte("workflow test {}"), nil
		},
	}
	uc := NewSubmitUseCase(repo, fp)

	input := SubmitInput{
		WorkflowFile: "test.wdl",
	}
	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.WorkflowID != "test-id" {
		t.Errorf("expected WorkflowID test-id, got %s", output.WorkflowID)
	}
}

func TestSubmitUseCase_Execute_Error(t *testing.T) {
	repo := &mockWorkflowRepository{}
	fp := &mockSubmitFileProvider{
		readBytesFunc: func(ctx context.Context, path string) ([]byte, error) {
			return nil, errors.New("file not found")
		},
	}
	uc := NewSubmitUseCase(repo, fp)

	input := SubmitInput{
		WorkflowFile: "non-existent.wdl",
	}
	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
