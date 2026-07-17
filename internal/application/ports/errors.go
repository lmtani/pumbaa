// Package ports contains error definitions for port interfaces.
package ports

import (
	"errors"
	"fmt"
)

// File storage errors
var (
	// ErrFileNotFound reports that a path does not exist. Storage backends
	// wrap their not-found errors with it so callers can tell "this file is
	// missing" (the user's problem) from "I could not check" (credentials,
	// network), which must not be treated the same way.
	ErrFileNotFound = errors.New("file not found")
)

// Batch logs errors
var (
	// ErrInvalidJobName is returned when the job name is not a valid full resource name.
	ErrInvalidJobName = errors.New("invalid job name: must be full resource name (projects/{project}/locations/{location}/jobs/{jobId})")

	// ErrLogsNotFound is returned when no logs are found for the job.
	ErrLogsNotFound = errors.New("no logs found for job")

	// ErrUnauthorized is returned when access to logs is denied.
	ErrUnauthorized = errors.New("unauthorized to access logs")
)

// BatchLogsError represents an error related to batch logs retrieval with context.
type BatchLogsError struct {
	JobName string
	Op      string // Operation: "fetch", "parse"
	Message string
	Cause   error
}

func (e BatchLogsError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s batch logs for '%s': %s (caused by: %v)", e.Op, e.JobName, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s batch logs for '%s': %s", e.Op, e.JobName, e.Message)
}

func (e BatchLogsError) Unwrap() error {
	return e.Cause
}

// NewBatchLogsError creates a new BatchLogsError with the given details.
func NewBatchLogsError(op, jobName, message string, cause error) *BatchLogsError {
	return &BatchLogsError{
		JobName: jobName,
		Op:      op,
		Message: message,
		Cause:   cause,
	}
}
