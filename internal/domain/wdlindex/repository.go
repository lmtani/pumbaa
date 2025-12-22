// Package wdlindex provides domain models for WDL workflow indexing.
package wdlindex

// Repository provides read access to indexed WDL content.
type Repository interface {
	// List returns all indexed tasks and workflows.
	List() (*Index, error)

	// SearchTasks finds tasks matching the query (case-insensitive, checks name and command).
	SearchTasks(query string) ([]*IndexedTask, error)

	// SearchWorkflows finds workflows matching the query (case-insensitive, checks name and calls).
	SearchWorkflows(query string) ([]*IndexedWorkflow, error)

	// GetTask returns a specific task by name.
	GetTask(name string) (*IndexedTask, error)

	// GetWorkflow returns a specific workflow by name.
	GetWorkflow(name string) (*IndexedWorkflow, error)
}
