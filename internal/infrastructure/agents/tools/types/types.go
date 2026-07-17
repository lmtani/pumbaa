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

	// Content is the file body for the write_file action.
	Content string `json:"content,omitempty"`

	// Executable marks the written file as executable (write_file action).
	Executable bool `json:"executable,omitempty"`

	// Overwrite allows write_file to replace an existing file.
	Overwrite bool `json:"overwrite,omitempty"`

	// Task is the task name for the read_log action (short or full call name).
	Task string `json:"task,omitempty"`

	// Shard selects a scatter shard for read_log (default: -1, non-scattered).
	Shard *int `json:"shard,omitempty"`

	// Stream selects the log stream for read_log: "stderr" (default) or "stdout".
	Stream string `json:"stream,omitempty"`

	// Lines is how many tail lines read_log returns (default 100, max 500).
	Lines int `json:"lines,omitempty"`
}

// Output represents the standardized output for all actions.
type Output struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
	Action  string `json:"action"`
	Data    any    `json:"data,omitempty"`
}

// NewSuccessOutput creates a successful output for the given action.
func NewSuccessOutput(action string, data any) Output {
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
