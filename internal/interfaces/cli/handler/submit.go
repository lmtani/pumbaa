// Package handler provides CLI command handlers.
package handler

import (
	"context"
	"errors"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/lmtani/pumbaa/internal/application/workflow"
	"github.com/lmtani/pumbaa/internal/interfaces/cli/presenter"
)

// SubmitHandler handles the workflow submission command.
type SubmitHandler struct {
	useCase   *workflow.SubmitUseCase
	presenter *presenter.Presenter
}

// NewSubmitHandler creates a new SubmitHandler.
func NewSubmitHandler(uc *workflow.SubmitUseCase, p *presenter.Presenter) *SubmitHandler {
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
			&cli.BoolFlag{
				Name:  "skip-preflight",
				Usage: "[optional] Submit without checking the workflow and inputs first",
			},
		},
		Action: h.handle,
	}
}

func (h *SubmitHandler) handle(c *cli.Context) error {
	ctx := context.Background()

	// Parse labels (format: key=value)
	labels := make(map[string]string)
	for _, l := range c.StringSlice("label") {
		parts := strings.SplitN(l, "=", 2)
		if len(parts) == 2 {
			labels[parts[0]] = parts[1]
		} else {
			labels[l] = ""
		}
	}

	input := workflow.SubmitInput{
		WorkflowFile:     c.String("workflow"),
		InputsFile:       c.String("inputs"),
		OptionsFile:      c.String("options"),
		DependenciesFile: c.String("dependencies"),
		Labels:           labels,
		SkipPreflight:    c.Bool("skip-preflight"),
	}

	output, err := h.useCase.Execute(ctx, input)
	if err != nil {
		// Preflight found problems: show the whole checklist rather than a
		// single message, so everything can be fixed in one pass.
		var preflightErr *workflow.PreflightFailedError
		if errors.As(err, &preflightErr) {
			renderPreflightReport(h.presenter, preflightErr.Report)
			h.presenter.Newline()
			h.presenter.Info("Nothing was submitted. Fix the problems above, or use --skip-preflight to submit anyway.")
			return cli.Exit("", 1)
		}
		h.presenter.Error("Failed to submit workflow: %v", err)
		return err
	}

	// Confirm preflight ran: silent success would make the feature look like
	// it never happened, and would swallow non-blocking warnings.
	reportPreflightBeforeSubmit(h.presenter, output.Preflight, c.Bool("skip-preflight"))

	h.presenter.Success("Workflow submitted successfully!")
	h.presenter.KeyValue("Workflow ID", output.WorkflowID)
	h.presenter.KeyValue("Status", h.presenter.StatusColor(output.Status))

	return nil
}

// reportPreflightBeforeSubmit shows the outcome of the pre-submission checks:
// the full checklist when there are warnings worth seeing, a one-line
// confirmation when everything was clean, and a note when it was skipped.
func reportPreflightBeforeSubmit(p *presenter.Presenter, report *workflow.PreflightReport, skipped bool) {
	if skipped || report == nil {
		p.Info("Preflight skipped.")
		return
	}
	if _, warnCount := report.Counts(); warnCount > 0 {
		renderPreflightReport(p, report)
		p.Newline()
		return
	}
	p.Success("Preflight passed.")
}
