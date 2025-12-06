// Package workflow contains domain repository interfaces.
package workflow

import (
	"context"
)

// Repository defines the interface for workflow persistence operations.
// This is a port in the hexagonal architecture sense.
type Repository interface {
	// Submit submits a new workflow for execution.
	Submit(ctx context.Context, req SubmitRequest) (*SubmitResponse, error)

	// GetMetadata retrieves detailed metadata for a specific workflow.
	GetMetadata(ctx context.Context, workflowID string) (*Workflow, error)

	// GetStatus retrieves only the status of a specific workflow.
	GetStatus(ctx context.Context, workflowID string) (Status, error)

	// Abort aborts a running workflow.
	Abort(ctx context.Context, workflowID string) error

	// Query queries workflows based on filters.
	Query(ctx context.Context, filter QueryFilter) (*QueryResult, error)

	// GetOutputs retrieves the outputs of a completed workflow.
	GetOutputs(ctx context.Context, workflowID string) (map[string]interface{}, error)

	// GetLogs retrieves the logs for a specific workflow.
	GetLogs(ctx context.Context, workflowID string) (map[string][]CallLog, error)
}

// CallLog represents log information for a call.
type CallLog struct {
	Stdout     string
	Stderr     string
	Attempt    int
	ShardIndex int
}
