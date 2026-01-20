// Package types provides common types shared across tool handlers.
// This package exists to avoid import cycles between tools and its subpackages.
package types

// Input represents the unified input for the Pumbaa tool.
// It aggregates parameters for all available actions.
type Input struct {
	// Action is the operation to perform.
	// Cromwell: "query", "status", "metadata", "outputs", "logs"
	// GCS: "gcs_download"
	// WDL: "wdl_list", "wdl_search", "wdl_info"
	Action string `json:"action"`

	// WorkflowID is the UUID of the workflow (required for status, metadata, outputs, logs).
	WorkflowID string `json:"workflow_id,omitempty"`

	// Status filter for query action (e.g., "Running", "Succeeded", "Failed").
	Status string `json:"status,omitempty"`

	// Name filter for query action or name for wdl_info.
	Name string `json:"name,omitempty"`

	// Path for gcs_download action (gs://bucket/path).
	Path string `json:"path,omitempty"`

	// Query for wdl_search action.
	Query string `json:"query,omitempty"`

	// Type for wdl_info action: "task" or "workflow".
	Type string `json:"type,omitempty"`

	// PageSize for query action (default: 10).
	PageSize int `json:"page_size,omitempty"`
}

// Output represents the standardized output for all actions.
type Output struct {
	Success bool        `json:"success"`
	Error   string      `json:"error,omitempty"`
	Action  string      `json:"action"`
	Data    interface{} `json:"data,omitempty"`
}

// NewSuccessOutput creates a successful output for the given action.
func NewSuccessOutput(action string, data interface{}) Output {
	return Output{
		Success: true,
		Action:  action,
		Data:    data,
	}
}

// NewErrorOutput creates an error output for the given action.
func NewErrorOutput(action, errMsg string) Output {
	return Output{
		Success: false,
		Action:  action,
		Error:   errMsg,
	}
}
