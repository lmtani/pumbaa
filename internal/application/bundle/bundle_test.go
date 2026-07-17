package bundle

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/lmtani/pumbaa/internal/application"
	"github.com/lmtani/pumbaa/internal/domain/bundle"
)

func TestUseCase_Execute_Validation(t *testing.T) {
	uc := New()
	ctx := context.Background()

	tests := []struct {
		name      string
		input     Input
		wantField string
	}{
		{
			name:      "empty main workflow path",
			input:     Input{MainWorkflowPath: "", OutputPath: "out"},
			wantField: "mainWorkflowPath",
		},
		{
			name:      "empty output path",
			input:     Input{MainWorkflowPath: "workflow.wdl", OutputPath: ""},
			wantField: "outputPath",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := uc.Execute(ctx, tt.input)
			if err == nil {
				t.Fatalf("Execute() expected error for %s", tt.name)
			}
			if !errors.Is(err, application.ErrInvalidInput) {
				t.Errorf("expected ErrInvalidInput, got %v", err)
			}
			var inputErr *application.InputValidationError
			if !errors.As(err, &inputErr) {
				t.Fatalf("expected InputValidationError, got %T", err)
			}
			if inputErr.Field != tt.wantField {
				t.Errorf("expected field %s, got %s", tt.wantField, inputErr.Field)
			}
		})
	}
}

func TestUseCase_Execute_Success(t *testing.T) {
	// Este teste usa arquivos reais pois o BundleUseCase chama pkg/wdl.CreateBundle diretamente,
	// que por sua vez manipula o sistema de arquivos.

	uc := New()
	ctx := context.Background()

	// Setup: create a simple WDL file
	tmpDir, err := os.MkdirTemp("", "bundle-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	wdlPath := filepath.Join(tmpDir, "main.wdl")
	wdlContent := "version 1.0\nworkflow test {}"
	if err := os.WriteFile(wdlPath, []byte(wdlContent), 0644); err != nil {
		t.Fatalf("failed to write wdl file: %v", err)
	}

	outDir := filepath.Join(tmpDir, "out")
	if err := os.Mkdir(outDir, 0755); err != nil {
		t.Fatalf("failed to create out dir: %v", err)
	}

	input := Input{
		MainWorkflowPath: wdlPath,
		OutputPath:       outDir,
	}

	output, err := uc.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	if output.MainWDLPath == "" {
		t.Error("Execute() expected MainWDLPath to be set")
	}
	if output.TotalFiles != 1 {
		t.Errorf("Execute() TotalFiles = %d, want 1", output.TotalFiles)
	}
}

func TestUseCase_Execute_NotFound(t *testing.T) {
	uc := New()
	ctx := context.Background()

	input := Input{
		MainWorkflowPath: "non-existent.wdl",
		OutputPath:       filepath.Join(t.TempDir(), "out"),
	}

	_, err := uc.Execute(ctx, input)
	if err == nil {
		t.Error("Execute() expected error for non-existent file")
	}
	if !errors.Is(err, application.ErrOperationFailed) {
		t.Errorf("expected ErrOperationFailed, got %v", err)
	}
	if !errors.Is(err, bundle.ErrBundleCreationFailed) {
		t.Errorf("expected ErrBundleCreationFailed, got %v", err)
	}
	var ucErr *application.UseCaseError
	if !errors.As(err, &ucErr) {
		t.Fatalf("expected UseCaseError, got %T", err)
	}
	if ucErr.Operation != "bundle" {
		t.Errorf("expected operation bundle, got %s", ucErr.Operation)
	}
}
