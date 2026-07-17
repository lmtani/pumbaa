package submit

import (
	"context"

	"github.com/lmtani/pumbaa/internal/infrastructure/agents/tools/types"
	"github.com/lmtani/pumbaa/pkg/wdl"
)

// ScaffoldHandler handles the "scaffold" action: it reports a workflow's
// declared inputs and an inputs-JSON template to fill in, so a user can see
// what a workflow needs before writing anything.
type ScaffoldHandler struct{}

// NewScaffoldHandler creates a new ScaffoldHandler.
func NewScaffoldHandler() *ScaffoldHandler {
	return &ScaffoldHandler{}
}

// Handle implements types.Handler.
func (h *ScaffoldHandler) Handle(_ context.Context, input types.Input) (types.Output, error) {
	const action = "scaffold"
	if input.WorkflowFile == "" {
		return types.NewErrorOutput(action, "workflow_file is required (a .wdl path in the working directory)"), nil
	}

	source, err := readWorkingDirFile(input.WorkflowFile)
	if err != nil {
		return types.NewErrorOutput(action, err.Error()), nil
	}

	scaffold, err := wdl.ScaffoldInputs(source, wdl.ScaffoldOptions{IncludeOptional: input.IncludeOptional})
	if err != nil {
		return types.NewErrorOutput(action, err.Error()), nil
	}

	inputs := make([]map[string]any, 0, len(scaffold.Inputs))
	for _, in := range scaffold.Inputs {
		entry := map[string]any{
			"name":     in.Name,
			"type":     in.Type,
			"required": in.Required(),
		}
		if in.Default != "" {
			entry["default"] = in.Default
		}
		if in.Description != "" {
			entry["description"] = in.Description
		}
		inputs = append(inputs, entry)
	}

	return types.NewSuccessOutput(action, map[string]any{
		"workflow": scaffold.WorkflowName,
		"inputs":   inputs,
		"template": string(scaffold.Template),
		"hint":     "fill in the placeholders, then use write_file to save the template and action=preflight to check it",
	}), nil
}
