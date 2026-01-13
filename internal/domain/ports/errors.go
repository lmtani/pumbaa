// Package ports contains error definitions for port interfaces.
package ports

import (
	"errors"
	"fmt"
)

// File storage errors
var (
	// ErrFileNotFound is returned when a file cannot be found at the specified path.
	ErrFileNotFound = errors.New("file not found")

	// ErrFileTooLarge is returned when a file exceeds the allowed size limit.
	ErrFileTooLarge = errors.New("file exceeds size limit")

	// ErrInvalidPath is returned when the file path is malformed or invalid.
	ErrInvalidPath = errors.New("invalid file path")

	// ErrAccessDenied is returned when access to a file is denied.
	ErrAccessDenied = errors.New("access denied")
)

// FileError represents an error related to file operations with context.
type FileError struct {
	Path    string
	Op      string // Operation: "read", "write", "stat"
	Message string
	Cause   error
}

func (e FileError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s '%s': %s (caused by: %v)", e.Op, e.Path, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s '%s': %s", e.Op, e.Path, e.Message)
}

func (e FileError) Unwrap() error {
	return e.Cause
}

// NewFileError creates a new FileError with the given details.
func NewFileError(op, path, message string, cause error) *FileError {
	return &FileError{
		Path:    path,
		Op:      op,
		Message: message,
		Cause:   cause,
	}
}

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
