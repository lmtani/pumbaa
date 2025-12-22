// Package tools provides implementations of tools for use with Google Agents ADK.
package tools

import (
	"google.golang.org/adk/tool"
)

// GetAllTools returns all available tools in this package.
// cromwellRepo is the Cromwell repository implementation for API interactions.
// wdlRepo is the WDL index repository (can be nil if not configured).
func GetAllTools(cromwellRepo CromwellRepository, wdlRepo WDLRepository) []tool.Tool {
	// Return a single unified tool to avoid Vertex AI limitation
	// "Multiple tools are supported only when they are all search tools"
	return []tool.Tool{
		GetPumbaaTool(cromwellRepo, wdlRepo),
	}
}
