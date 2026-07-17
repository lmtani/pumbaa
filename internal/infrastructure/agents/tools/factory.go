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
	"github.com/lmtani/pumbaa/internal/infrastructure/agents/tools/submit"
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
	// requiresFetcher marks actions that need expanded-metadata access and
	// are skipped when no fetcher is wired (e.g. WDL-only registries).
	requiresFetcher bool
	// requiresFileProvider marks actions that read/verify local or cloud
	// files and are skipped when no file provider is wired.
	requiresFileProvider bool
	build                func(deps Deps) types.Handler
}

// Deps carries the external dependencies handlers can draw from. Any of the
// fields may be nil; actions that need a missing dependency are not
// registered.
type Deps struct {
	Repo         ports.WorkflowReader
	Fetcher      ports.WorkflowMetadataFetcher
	WDLRepo      wdl.Repository
	FileProvider ports.FileProvider
}

func builtinActions() []actionSpec {
	return []actionSpec{
		{
			name:        "query",
			description: "Search Cromwell workflows. Optional: status (Running, Succeeded, Failed, Submitted, Aborted), name.",
			build: func(deps Deps) types.Handler {
				return cromwell.NewQueryHandler(deps.Repo)
			},
		},
		{
			name:        "status",
			description: "Get workflow status. Required: workflow_id.",
			build: func(deps Deps) types.Handler {
				return cromwell.NewStatusHandler(deps.Repo)
			},
		},
		{
			name:        "metadata",
			description: "Get workflow metadata (calls, inputs, outputs). Required: workflow_id.",
			build: func(deps Deps) types.Handler {
				return cromwell.NewMetadataHandler(deps.Repo)
			},
		},
		{
			name:        "outputs",
			description: "Get workflow output files. Required: workflow_id.",
			build: func(deps Deps) types.Handler {
				return cromwell.NewOutputsHandler(deps.Repo)
			},
		},
		{
			name:        "logs",
			description: "Get log file paths for debugging. Required: workflow_id.",
			build: func(deps Deps) types.Handler {
				return cromwell.NewLogsHandler(deps.Repo)
			},
		},
		{
			name:        "scaffold",
			description: "Show a workflow's declared inputs and an inputs-JSON template to fill in. Answers \"what does this workflow need to run?\". Required: workflow_file (a .wdl path in the working directory). Optional: include_optional.",
			build: func(_ Deps) types.Handler {
				return submit.NewScaffoldHandler()
			},
		},
		{
			name:                 "preflight",
			description:          "Check an inputs JSON against a WDL before submitting: required inputs present, well-typed, and their file paths existing. Required: workflow_file. Optional: inputs_file (both .wdl/.json paths in the working directory).",
			requiresFileProvider: true,
			build: func(deps Deps) types.Handler {
				return submit.NewPreflightHandler(deps.FileProvider)
			},
		},
		{
			name:            "failures",
			description:     "Compact summary of what failed and why: root causes deduplicated across shards/subworkflows, with affected tasks and stderr paths. Prefer this over metadata to debug failures. Required: workflow_id.",
			requiresFetcher: true,
			build: func(deps Deps) types.Handler {
				return cromwell.NewFailuresHandler(deps.Fetcher)
			},
		},
		{
			name:        "read_log",
			description: "Read the tail of a task's log in one call. Either path (a stderr/stdout path from failures/logs), or workflow_id + task (optional: shard, stream=stderr|stdout, lines<=500).",
			build: func(deps Deps) types.Handler {
				return cromwell.NewReadLogHandler(deps.Repo)
			},
		},
		{
			name:            "cost",
			description:     "Per-task cost breakdown (subworkflows included): real dollars from VM rates, resource-hour estimates kept separate, most expensive first. Required: workflow_id.",
			requiresFetcher: true,
			build: func(deps Deps) types.Handler {
				return cromwell.NewCostHandler(deps.Fetcher)
			},
		},
		{
			name:            "preemption",
			description:     "Preemption efficiency: attempts, preemptions and the tasks losing the most work to preempted VMs. Required: workflow_id.",
			requiresFetcher: true,
			build: func(deps Deps) types.Handler {
				return cromwell.NewPreemptionHandler(deps.Fetcher)
			},
		},
		{
			name:        "gcs_download",
			description: "Read file from Google Cloud Storage. Required: path (gs://bucket/file).",
			build: func(deps Deps) types.Handler {
				return gcs.NewDownloadHandler()
			},
		},
		{
			name:        "write_file",
			description: "Write a text file (e.g. a bash script to reproduce/debug a task locally) into the user's current working directory. Required: path (relative), content. Optional: executable (true for scripts), overwrite (must be true to replace an existing file).",
			build: func(deps Deps) types.Handler {
				return localfs.NewWriteHandler()
			},
		},
		{
			name:        "wdl_list",
			description: "List all indexed WDL tasks and workflows.",
			requiresWDL: true,
			build: func(deps Deps) types.Handler {
				return wdl.NewListHandler(deps.WDLRepo)
			},
		},
		{
			name:        "wdl_search",
			description: "Search tasks/workflows by name or command content. Required: query.",
			requiresWDL: true,
			build: func(deps Deps) types.Handler {
				return wdl.NewSearchHandler(deps.WDLRepo)
			},
		},
		{
			name:        "wdl_info",
			description: `Get detailed info about a task or workflow. Required: name, type ("task" or "workflow").`,
			requiresWDL: true,
			build: func(deps Deps) types.Handler {
				return wdl.NewInfoHandler(deps.WDLRepo)
			},
		},
	}
}

// NewDefaultRegistry creates a Registry with all built-in handlers registered.
// wdlRepo can be nil if WDL indexing is not configured; WDL actions are then
// omitted. Callers can Register additional actions on the returned registry
// before passing it to GetPumbaaTool.
func NewDefaultRegistry(deps Deps) *Registry {
	r := NewRegistry()
	for _, spec := range builtinActions() {
		if spec.requiresWDL && deps.WDLRepo == nil {
			continue
		}
		if spec.requiresFetcher && deps.Fetcher == nil {
			continue
		}
		if spec.requiresFileProvider && deps.FileProvider == nil {
			continue
		}
		r.Register(spec.name, spec.description, spec.build(deps))
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
				r.Register(spec.name, spec.description, spec.build(Deps{WDLRepo: wdlRepo}))
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
func GetAllTools(deps Deps, extra ...tool.Tool) []tool.Tool {
	registry := NewDefaultRegistry(deps)
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
