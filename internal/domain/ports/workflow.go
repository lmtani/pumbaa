// Package ports defines interfaces (ports) for external adapters.
// This follows the Hexagonal Architecture pattern where ports are defined in the domain
// and implemented by infrastructure adapters.
package ports

import (
	"context"

	"github.com/lmtani/pumbaa/internal/domain/workflow"
)

// WorkflowQuerier handles workflow listing and abort operations.
// Used by TUI dashboard for displaying and managing workflows.
type WorkflowQuerier interface {
	Query(ctx context.Context, filter workflow.QueryFilter) (*workflow.QueryResult, error)
	Abort(ctx context.Context, workflowID string) error
}

// WorkflowMetadataFetcher handles raw metadata and cost retrieval.
// Used by TUI debug view for loading workflow details and subworkflows.
type WorkflowMetadataFetcher interface {
	GetRawMetadataWithOptions(ctx context.Context, workflowID string, expandSubWorkflows bool) ([]byte, error)
	GetWorkflowCost(ctx context.Context, workflowID string) (float64, string, error)
}

// HealthChecker handles server health monitoring.
// Used by TUI dashboard to display server status.
type HealthChecker interface {
	GetHealthStatus(ctx context.Context) (*workflow.HealthStatus, error)
}

// LabelManager handles workflow label operations.
// Used by TUI dashboard for viewing and editing workflow labels.
type LabelManager interface {
	GetLabels(ctx context.Context, workflowID string) (map[string]string, error)
	UpdateLabels(ctx context.Context, workflowID string, labels map[string]string) error
}

// MetadataParser handles parsing of raw workflow metadata.
// Used by handlers to convert raw bytes into domain Workflow entities.
type MetadataParser interface {
	ParseMetadata(data []byte) (*workflow.Workflow, error)
}

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
