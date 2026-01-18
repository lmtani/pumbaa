// Package ports defines the interfaces for external dependencies (repositories, services).
// This file defines the interface for reading task metrics from external sources.
package ports

import (
	"github.com/lmtani/pumbaa/internal/domain/workflow"
)

// TaskMetricsReader abstracts the reading of task metrics from external sources.
// Implementations may read from TSV files, databases, or other formats.
type TaskMetricsReader interface {
	// ReadFromDirectory reads task metrics from all files in a directory.
	// Returns the collection of metrics, a list of workflow IDs found, and any error.
	ReadFromDirectory(dir string) (*workflow.TaskMetricsCollection, []string, error)
}
