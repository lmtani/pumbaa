// Package handler provides CLI command handlers.
package handler

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/lmtani/pumbaa/internal/application/workflow"
	"github.com/lmtani/pumbaa/internal/interfaces/cli/presenter"
)

// InputsHandler handles the workflow inputs command.
type InputsHandler struct {
	useCase   *workflow.InputsUseCase
	presenter *presenter.Presenter
}

// NewInputsHandler creates a new InputsHandler.
func NewInputsHandler(uc *workflow.InputsUseCase, p *presenter.Presenter) *InputsHandler {
	return &InputsHandler{
		useCase:   uc,
		presenter: p,
	}
}

// Command returns the CLI command for workflow inputs retrieval.
func (h *InputsHandler) Command() *cli.Command {
	return &cli.Command{
		Name:      "inputs",
		Aliases:   []string{"i", "in"},
		Usage:     "Get workflow inputs",
		ArgsUsage: "<workflow-id>",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "json",
				Aliases: []string{"j"},
				Usage:   "Output in JSON format",
			},
		},
		Action: h.handle,
	}
}

func (h *InputsHandler) handle(c *cli.Context) error {
	if c.NArg() < 1 {
		h.presenter.Error("Workflow ID is required")
		return cli.Exit("workflow ID required", 1)
	}

	ctx := context.Background()
	workflowID := c.Args().First()

	input := workflow.InputsInput{
		WorkflowID: workflowID,
	}

	output, err := h.useCase.Execute(ctx, input)
	if err != nil {
		h.presenter.Error("Failed to get workflow inputs: %v", err)
		return err
	}

	if c.Bool("json") {
		return h.displayJSON(output.Inputs)
	}

	h.displayHuman(output)
	return nil
}

func (h *InputsHandler) displayJSON(inputs map[string]interface{}) error {
	data, err := json.MarshalIndent(inputs, "", "  ")
	if err != nil {
		h.presenter.Error("Failed to marshal inputs: %v", err)
		return err
	}
	h.presenter.Println(string(data))
	return nil
}

func (h *InputsHandler) displayHuman(output *workflow.InputsOutput) {
	h.presenter.Title("Workflow Inputs")
	h.presenter.KeyValue("Workflow ID", output.WorkflowID)
	h.presenter.KeyValue("Workflow Name", output.WorkflowName)
	h.presenter.Newline()

	if len(output.Inputs) == 0 {
		h.presenter.Info("No inputs available")
		return
	}

	// Sort keys for consistent output
	keys := make([]string, 0, len(output.Inputs))
	for k := range output.Inputs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := output.Inputs[k]
		// Remove workflow name prefix for cleaner display
		displayKey := stripInputWorkflowPrefix(k, output.WorkflowName)
		h.presenter.KeyValue(displayKey, formatInputValue(v))
	}
}

// formatInputValue formats a value for display, handling arrays and nested structures.
func formatInputValue(v interface{}) interface{} {
	switch val := v.(type) {
	case []interface{}:
		if len(val) == 0 {
			return "[]"
		}
		if len(val) == 1 {
			return formatInputValue(val[0])
		}
		// For arrays, format each element on a new line
		return v
	default:
		return v
	}
}

// stripInputWorkflowPrefix removes the workflow name prefix from a key.
// e.g., "MyWorkflow.input_file" becomes "input_file"
func stripInputWorkflowPrefix(key, workflowName string) string {
	prefix := workflowName + "."
	if strings.HasPrefix(key, prefix) {
		return strings.TrimPrefix(key, prefix)
	}
	return key
}
