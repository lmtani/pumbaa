package wdl

import (
	"context"
	"fmt"
	"strings"

	"github.com/lmtani/pumbaa/internal/domain/wdlindex"
	"github.com/lmtani/pumbaa/internal/infrastructure/chat/agent/tools/types"
)

// InfoHandler handles the "wdl_info" action to get detailed info about a task or workflow.
type InfoHandler struct {
	repo Repository
}

// NewInfoHandler creates a new InfoHandler.
func NewInfoHandler(repo Repository) *InfoHandler {
	return &InfoHandler{repo: repo}
}

// Handle implements types.Handler.
func (h *InfoHandler) Handle(_ context.Context, input types.Input) (types.Output, error) {
	const action = "wdl_info"

	if h.repo == nil {
		return types.NewErrorOutput(action, notConfiguredError), nil
	}

	if input.Name == "" {
		return types.NewErrorOutput(action, "name is required"), nil
	}

	switch strings.ToLower(input.Type) {
	case "task", "":
		task, err := h.repo.GetTask(input.Name)
		if err != nil {
			// Try as workflow if task not found and type wasn't specified
			if input.Type == "" {
				wf, wfErr := h.repo.GetWorkflow(input.Name)
				if wfErr == nil {
					return buildWorkflowInfoOutput(wf), nil
				}
			}
			return types.NewErrorOutput(action, err.Error()), nil
		}
		return buildTaskInfoOutput(task), nil

	case "workflow":
		wf, err := h.repo.GetWorkflow(input.Name)
		if err != nil {
			return types.NewErrorOutput(action, err.Error()), nil
		}
		return buildWorkflowInfoOutput(wf), nil

	default:
		return types.NewErrorOutput(action, fmt.Sprintf(
			"invalid type: %s. Valid: task, workflow", input.Type,
		)), nil
	}
}

func buildTaskInfoOutput(task *wdlindex.IndexedTask) types.Output {
	inputs := make([]map[string]interface{}, 0, len(task.Inputs))
	for _, in := range task.Inputs {
		inputs = append(inputs, map[string]interface{}{
			"name":     in.Name,
			"type":     in.Type,
			"optional": in.Optional,
		})
	}

	outputs := make([]map[string]interface{}, 0, len(task.Outputs))
	for _, out := range task.Outputs {
		outputs = append(outputs, map[string]interface{}{
			"name": out.Name,
			"type": out.Type,
		})
	}

	return types.NewSuccessOutput("wdl_info", map[string]interface{}{
		"type":        "task",
		"name":        task.Name,
		"source":      task.Source,
		"description": task.Description,
		"inputs":      inputs,
		"outputs":     outputs,
		"command":     task.Command,
		"runtime":     task.Runtime,
	})
}

func buildWorkflowInfoOutput(wf *wdlindex.IndexedWorkflow) types.Output {
	inputs := make([]map[string]interface{}, 0, len(wf.Inputs))
	for _, in := range wf.Inputs {
		inputs = append(inputs, map[string]interface{}{
			"name":     in.Name,
			"type":     in.Type,
			"optional": in.Optional,
		})
	}

	outputs := make([]map[string]interface{}, 0, len(wf.Outputs))
	for _, out := range wf.Outputs {
		outputs = append(outputs, map[string]interface{}{
			"name": out.Name,
			"type": out.Type,
		})
	}

	return types.NewSuccessOutput("wdl_info", map[string]interface{}{
		"type":        "workflow",
		"name":        wf.Name,
		"source":      wf.Source,
		"description": wf.Description,
		"inputs":      inputs,
		"outputs":     outputs,
		"calls":       wf.Calls,
	})
}
