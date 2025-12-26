// Package application contains error definitions for application layer use cases.
package application

import (
	"errors"
	"fmt"
)

// Use case errors
var (
	// ErrInvalidInput is returned when use case input validation fails.
	ErrInvalidInput = errors.New("invalid input")

	// ErrOperationFailed is returned when a use case operation fails.
	ErrOperationFailed = errors.New("operation failed")

	// ErrNotAuthorized is returned when an operation is not authorized.
	ErrNotAuthorized = errors.New("not authorized")
)

// UseCaseError wraps domain/infrastructure errors with operation context.
// This allows handlers to understand which use case failed and why.
type UseCaseError struct {
	Operation string // Use case name: "submit", "abort", "query", etc.
	Message   string // Human-readable description
	Cause     error  // Underlying error (domain or infrastructure)
}

func (e UseCaseError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Operation, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Operation, e.Message)
}

func (e UseCaseError) Unwrap() error {
	return e.Cause
}

// NewUseCaseError creates a new UseCaseError with the given details.
func NewUseCaseError(operation, message string, cause error) *UseCaseError {
	return &UseCaseError{
		Operation: operation,
		Message:   message,
		Cause:     cause,
	}
}

// InputValidationError represents a validation error for use case inputs.
type InputValidationError struct {
	Field   string
	Message string
}

func (e InputValidationError) Error() string {
	return fmt.Sprintf("invalid input for '%s': %s", e.Field, e.Message)
}

// Is allows InputValidationError to match ErrInvalidInput with errors.Is().
func (e InputValidationError) Is(target error) bool {
	return errors.Is(target, ErrInvalidInput)
}

// NewInputValidationError creates a new InputValidationError.
func NewInputValidationError(field, message string) *InputValidationError {
	return &InputValidationError{
		Field:   field,
		Message: message,
	}
}
