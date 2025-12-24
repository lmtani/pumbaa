package tools

import "github.com/lmtani/pumbaa/internal/infrastructure/agent/tools/types"

// GetParametersSchema returns the JSON schema for the Pumbaa tool parameters.
// This is the single source of truth for the tool's parameter schema.
// Used by LLM providers that need explicit schema (e.g., Ollama).
func GetParametersSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"description": "The action to perform",
				"enum":        []string{"query", "status", "metadata", "outputs", "logs", "gcs_download", "wdl_list", "wdl_search", "wdl_info"},
			},
			"workflow_id": map[string]interface{}{
				"type":        "string",
				"description": "UUID of the workflow (required for status, metadata, outputs, logs actions)",
			},
			"status": map[string]interface{}{
				"type":        "string",
				"description": "Status filter for query action",
				"enum":        []string{"Running", "Succeeded", "Failed", "Submitted", "Aborted"},
			},
			"name": map[string]interface{}{
				"type":        "string",
				"description": "Name filter for query action or name for wdl_info",
			},
			"path": map[string]interface{}{
				"type":        "string",
				"description": "GCS path (gs://bucket/file) for gcs_download action",
			},
			"query": map[string]interface{}{
				"type":        "string",
				"description": "Search query for wdl_search action",
			},
			"type": map[string]interface{}{
				"type":        "string",
				"description": "Type for wdl_info action",
				"enum":        []string{"task", "workflow"},
			},
			"page_size": map[string]interface{}{
				"type":        "integer",
				"description": "Number of results to return for query action (default: 10)",
			},
		},
		"required": []string{"action"},
	}
}

// Re-export types for backward compatibility with external packages
type (
	// Input is an alias for types.Input for backward compatibility.
	Input = types.Input
	// Output is an alias for types.Output for backward compatibility.
	Output = types.Output
)
