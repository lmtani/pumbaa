// Package tools provides implementations of tools for use with Google Agents ADK.
package tools

import (
	"google.golang.org/adk/tool"
)

// GetAllTools returns all available tools in this package.
func GetAllTools() []tool.Tool {
	return []tool.Tool{
		GetGCSDownload(),
	}
}
