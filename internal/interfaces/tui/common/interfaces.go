// Package common provides shared components for TUI screens.
package common

import (
	"context"

	"github.com/lmtani/pumbaa/internal/domain/workflow"
)

// MetadataFetcher provides workflow metadata fetching capabilities.
// Used by both dashboard (for debug transitions) and debug (for subworkflows).
type MetadataFetcher interface {
	GetRawMetadataWithOptions(ctx context.Context, workflowID string, expandSubWorkflows bool) ([]byte, error)
	GetWorkflowCost(ctx context.Context, workflowID string) (float64, string, error)
}

// WorkflowFetcher provides workflow querying and management capabilities.
// Used by dashboard for listing and aborting workflows.
type WorkflowFetcher interface {
	Query(ctx context.Context, filter workflow.QueryFilter) (*workflow.QueryResult, error)
	Abort(ctx context.Context, workflowID string) error
}

// HealthChecker provides health status checking for the workflow server.
type HealthChecker interface {
	GetHealthStatus(ctx context.Context) (*workflow.HealthStatus, error)
}

// LabelManager provides label management for workflows.
type LabelManager interface {
	GetLabels(ctx context.Context, workflowID string) (map[string]string, error)
	UpdateLabels(ctx context.Context, workflowID string, labels map[string]string) error
}
