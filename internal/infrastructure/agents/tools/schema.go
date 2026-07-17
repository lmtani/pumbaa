package tools

import "github.com/lmtani/pumbaa/internal/infrastructure/agents/tools/types"

// GetParametersSchema returns the JSON schema for the Pumbaa tool parameters.
// This is the single source of truth for the tool's parameter schema.
// Used by LLM providers that need explicit schema (e.g., Ollama).
func GetParametersSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"description": "The action to perform",
				"enum":        builtinActionNames(),
			},
			"workflow_id": map[string]any{
				"type":        "string",
				"description": "UUID of the workflow (required for status, metadata, outputs, logs actions)",
			},
			"status": map[string]any{
				"type":        "string",
				"description": "Status filter for query action",
				"enum":        []string{"Running", "Succeeded", "Failed", "Submitted", "Aborted"},
			},
			"name": map[string]any{
				"type":        "string",
				"description": "Name filter for query action or name for wdl_info",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "GCS path (gs://bucket/file) for gcs_download, or relative file path for write_file",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "File body for write_file action",
			},
			"executable": map[string]any{
				"type":        "boolean",
				"description": "Mark the written file as executable (write_file action)",
			},
			"overwrite": map[string]any{
				"type":        "boolean",
				"description": "Allow write_file to replace an existing file",
			},
			"query": map[string]any{
				"type":        "string",
				"description": "Search query for wdl_search action",
			},
			"type": map[string]any{
				"type":        "string",
				"description": "Type for wdl_info action",
				"enum":        []string{"task", "workflow"},
			},
			"page_size": map[string]any{
				"type":        "integer",
				"description": "Number of results to return for query action (default: 10)",
			},
			"task": map[string]any{
				"type":        "string",
				"description": "Task name for read_log (short name or full call name)",
			},
			"shard": map[string]any{
				"type":        "integer",
				"description": "Scatter shard for read_log (default -1 for non-scattered tasks)",
			},
			"stream": map[string]any{
				"type":        "string",
				"description": "Log stream for read_log",
				"enum":        []string{"stderr", "stdout"},
			},
			"lines": map[string]any{
				"type":        "integer",
				"description": "Tail lines for read_log (default 100, max 500)",
			},
			"workflow_file": map[string]any{
				"type":        "string",
				"description": "Local WDL path (working directory) for scaffold and preflight",
			},
			"inputs_file": map[string]any{
				"type":        "string",
				"description": "Local inputs JSON path (working directory) for preflight",
			},
			"include_optional": map[string]any{
				"type":        "boolean",
				"description": "Include optional inputs in the scaffold template",
			},
		},
		"required": []string{"action"},
	}
}

// builtinActionNames returns the names of all built-in actions for the
// schema enum, derived from the same table that drives registration.
func builtinActionNames() []string {
	specs := builtinActions()
	names := make([]string, 0, len(specs))
	for _, spec := range specs {
		names = append(names, spec.name)
	}
	return names
}

// Re-export types for backward compatibility with external packages
type (
	// Input is an alias for types.Input for backward compatibility.
	Input = types.Input
	// Output is an alias for types.Output for backward compatibility.
	Output = types.Output
)
