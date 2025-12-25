package ports

import "github.com/lmtani/pumbaa/internal/domain/wdlindex"

// WDLRepository defines the interface for WDL workflow indexing operations.
// This port allows querying and searching through indexed WDL tasks and workflows.
type WDLRepository interface {
	// List returns the complete index of tasks and workflows.
	List() (*wdlindex.Index, error)

	// SearchByName searches for tasks and workflows by name.
	SearchByName(query string) (*wdlindex.Index, error)

	// SearchByCommand searches for tasks by command content.
	SearchByCommand(query string) ([]*wdlindex.IndexedTask, error)

	// GetTask retrieves a specific task by name.
	GetTask(name string) (*wdlindex.IndexedTask, error)

	// GetWorkflow retrieves a specific workflow by name.
	GetWorkflow(name string) (*wdlindex.IndexedWorkflow, error)
}
