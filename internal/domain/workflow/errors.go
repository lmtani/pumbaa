// Package workflow contains domain errors.
package workflow

import (
	"errors"
	"fmt"
)

var (
	// ErrWorkflowNotFound is returned when a workflow is not found.
	ErrWorkflowNotFound = errors.New("workflow not found")

	// ErrInvalidWorkflowID is returned when a workflow ID is invalid.
	ErrInvalidWorkflowID = errors.New("invalid workflow ID")

	// ErrWorkflowAlreadyTerminal is returned when trying to abort a terminal workflow.
	ErrWorkflowAlreadyTerminal = errors.New("workflow is already in terminal state")

	// ErrSubmissionFailed is returned when workflow submission fails.
	ErrSubmissionFailed = errors.New("workflow submission failed")

	// ErrConnectionFailed is returned when connection to Cromwell fails.
	ErrConnectionFailed = errors.New("connection to Cromwell server failed")
)

// ValidationError represents a validation error with field information.
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation error on field '%s': %s", e.Field, e.Message)
}

// APIError represents an error from the Cromwell API.
type APIError struct {
	StatusCode int
	Message    string
}

func (e APIError) Error() string {
	return fmt.Sprintf("API error (status %d): %s", e.StatusCode, e.Message)
}
