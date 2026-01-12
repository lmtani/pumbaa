package workflow

import (
	"context"
	"errors"
	"testing"

	"github.com/lmtani/pumbaa/internal/application"
)

// mockFileProvider is defined in testutil_test.go

func TestMonitoringUseCase_Execute(t *testing.T) {
	tsvContent := "timestamp\tcpu_percent\tmem_used_mb\tmem_total_mb\tdisk_used_gb\tdisk_total_gb\n2023-01-01 00:00:00\t10.0\t20.0\t100.0\t5.0\t50.0\n2023-01-01 00:01:00\t15.0\t25.0\t100.0\t6.0\t50.0"

	fileProvider := &mockFileProvider{
		readFunc: func(ctx context.Context, path string) (string, error) {
			return tsvContent, nil
		},
	}
	uc := NewMonitoringUseCase(fileProvider)

	input := MonitoringInput{LogPath: "gs://bucket/logs/monitoring.tsv"}
	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.Report == nil {
		t.Fatal("expected a report, got nil")
	}
}

func TestMonitoringUseCase_Execute_Validation(t *testing.T) {
	uc := NewMonitoringUseCase(&mockFileProvider{})

	_, err := uc.Execute(context.Background(), MonitoringInput{})
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
	if inputErr.Field != "logPath" {
		t.Errorf("expected field logPath, got %s", inputErr.Field)
	}
}

func TestMonitoringUseCase_Execute_ReadError(t *testing.T) {
	fp := &mockFileProvider{
		readFunc: func(ctx context.Context, path string) (string, error) {
			return "", errors.New("read failed")
		},
	}
	uc := NewMonitoringUseCase(fp)

	_, err := uc.Execute(context.Background(), MonitoringInput{LogPath: "gs://bucket/logs/monitoring.tsv"})
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
	if ucErr.Operation != "monitoring" {
		t.Errorf("expected operation monitoring, got %s", ucErr.Operation)
	}
}

func TestMonitoringUseCase_Execute_ParseError(t *testing.T) {
	fp := &mockFileProvider{
		readFunc: func(ctx context.Context, path string) (string, error) {
			return "timestamp\tcpu_percent\n", nil
		},
	}
	uc := NewMonitoringUseCase(fp)

	_, err := uc.Execute(context.Background(), MonitoringInput{LogPath: "gs://bucket/logs/monitoring.tsv"})
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
	if ucErr.Operation != "monitoring" {
		t.Errorf("expected operation monitoring, got %s", ucErr.Operation)
	}
}
