// Package wdl provides action handlers for WDL knowledge base operations.
package wdl

import (
	"github.com/lmtani/pumbaa/internal/domain/wdlindex"
)

const notConfiguredError = "WDL index not configured. Set PUMBAA_WDL_DIR environment variable."

// Repository defines the interface for WDL index operations.
type Repository interface {
	List() (*wdlindex.Index, error)
	SearchTasks(query string) ([]*wdlindex.IndexedTask, error)
	SearchWorkflows(query string) ([]*wdlindex.IndexedWorkflow, error)
	GetTask(name string) (*wdlindex.IndexedTask, error)
	GetWorkflow(name string) (*wdlindex.IndexedWorkflow, error)
}
