package tools

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"

	"github.com/lmtani/pumbaa/internal/application/ports"
	"github.com/lmtani/pumbaa/internal/infrastructure/agents/tools/cromwell"
	"github.com/lmtani/pumbaa/internal/infrastructure/agents/tools/gcs"
	"github.com/lmtani/pumbaa/internal/infrastructure/agents/tools/localfs"
	"github.com/lmtani/pumbaa/internal/infrastructure/agents/tools/types"
	"github.com/lmtani/pumbaa/internal/infrastructure/agents/tools/wdl"
)

// actionSpec declares a built-in action of the pumbaa tool: its name, the
// description surfaced to the LLM, and how to build its handler.
//
// This table is the single source of truth for the built-in action set:
// registration (NewDefaultRegistry), the generated tool description and the
// parameters-schema enum (schema.go) all derive from it. Adding an action is
// one entry here — plus a property in GetParametersSchema if the action
// introduces new parameters.
type actionSpec struct {
	name        string
	description string
	requiresWDL bool
	build       func(repo ports.WorkflowReader, wdlRepo wdl.Repository) types.Handler
}

func builtinActions() []actionSpec {
	return []actionSpec{
		{
			name:        "query",
			description: "Search Cromwell workflows. Optional: status (Running, Succeeded, Failed, Submitted, Aborted), name.",
			build: func(repo ports.WorkflowReader, _ wdl.Repository) types.Handler {
				return cromwell.NewQueryHandler(repo)
			},
		},
		{
			name:        "status",
			description: "Get workflow status. Required: workflow_id.",
			build: func(repo ports.WorkflowReader, _ wdl.Repository) types.Handler {
				return cromwell.NewStatusHandler(repo)
			},
		},
		{
			name:        "metadata",
			description: "Get workflow metadata (calls, inputs, outputs). Required: workflow_id.",
			build: func(repo ports.WorkflowReader, _ wdl.Repository) types.Handler {
				return cromwell.NewMetadataHandler(repo)
			},
		},
		{
			name:        "outputs",
			description: "Get workflow output files. Required: workflow_id.",
			build: func(repo ports.WorkflowReader, _ wdl.Repository) types.Handler {
				return cromwell.NewOutputsHandler(repo)
			},
		},
		{
			name:        "logs",
			description: "Get log file paths for debugging. Required: workflow_id.",
			build: func(repo ports.WorkflowReader, _ wdl.Repository) types.Handler {
				return cromwell.NewLogsHandler(repo)
			},
		},
		{
			name:        "gcs_download",
			description: "Read file from Google Cloud Storage. Required: path (gs://bucket/file).",
			build: func(_ ports.WorkflowReader, _ wdl.Repository) types.Handler {
				return gcs.NewDownloadHandler()
			},
		},
		{
			name:        "write_file",
			description: "Write a text file (e.g. a bash script to reproduce/debug a task locally) into the user's current working directory. Required: path (relative), content. Optional: executable (true for scripts), overwrite (must be true to replace an existing file).",
			build: func(_ ports.WorkflowReader, _ wdl.Repository) types.Handler {
				return localfs.NewWriteHandler()
			},
		},
		{
			name:        "wdl_list",
			description: "List all indexed WDL tasks and workflows.",
			requiresWDL: true,
			build: func(_ ports.WorkflowReader, wdlRepo wdl.Repository) types.Handler {
				return wdl.NewListHandler(wdlRepo)
			},
		},
		{
			name:        "wdl_search",
			description: "Search tasks/workflows by name or command content. Required: query.",
			requiresWDL: true,
			build: func(_ ports.WorkflowReader, wdlRepo wdl.Repository) types.Handler {
				return wdl.NewSearchHandler(wdlRepo)
			},
		},
		{
			name:        "wdl_info",
			description: `Get detailed info about a task or workflow. Required: name, type ("task" or "workflow").`,
			requiresWDL: true,
			build: func(_ ports.WorkflowReader, wdlRepo wdl.Repository) types.Handler {
				return wdl.NewInfoHandler(wdlRepo)
			},
		},
	}
}

// NewDefaultRegistry creates a Registry with all built-in handlers registered.
// wdlRepo can be nil if WDL indexing is not configured; WDL actions are then
// omitted. Callers can Register additional actions on the returned registry
// before passing it to GetPumbaaTool.
func NewDefaultRegistry(repo ports.WorkflowReader, wdlRepo wdl.Repository) *Registry {
	r := NewRegistry()
	for _, spec := range builtinActions() {
		if spec.requiresWDL && wdlRepo == nil {
			continue
		}
		r.Register(spec.name, spec.description, spec.build(repo, wdlRepo))
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

// GetWDLOnlyTools returns only WDL tools for use cases like recommendation.
// This allows features to receive pre-configured tools without knowing how to create them.
func GetWDLOnlyTools(wdlRepo wdl.Repository) []tool.Tool {
	r := NewRegistry()
	if wdlRepo != nil {
		for _, spec := range builtinActions() {
			if spec.requiresWDL {
				r.Register(spec.name, spec.description, spec.build(nil, wdlRepo))
			}
		}
	}
	return []tool.Tool{GetPumbaaTool(r)}
}

// GetAllTools returns the tools available to the chat agent: the unified
// pumbaa tool plus any extra ADK tools.
//
// repo is the workflow repository for API interactions; wdlRepo is the WDL
// index repository (nil if not configured). extra is the extension point for
// adding standalone tools to the agent — each must be an ADK tool exposing
// Declaration()/Run (e.g. built with functiontool.New).
func GetAllTools(repo ports.WorkflowReader, wdlRepo wdl.Repository, extra ...tool.Tool) []tool.Tool {
	registry := NewDefaultRegistry(repo, wdlRepo)
	return append([]tool.Tool{GetPumbaaTool(registry)}, extra...)
}

// buildDescription generates the tool description from the registered
// actions, so registration is the only step needed to document an action.
func buildDescription(registry *Registry) string {
	var sb strings.Builder
	sb.WriteString("Unified tool for Cromwell workflow management and GCS file access.\n\nAvailable actions:")
	for _, doc := range registry.Docs() {
		fmt.Fprintf(&sb, "\n- %q: %s", doc.Action, doc.Description)
	}
	return sb.String()
}
