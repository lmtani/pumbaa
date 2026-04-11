// Package ports defines interfaces (ports) for external adapters.
// This follows the Hexagonal Architecture pattern where ports are defined in the domain
// and implemented by infrastructure adapters.
package ports

import (
	"context"

	"github.com/lmtani/pumbaa/internal/domain/workflow"
)

// WorkflowQuerier handles workflow listing.
// Used by application query use case and TUI dashboard.
type WorkflowQuerier interface {
	Query(ctx context.Context, filter workflow.QueryFilter) (*workflow.QueryResult, error)
}

// WorkflowMetadataFetcher handles raw metadata retrieval, parsing, and cost estimation.
// Used by TUI debug and dashboard views for loading workflow details and subworkflows.
type WorkflowMetadataFetcher interface {
	GetRawMetadataWithOptions(ctx context.Context, workflowID string, expandSubWorkflows bool) ([]byte, error)
	GetWorkflowCost(ctx context.Context, workflowID string) (float64, string, error)
	ParseMetadata(data []byte) (*workflow.Workflow, error)
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

// WorkflowSubmitter handles workflow submission.
// Used by application submit use case.
type WorkflowSubmitter interface {
	Submit(ctx context.Context, req workflow.SubmitRequest) (*workflow.SubmitResponse, error)
}

// WorkflowAborter handles workflow abort operations and status checks.
// Used by application abort use case.
type WorkflowAborter interface {
	GetStatus(ctx context.Context, workflowID string) (workflow.Status, error)
	Abort(ctx context.Context, workflowID string) error
}

// WorkflowMetadataReader handles workflow metadata retrieval.
// Used by application metadata use case.
type WorkflowMetadataReader interface {
	GetMetadata(ctx context.Context, workflowID string) (*workflow.Workflow, error)
}

// WorkflowReader provides read-only workflow operations.
// Used by agent tools that need to query and inspect workflows without modifying them.
type WorkflowReader interface {
	Query(ctx context.Context, filter workflow.QueryFilter) (*workflow.QueryResult, error)
	GetStatus(ctx context.Context, workflowID string) (workflow.Status, error)
	GetMetadata(ctx context.Context, workflowID string) (*workflow.Workflow, error)
	GetOutputs(ctx context.Context, workflowID string) (map[string]interface{}, error)
	GetLogs(ctx context.Context, workflowID string) (map[string][]workflow.CallLog, error)
}

// WorkflowRepository defines the interface for workflow management operations.
// This is the primary port for all workflow-related operations including
// execution management, metadata retrieval, and server health monitoring.
//
// It composes smaller, focused interfaces following Interface Segregation Principle.
// Consumers should depend on the smallest interface that meets their needs.
type WorkflowRepository interface {
	// Composed interfaces
	WorkflowQuerier         // Query
	WorkflowAborter         // GetStatus, Abort
	WorkflowMetadataFetcher // GetRawMetadataWithOptions, GetWorkflowCost
	HealthChecker           // GetHealthStatus
	LabelManager            // GetLabels, UpdateLabels

	// Workflow submission
	Submit(ctx context.Context, req workflow.SubmitRequest) (*workflow.SubmitResponse, error)

	// Additional metadata operations
	GetMetadata(ctx context.Context, workflowID string) (*workflow.Workflow, error)
	GetOutputs(ctx context.Context, workflowID string) (map[string]interface{}, error)
	GetLogs(ctx context.Context, workflowID string) (map[string][]workflow.CallLog, error)
}
