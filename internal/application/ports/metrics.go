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

// TaskMetricsWriter abstracts writing task metrics to external sinks.
// Implementations may write TSV files, databases, or other formats.
type TaskMetricsWriter interface {
	// WriteToFile writes task metrics to a file in the implementation's format.
	WriteToFile(filename string, metrics []workflow.TaskMetrics) error
}
