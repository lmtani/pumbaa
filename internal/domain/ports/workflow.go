// Package ports defines interfaces (ports) for external adapters.
// This follows the Hexagonal Architecture pattern where ports are defined in the domain
// and implemented by infrastructure adapters.
package ports

import (
	"context"

	"github.com/lmtani/pumbaa/internal/domain/workflow"
)

// WorkflowRepository defines the interface for workflow management operations.
// This is the primary port for all workflow-related operations including
// execution management, metadata retrieval, and server health monitoring.
type WorkflowRepository interface {
	// Workflow execution operations
	Submit(ctx context.Context, req workflow.SubmitRequest) (*workflow.SubmitResponse, error)
	Abort(ctx context.Context, workflowID string) error
	Query(ctx context.Context, filter workflow.QueryFilter) (*workflow.QueryResult, error)

	// Metadata operations
	GetMetadata(ctx context.Context, workflowID string) (*workflow.Workflow, error)
	GetRawMetadataWithOptions(ctx context.Context, workflowID string, expandSubWorkflows bool) ([]byte, error)
	GetStatus(ctx context.Context, workflowID string) (workflow.Status, error)
	GetOutputs(ctx context.Context, workflowID string) (map[string]interface{}, error)
	GetLogs(ctx context.Context, workflowID string) (map[string][]workflow.CallLog, error)

	// Cost analysis
	GetWorkflowCost(ctx context.Context, workflowID string) (float64, string, error)

	// Server health monitoring
	GetHealthStatus(ctx context.Context) (*workflow.HealthStatus, error)

	// Workflow labels management
	GetLabels(ctx context.Context, workflowID string) (map[string]string, error)
	UpdateLabels(ctx context.Context, workflowID string, labels map[string]string) error
}
