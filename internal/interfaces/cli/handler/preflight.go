package handler

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/lmtani/pumbaa/internal/application/workflow"
	"github.com/lmtani/pumbaa/internal/interfaces/cli/presenter"
)

// PreflightHandler handles the pre-submission check command.
type PreflightHandler struct {
	useCase   *workflow.PreflightUseCase
	presenter *presenter.Presenter
}

// NewPreflightHandler creates a new PreflightHandler.
func NewPreflightHandler(uc *workflow.PreflightUseCase, p *presenter.Presenter) *PreflightHandler {
	return &PreflightHandler{useCase: uc, presenter: p}
}

// Command returns the CLI command for preflight checks.
func (h *PreflightHandler) Command() *cli.Command {
	return &cli.Command{
		Name:    "preflight",
		Aliases: []string{"check"},
		Usage:   "Check a workflow and its inputs before submitting",
		Description: "Verifies that Cromwell is reachable, the WDL parses, every required input is\n" +
			"present and well-typed, and the files the inputs point at exist — so a broken\n" +
			"submission fails in seconds instead of minutes.",
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
				Name:    "dependencies",
				Aliases: []string{"d"},
				Usage:   "[optional] Path to the dependencies ZIP; its imports are checked",
			},
			&cli.BoolFlag{
				Name:  "skip-paths",
				Usage: "[optional] Do not check that input files exist",
			},
			&cli.BoolFlag{
				Name:  "skip-server",
				Usage: "[optional] Do not check that Cromwell is reachable",
			},
		},
		Action: h.handle,
	}
}

func (h *PreflightHandler) handle(c *cli.Context) error {
	report, err := h.useCase.Execute(context.Background(), workflow.PreflightInput{
		WorkflowFile:     c.String("workflow"),
		InputsFile:       c.String("inputs"),
		DependenciesFile: c.String("dependencies"),
		SkipPaths:        c.Bool("skip-paths"),
		SkipServer:       c.Bool("skip-server"),
	})
	if err != nil {
		return err
	}

	renderPreflightReport(h.presenter, report)

	if report.HasErrors() {
		// Non-zero exit so scripts and CI can gate on it.
		return cli.Exit("", 1)
	}
	return nil
}

// renderPreflightReport prints the checklist. Shared with the submit handler,
// which shows the same report when it refuses to submit.
func renderPreflightReport(p *presenter.Presenter, r *workflow.PreflightReport) {
	title := "Preflight"
	if r.WorkflowName != "" {
		title += " — workflow " + r.WorkflowName
	}
	p.Title(title)

	for _, check := range r.Checks {
		p.Print("  %s %-16s %s\n", checkSymbol(check.Status), check.Name, check.Detail)
		for _, item := range check.Items {
			p.Print("      %s %s\n", itemSymbol(item.Severity), itemText(item))
		}
	}

	errCount, warnCount := r.Counts()
	p.Newline()
	switch {
	case errCount > 0:
		p.Error("%d problem(s) must be fixed before this run can start%s", errCount, warnSuffix(warnCount))
	case warnCount > 0:
		p.Warning("ready to submit, with %d warning(s) worth a look", warnCount)
	default:
		p.Success("ready to submit")
	}
}

// itemText prefixes the message with the input or path it is about.
func itemText(item workflow.PreflightItem) string {
	if item.Subject == "" {
		return item.Message
	}
	return fmt.Sprintf("%s: %s", item.Subject, item.Message)
}

func checkSymbol(status workflow.CheckStatus) string {
	switch status {
	case workflow.CheckOK:
		return "✓"
	case workflow.CheckWarning:
		return "⚠"
	case workflow.CheckFailed:
		return "✗"
	default:
		return "·"
	}
}

func itemSymbol(severity string) string {
	if severity == "error" {
		return "✗"
	}
	return "⚠"
}

func warnSuffix(warnCount int) string {
	if warnCount == 0 {
		return ""
	}
	return fmt.Sprintf(" (%d warning(s) too)", warnCount)
}
