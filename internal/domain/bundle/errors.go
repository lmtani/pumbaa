// Package bundle contains domain errors for bundle operations.
package bundle

import (
	"errors"
	"fmt"
)

var (
	// ErrMainWorkflowNotFound is returned when the main workflow file is not found.
	ErrMainWorkflowNotFound = errors.New("main workflow file not found")

	// ErrInvalidWDL is returned when the WDL file is invalid.
	ErrInvalidWDL = errors.New("invalid WDL file")

	// ErrCircularDependency is returned when circular dependencies are detected.
	ErrCircularDependency = errors.New("circular dependency detected")

	// ErrDependencyNotFound is returned when a dependency file is not found.
	ErrDependencyNotFound = errors.New("dependency file not found")

	// ErrBundleCreationFailed is returned when bundle creation fails.
	ErrBundleCreationFailed = errors.New("bundle creation failed")
)

// DependencyError represents an error related to a specific dependency.
type DependencyError struct {
	Path    string
	Message string
	Cause   error
}

func (e DependencyError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("dependency error for '%s': %s (caused by: %v)", e.Path, e.Message, e.Cause)
	}
	return fmt.Sprintf("dependency error for '%s': %s", e.Path, e.Message)
}

func (e DependencyError) Unwrap() error {
	return e.Cause
}
