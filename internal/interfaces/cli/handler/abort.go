// Package handler provides CLI command handlers.
package handler

import (
	"context"

	"github.com/lmtani/pumbaa/internal/application/workflow/abort"
	"github.com/lmtani/pumbaa/internal/interfaces/cli/presenter"
	"github.com/urfave/cli/v2"
)

// AbortHandler handles the workflow abort command.
type AbortHandler struct {
	useCase   *abort.UseCase
	presenter *presenter.Presenter
}

// NewAbortHandler creates a new AbortHandler.
func NewAbortHandler(uc *abort.UseCase, p *presenter.Presenter) *AbortHandler {
	return &AbortHandler{
		useCase:   uc,
		presenter: p,
	}
}

// Command returns the CLI command for aborting workflows.
func (h *AbortHandler) Command() *cli.Command {
	return &cli.Command{
		Name:      "abort",
		Aliases:   []string{"a", "kill"},
		Usage:     "Abort a running workflow",
		ArgsUsage: "<workflow-id>",
		Action:    h.handle,
	}
}

func (h *AbortHandler) handle(c *cli.Context) error {
	if c.NArg() < 1 {
		h.presenter.Error("Workflow ID is required")
		return cli.Exit("workflow ID required", 1)
	}

	ctx := context.Background()
	workflowID := c.Args().First()

	input := abort.Input{
		WorkflowID: workflowID,
	}

	output, err := h.useCase.Execute(ctx, input)
	if err != nil {
		h.presenter.Error("Failed to abort workflow: %v", err)
		return err
	}

	h.presenter.Success(output.Message)
	h.presenter.KeyValue("Workflow ID", output.WorkflowID)

	return nil
}
