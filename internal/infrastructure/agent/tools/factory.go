package tools

import (
	"context"
	"fmt"

	"github.com/lmtani/pumbaa/internal/infrastructure/agent/tools/cromwell"
	"github.com/lmtani/pumbaa/internal/infrastructure/agent/tools/gcs"
	"github.com/lmtani/pumbaa/internal/infrastructure/agent/tools/types"
	"github.com/lmtani/pumbaa/internal/infrastructure/agent/tools/wdl"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// NewDefaultRegistry creates a Registry with all default handlers registered.
// wdlRepo can be nil if WDL indexing is not configured.
func NewDefaultRegistry(repo cromwell.Repository, wdlRepo wdl.Repository) *Registry {
	r := NewRegistry()

	// Cromwell actions
	r.Register("query", cromwell.NewQueryHandler(repo))
	r.Register("status", cromwell.NewStatusHandler(repo))
	r.Register("metadata", cromwell.NewMetadataHandler(repo))
	r.Register("outputs", cromwell.NewOutputsHandler(repo))
	r.Register("logs", cromwell.NewLogsHandler(repo))

	// GCS actions
	r.Register("gcs_download", gcs.NewDownloadHandler())

	// WDL actions (only if configured)
	if wdlRepo != nil {
		r.Register("wdl_list", wdl.NewListHandler(wdlRepo))
		r.Register("wdl_search", wdl.NewSearchHandler(wdlRepo))
		r.Register("wdl_info", wdl.NewInfoHandler(wdlRepo))
	}

	return r
}

// GetPumbaaTool creates the unified Pumbaa tool using the provided registry.
// This function creates an ADK-compatible tool that routes to the appropriate handler.
func GetPumbaaTool(registry *Registry) tool.Tool {
	description := buildDescription(registry)

	t, err := functiontool.New(
		functiontool.Config{
			Name:        "pumbaa",
			Description: description,
		},
		func(ctx tool.Context, input types.Input) (types.Output, error) {
			bgCtx := context.Background()
			return registry.Handle(bgCtx, input)
		},
	)
	if err != nil {
		panic(fmt.Sprintf("failed to create pumbaa tool: %v", err))
	}
	return t
}

// GetAllTools returns all available tools in this package.
// cromwellRepo is the Cromwell repository implementation for API interactions.
// wdlRepo is the WDL index repository (can be nil if not configured).
func GetAllTools(cromwellRepo cromwell.Repository, wdlRepo wdl.Repository) []tool.Tool {
	registry := NewDefaultRegistry(cromwellRepo, wdlRepo)
	return []tool.Tool{
		GetPumbaaTool(registry),
	}
}

// buildDescription generates the tool description based on registered actions.
func buildDescription(registry *Registry) string {
	baseDescription := `Unified tool for Cromwell workflow management and GCS file access.

Available actions:
- "query": Search Cromwell workflows. Optional: status (Running, Succeeded, Failed, Submitted, Aborted), name.
- "status": Get workflow status. Required: workflow_id.
- "metadata": Get workflow metadata (calls, inputs, outputs). Required: workflow_id.
- "outputs": Get workflow output files. Required: workflow_id.
- "logs": Get log file paths for debugging. Required: workflow_id.
- "gcs_download": Read file from Google Cloud Storage. Required: path (gs://bucket/file).`

	// Check if WDL actions are registered
	if _, ok := registry.Get("wdl_list"); ok {
		baseDescription += `
- "wdl_list": List all indexed WDL tasks and workflows.
- "wdl_search": Search tasks/workflows by name or command content. Required: query.
- "wdl_info": Get detailed info about a task or workflow. Required: name, type ("task" or "workflow").`
	}

	return baseDescription
}
