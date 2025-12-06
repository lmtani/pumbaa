// Package handler provides CLI command handlers.
package handler

import (
	"context"

	"github.com/lmtani/pumbaa/internal/application/workflow/submit"
	"github.com/lmtani/pumbaa/internal/interfaces/cli/presenter"
	"github.com/urfave/cli/v2"
)

// SubmitHandler handles the workflow submission command.
type SubmitHandler struct {
	useCase   *submit.UseCase
	presenter *presenter.Presenter
}

// NewSubmitHandler creates a new SubmitHandler.
func NewSubmitHandler(uc *submit.UseCase, p *presenter.Presenter) *SubmitHandler {
	return &SubmitHandler{
		useCase:   uc,
		presenter: p,
	}
}

// Command returns the CLI command for workflow submission.
func (h *SubmitHandler) Command() *cli.Command {
	return &cli.Command{
		Name:    "submit",
		Aliases: []string{"s"},
		Usage:   "Submit a workflow to Cromwell",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "workflow",
				Aliases:  []string{"w"},
				Usage:    "[required] Path to the WDL workflow file",
				Required: true,
			},
			&cli.StringFlag{
				Name:    "inputs",
				Aliases: []string{"i"},
				Usage:   "[optional] Path to the inputs JSON file",
			},
			&cli.StringFlag{
				Name:    "options",
				Aliases: []string{"o"},
				Usage:   "[optional] Path to the options JSON file",
			},
			&cli.StringFlag{
				Name:    "dependencies",
				Aliases: []string{"d"},
				Usage:   "[optional] Path to the dependencies ZIP file",
			},
			&cli.StringSliceFlag{
				Name:    "label",
				Aliases: []string{"l"},
				Usage:   "[optional] Labels to attach to the workflow (format: key=value)",
			},
		},
		Action: h.handle,
	}
}

func (h *SubmitHandler) handle(c *cli.Context) error {
	ctx := context.Background()

	// Parse labels
	labels := make(map[string]string)
	for _, l := range c.StringSlice("label") {
		// Parse key=value format
		// This is simplified - you might want more robust parsing
		labels[l] = "" // TODO: proper parsing
	}

	input := submit.Input{
		WorkflowFile:     c.String("workflow"),
		InputsFile:       c.String("inputs"),
		OptionsFile:      c.String("options"),
		DependenciesFile: c.String("dependencies"),
		Labels:           labels,
	}

	output, err := h.useCase.Execute(ctx, input)
	if err != nil {
		h.presenter.Error("Failed to submit workflow: %v", err)
		return err
	}

	h.presenter.Success("Workflow submitted successfully!")
	h.presenter.KeyValue("Workflow ID", output.WorkflowID)
	h.presenter.KeyValue("Status", h.presenter.StatusColor(output.Status))

	return nil
}
