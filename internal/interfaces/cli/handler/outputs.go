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

// OutputsHandler handles the workflow outputs command.
type OutputsHandler struct {
	useCase   *workflow.OutputsUseCase
	presenter *presenter.Presenter
}

// NewOutputsHandler creates a new OutputsHandler.
func NewOutputsHandler(uc *workflow.OutputsUseCase, p *presenter.Presenter) *OutputsHandler {
	return &OutputsHandler{
		useCase:   uc,
		presenter: p,
	}
}

// Command returns the CLI command for workflow outputs retrieval.
func (h *OutputsHandler) Command() *cli.Command {
	return &cli.Command{
		Name:      "outputs",
		Aliases:   []string{"o", "out"},
		Usage:     "Get workflow outputs",
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

func (h *OutputsHandler) handle(c *cli.Context) error {
	if c.NArg() < 1 {
		h.presenter.Error("Workflow ID is required")
		return cli.Exit("workflow ID required", 1)
	}

	ctx := context.Background()
	workflowID := c.Args().First()

	input := workflow.OutputsInput{
		WorkflowID: workflowID,
	}

	output, err := h.useCase.Execute(ctx, input)
	if err != nil {
		h.presenter.Error("Failed to get workflow outputs: %v", err)
		return err
	}

	if c.Bool("json") {
		return h.displayJSON(output.Outputs)
	}

	h.displayHuman(output)
	return nil
}

func (h *OutputsHandler) displayJSON(outputs map[string]any) error {
	data, err := json.MarshalIndent(outputs, "", "  ")
	if err != nil {
		h.presenter.Error("Failed to marshal outputs: %v", err)
		return err
	}
	h.presenter.Println(string(data))
	return nil
}

func (h *OutputsHandler) displayHuman(output *workflow.OutputsOutput) {
	h.presenter.Title("Workflow Outputs")
	h.presenter.KeyValue("Workflow ID", output.WorkflowID)
	h.presenter.KeyValue("Workflow Name", output.WorkflowName)
	h.presenter.Newline()

	if len(output.Outputs) == 0 {
		h.presenter.Info("No outputs available")
		return
	}

	// Sort keys for consistent output
	keys := make([]string, 0, len(output.Outputs))
	for k := range output.Outputs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := output.Outputs[k]
		// Remove workflow name prefix for cleaner display
		displayKey := stripWorkflowPrefix(k, output.WorkflowName)
		h.presenter.KeyValue(displayKey, formatValue(v))
	}
}

// formatValue formats a value for display, handling arrays and nested structures.
func formatValue(v any) any {
	switch val := v.(type) {
	case []any:
		if len(val) == 0 {
			return "[]"
		}
		if len(val) == 1 {
			return formatValue(val[0])
		}
		// For arrays, format each element on a new line
		return v
	case map[string]any:
		// For objects, just return as-is (will be displayed as JSON-like)
		return v
	default:
		return v
	}
}

// stripWorkflowPrefix removes the workflow name prefix from a key.
// e.g., "MyWorkflow.output_file" becomes "output_file"
func stripWorkflowPrefix(key, workflowName string) string {
	prefix := workflowName + "."
	if strings.HasPrefix(key, prefix) {
		return strings.TrimPrefix(key, prefix)
	}
	return key
}
