// Package ports contains domain port (interface) definitions.
package ports

import (
	"context"
	"time"
)

// BatchLogsRepository is the port for fetching Google Batch job logs.
// Implementations should handle Cloud Logging API interactions without
// exposing infrastructure details to the domain.
type BatchLogsRepository interface {
	// GetLogs retrieves formatted log entries for a Google Batch job.
	// jobName must be the full resource name: projects/{project}/locations/{location}/jobs/{jobId}
	// Returns logs in ascending order by timestamp (oldest first).
	GetLogs(ctx context.Context, jobName string, filter BatchLogsFilter) ([]BatchLogEntry, error)
}

// BatchLogsFilter specifies criteria for retrieving batch logs.
type BatchLogsFilter struct {
	// MinSeverity filters logs by minimum severity level.
	// Valid values: "DEFAULT", "DEBUG", "INFO", "WARNING", "ERROR", "CRITICAL"
	// Empty string defaults to "INFO".
	MinSeverity string

	// Limit is the maximum number of log entries to return.
	// If 0, defaults to 300.
	Limit int

	// StartTime optionally filters logs by minimum timestamp (inclusive).
	// Zero value means no lower bound.
	StartTime time.Time

	// EndTime optionally filters logs by maximum timestamp (inclusive).
	// Zero value means no upper bound.
	EndTime time.Time
}

// BatchLogEntry represents a cleaned, formatted log entry from Google Batch.
// It is a Value Object: immutable and compared by value.
type BatchLogEntry struct {
	// Timestamp when the log entry was recorded.
	Timestamp time.Time

	// Severity of the log entry (e.g., "INFO", "ERROR", "WARNING").
	Severity string

	// Message is the cleaned, human-readable message extracted from the log payload.
	// Priority: textPayload > jsonPayload.message > truncated payload representation.
	Message string
}
