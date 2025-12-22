package workflow

import "context"

// HealthStatus represents the health status of the Cromwell server.
type HealthStatus struct {
	OK               bool     // All subsystems are healthy
	Degraded         bool     // Some subsystems are unhealthy
	UnhealthySystems []string // List of unhealthy subsystem names
}

// HealthChecker provides health status checking for the workflow server.
type HealthChecker interface {
	GetHealthStatus(ctx context.Context) (*HealthStatus, error)
}

// LabelManager provides label management for workflows.
type LabelManager interface {
	GetLabels(ctx context.Context, workflowID string) (map[string]string, error)
	UpdateLabels(ctx context.Context, workflowID string, labels map[string]string) error
}
