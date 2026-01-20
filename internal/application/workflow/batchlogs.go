// Package workflow contains use cases for workflow management operations.
package workflow

import (
	"context"
	"strings"
	"time"

	"github.com/lmtani/pumbaa/internal/application"
	"github.com/lmtani/pumbaa/internal/application/ports"
)

// GetBatchLogsUseCase handles retrieval of Google Batch job logs.
type GetBatchLogsUseCase struct {
	logsRepo ports.BatchLogsRepository
}

// NewGetBatchLogsUseCase creates a new batch logs use case.
func NewGetBatchLogsUseCase(logsRepo ports.BatchLogsRepository) *GetBatchLogsUseCase {
	return &GetBatchLogsUseCase{logsRepo: logsRepo}
}

// GetBatchLogsInput represents the input for batch logs retrieval.
type GetBatchLogsInput struct {
	// JobName is the full resource name of the job:
	// projects/{project}/locations/{location}/jobs/{jobId}
	JobName string

	// MinSeverity filters logs by minimum severity.
	// Valid: "DEFAULT", "DEBUG", "INFO", "WARNING", "ERROR", "CRITICAL"
	// Empty string defaults to "INFO".
	MinSeverity string

	// Limit is the maximum number of log entries to return.
	// If 0 or negative, defaults to 300.
	Limit int

	// StartTime optionally filters logs by minimum timestamp.
	// Zero value means no lower bound.
	StartTime time.Time

	// EndTime optionally filters logs by maximum timestamp.
	// Zero value means no upper bound.
	EndTime time.Time
}

// GetBatchLogsOutput represents the result of batch logs retrieval.
type GetBatchLogsOutput struct {
	// Entries are the formatted log entries in ascending order (oldest first).
	Entries []ports.BatchLogEntry

	// JobID is the short job ID extracted from the resource name.
	JobID string
}

// Execute retrieves and formats Google Batch logs for a job.
// Validates the job name format and returns domain entities directly.
func (uc *GetBatchLogsUseCase) Execute(ctx context.Context, input GetBatchLogsInput) (*GetBatchLogsOutput, error) {
	// Validate job name format
	jobID, err := parseJobName(input.JobName)
	if err != nil {
		return nil, application.NewInputValidationError("jobName", err.Error())
	}

	// Normalize severity
	// Default to DEFAULT (lowest severity) to capture all logs
	minSeverity := input.MinSeverity
	if minSeverity == "" {
		minSeverity = "DEFAULT"
	}

	// Normalize limit
	limit := input.Limit
	if limit <= 0 {
		limit = 300
	}

	// Build filter
	filter := ports.BatchLogsFilter{
		MinSeverity: minSeverity,
		Limit:       limit,
		StartTime:   input.StartTime,
		EndTime:     input.EndTime,
	}

	// Call the port to retrieve logs
	entries, err := uc.logsRepo.GetLogs(ctx, input.JobName, filter)
	if err != nil {
		return nil, application.NewUseCaseError("batch-logs", "failed to fetch batch logs", err)
	}

	return &GetBatchLogsOutput{
		Entries: entries,
		JobID:   jobID,
	}, nil
}

// parseJobName extracts the job ID from a full resource name.
// Expected format: projects/{project}/locations/{location}/jobs/{jobId}
// Returns the jobId (last component after final /).
func parseJobName(jobName string) (string, error) {
	// Must have both required path components
	if !strings.HasPrefix(jobName, "projects/") {
		return "", ports.ErrInvalidJobName
	}
	if !strings.Contains(jobName, "/locations/") {
		return "", ports.ErrInvalidJobName
	}
	if !strings.Contains(jobName, "/jobs/") {
		return "", ports.ErrInvalidJobName
	}

	// Extract job ID (last component)
	parts := strings.Split(jobName, "/")
	if len(parts) < 6 {
		return "", ports.ErrInvalidJobName
	}

	// Format is: projects/{p}/locations/{l}/jobs/{j}
	// parts[0]="projects", parts[1]={p}, parts[2]="locations", parts[3]={l}, parts[4]="jobs", parts[5]={j}
	jobID := parts[5]
	if jobID == "" {
		return "", ports.ErrInvalidJobName
	}

	return jobID, nil
}
