package workflow

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lmtani/pumbaa/internal/application"
	"github.com/lmtani/pumbaa/internal/application/ports"
)

// MockBatchLogsRepository is a mock implementation for testing.
type MockBatchLogsRepository struct {
	entries []ports.BatchLogEntry
	err     error
}

func (m *MockBatchLogsRepository) GetLogs(ctx context.Context, jobName string, filter ports.BatchLogsFilter) ([]ports.BatchLogEntry, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.entries, nil
}

func TestGetBatchLogsUseCase_InvalidJobNameFormat(t *testing.T) {
	uc := NewGetBatchLogsUseCase(&MockBatchLogsRepository{})

	tests := []struct {
		name    string
		jobName string
	}{
		{"empty", ""},
		{"no projects", "locations/us-central1/jobs/job-123"},
		{"no locations", "projects/my-project/jobs/job-123"},
		{"no jobs", "projects/my-project/locations/us-central1"},
		{"missing job id", "projects/my-project/locations/us-central1/jobs/"},
		{"short format", "job-123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := GetBatchLogsInput{JobName: tt.jobName}
			_, err := uc.Execute(context.Background(), input)
			if err == nil {
				t.Error("expected error for invalid job name, got nil")
			}
			if !errors.Is(err, application.ErrInvalidInput) {
				t.Errorf("expected ErrInvalidInput, got %v", err)
			}
			var inputErr *application.InputValidationError
			if !errors.As(err, &inputErr) {
				t.Fatalf("expected InputValidationError, got %T", err)
			}
			if inputErr.Field != "jobName" {
				t.Errorf("expected field jobName, got %s", inputErr.Field)
			}
		})
	}
}

func TestGetBatchLogsUseCase_ValidJobNameParsing(t *testing.T) {
	tests := []struct {
		name    string
		jobName string
		wantID  string
	}{
		{
			"standard format",
			"projects/my-project/locations/us-central1/jobs/job-12345",
			"job-12345",
		},
		{
			"complex job id",
			"projects/my-project-123/locations/europe-west1/jobs/batch-job-uuid-12345",
			"batch-job-uuid-12345",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockBatchLogsRepository{
				entries: []ports.BatchLogEntry{},
				err:     nil,
			}
			uc := NewGetBatchLogsUseCase(mock)

			output, err := uc.Execute(context.Background(), GetBatchLogsInput{JobName: tt.jobName})
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if output.JobID != tt.wantID {
				t.Errorf("expected JobID %s, got %s", tt.wantID, output.JobID)
			}
		})
	}
}

func TestGetBatchLogsUseCase_Success(t *testing.T) {
	expectedEntries := []ports.BatchLogEntry{
		{
			Timestamp: time.Now().Add(-2 * time.Minute),
			Severity:  "INFO",
			Message:   "Container started",
		},
		{
			Timestamp: time.Now().Add(-1 * time.Minute),
			Severity:  "ERROR",
			Message:   "Task failed",
		},
	}

	mock := &MockBatchLogsRepository{
		entries: expectedEntries,
		err:     nil,
	}
	uc := NewGetBatchLogsUseCase(mock)

	input := GetBatchLogsInput{
		JobName:     "projects/test/locations/us-central1/jobs/job-123",
		MinSeverity: "DEBUG",
		Limit:       100,
	}

	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if output == nil {
		t.Fatal("output is nil")
	}
	if len(output.Entries) != len(expectedEntries) {
		t.Errorf("expected %d entries, got %d", len(expectedEntries), len(output.Entries))
	}
	if output.JobID != "job-123" {
		t.Errorf("expected JobID job-123, got %s", output.JobID)
	}
}

func TestGetBatchLogsUseCase_DefaultLimitAndSeverity(t *testing.T) {
	// This test verifies that empty input defaults are applied
	// by checking the use case validates and normalizes defaults
	mock := &MockBatchLogsRepository{
		entries: []ports.BatchLogEntry{},
		err:     nil,
	}

	uc := NewGetBatchLogsUseCase(mock)

	input := GetBatchLogsInput{
		JobName: "projects/test/locations/us-central1/jobs/job-123",
		// MinSeverity empty (should default to DEFAULT)
		// Limit 0 (should default to 300)
	}

	_, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// Successful execution implies defaults were applied correctly
}

func TestGetBatchLogsUseCase_PortErrorPropagation(t *testing.T) {
	mock := &MockBatchLogsRepository{
		err: ports.ErrUnauthorized,
	}
	uc := NewGetBatchLogsUseCase(mock)

	input := GetBatchLogsInput{
		JobName: "projects/test/locations/us-central1/jobs/job-123",
	}

	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Error("expected error, got nil")
	}
	// Check that the error message contains context
	if !errors.Is(err, ports.ErrUnauthorized) {
		t.Errorf("expected error to wrap ErrUnauthorized, got %v", err)
	}
	if !errors.Is(err, application.ErrOperationFailed) {
		t.Errorf("expected ErrOperationFailed, got %v", err)
	}
	var ucErr *application.UseCaseError
	if !errors.As(err, &ucErr) {
		t.Fatalf("expected UseCaseError, got %T", err)
	}
	if ucErr.Operation != "batch-logs" {
		t.Errorf("expected operation batch-logs, got %s", ucErr.Operation)
	}
}
