package workflow

import (
	"context"
	"errors"
	"testing"

	"github.com/lmtani/pumbaa/internal/application"
	"github.com/lmtani/pumbaa/internal/domain/workflow"
)

// mockWorkflowRepository and mockFileProvider are defined in testutil_test.go

func TestSubmitUseCase_Execute(t *testing.T) {
	repo := &mockWorkflowRepository{
		submitFunc: func(ctx context.Context, req workflow.SubmitRequest) (*workflow.SubmitResponse, error) {
			if string(req.WorkflowSource) != "workflow test {}" {
				t.Errorf("unexpected workflow source: %s", string(req.WorkflowSource))
			}
			if string(req.WorkflowInputs) != `{"key":"value"}` {
				t.Errorf("unexpected workflow inputs: %s", string(req.WorkflowInputs))
			}
			if string(req.WorkflowOptions) != `{"opt":"value"}` {
				t.Errorf("unexpected workflow options: %s", string(req.WorkflowOptions))
			}
			if string(req.WorkflowDependencies) != "deps" {
				t.Errorf("unexpected workflow dependencies: %s", string(req.WorkflowDependencies))
			}
			if req.Labels["team"] != "bio" {
				t.Errorf("unexpected labels: %v", req.Labels)
			}
			if req.WorkflowType != "WDL" || req.WorkflowTypeVersion != "1.0" {
				t.Errorf("unexpected workflow type/version: %s/%s", req.WorkflowType, req.WorkflowTypeVersion)
			}
			return &workflow.SubmitResponse{ID: "test-id", Status: workflow.StatusSubmitted}, nil
		},
	}
	fp := &mockFileProvider{
		readBytesFunc: func(ctx context.Context, path string) ([]byte, error) {
			switch path {
			case "test.wdl":
				return []byte("workflow test {}"), nil
			case "inputs.json":
				return []byte(`{"key":"value"}`), nil
			case "options.json":
				return []byte(`{"opt":"value"}`), nil
			case "deps.zip":
				return []byte("deps"), nil
			default:
				return nil, errors.New("unexpected path")
			}
		},
	}
	uc := NewSubmitUseCase(repo, fp)

	input := SubmitInput{
		WorkflowFile:     "test.wdl",
		InputsFile:       "inputs.json",
		OptionsFile:      "options.json",
		DependenciesFile: "deps.zip",
		Labels:           map[string]string{"team": "bio"},
	}
	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.WorkflowID != "test-id" {
		t.Errorf("expected WorkflowID test-id, got %s", output.WorkflowID)
	}
}

func TestSubmitUseCase_Execute_Validation(t *testing.T) {
	repo := &mockWorkflowRepository{}
	fp := &mockFileProvider{}
	uc := NewSubmitUseCase(repo, fp)

	_, err := uc.Execute(context.Background(), SubmitInput{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, application.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
	var inputErr *application.InputValidationError
	if !errors.As(err, &inputErr) {
		t.Fatalf("expected InputValidationError, got %T", err)
	}
	if inputErr.Field != "workflowFile" {
		t.Errorf("expected field workflowFile, got %s", inputErr.Field)
	}
}

func TestSubmitUseCase_Execute_Error(t *testing.T) {
	repo := &mockWorkflowRepository{}
	fp := &mockFileProvider{
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
	if !errors.Is(err, application.ErrOperationFailed) {
		t.Errorf("expected ErrOperationFailed, got %v", err)
	}
	var ucErr *application.UseCaseError
	if !errors.As(err, &ucErr) {
		t.Fatalf("expected UseCaseError, got %T", err)
	}
	if ucErr.Operation != "submit" {
		t.Errorf("expected operation submit, got %s", ucErr.Operation)
	}
}

func TestSubmitUseCase_Execute_OptionalFileReadErrors(t *testing.T) {
	tests := []struct {
		name          string
		input         SubmitInput
		errorPath     string
		expectMessage string
	}{
		{
			name:          "inputs file",
			input:         SubmitInput{WorkflowFile: "test.wdl", InputsFile: "inputs.json"},
			errorPath:     "inputs.json",
			expectMessage: "failed to read inputs file",
		},
		{
			name:          "options file",
			input:         SubmitInput{WorkflowFile: "test.wdl", OptionsFile: "options.json"},
			errorPath:     "options.json",
			expectMessage: "failed to read options file",
		},
		{
			name:          "dependencies file",
			input:         SubmitInput{WorkflowFile: "test.wdl", DependenciesFile: "deps.zip"},
			errorPath:     "deps.zip",
			expectMessage: "failed to read dependencies file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockWorkflowRepository{}
			fp := &mockFileProvider{
				readBytesFunc: func(ctx context.Context, path string) ([]byte, error) {
					if path == tt.errorPath {
						return nil, errors.New("read error")
					}
					return []byte("ok"), nil
				},
			}
			uc := NewSubmitUseCase(repo, fp)

			_, err := uc.Execute(context.Background(), tt.input)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			var ucErr *application.UseCaseError
			if !errors.As(err, &ucErr) {
				t.Fatalf("expected UseCaseError, got %T", err)
			}
			if ucErr.Message != tt.expectMessage {
				t.Errorf("expected message %q, got %q", tt.expectMessage, ucErr.Message)
			}
		})
	}
}

func TestSubmitUseCase_Execute_SubmitError(t *testing.T) {
	repo := &mockWorkflowRepository{
		submitFunc: func(ctx context.Context, req workflow.SubmitRequest) (*workflow.SubmitResponse, error) {
			return nil, errors.New("submit failed")
		},
	}
	fp := &mockFileProvider{
		readBytesFunc: func(ctx context.Context, path string) ([]byte, error) {
			return []byte("workflow test {}"), nil
		},
	}
	uc := NewSubmitUseCase(repo, fp)

	input := SubmitInput{WorkflowFile: "test.wdl"}
	_, err := uc.Execute(context.Background(), input)
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
	if ucErr.Operation != "submit" {
		t.Errorf("expected operation submit, got %s", ucErr.Operation)
	}
}
