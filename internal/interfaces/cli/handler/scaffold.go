package handler

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/lmtani/pumbaa/internal/application/workflow"
	"github.com/lmtani/pumbaa/internal/interfaces/cli/presenter"
	"github.com/lmtani/pumbaa/pkg/wdl"
)

// ScaffoldHandler handles the inputs template command.
type ScaffoldHandler struct {
	useCase   *workflow.ScaffoldInputsUseCase
	presenter *presenter.Presenter
}

// NewScaffoldHandler creates a new ScaffoldHandler.
func NewScaffoldHandler(uc *workflow.ScaffoldInputsUseCase, p *presenter.Presenter) *ScaffoldHandler {
	return &ScaffoldHandler{useCase: uc, presenter: p}
}

// Command returns the CLI command for scaffolding an inputs file.
func (h *ScaffoldHandler) Command() *cli.Command {
	return &cli.Command{
		Name:    "scaffold",
		Aliases: []string{"template"},
		Usage:   "Generate an inputs JSON template from a WDL",
		Description: "Reads the workflow's own declarations and writes an inputs file to fill in:\n" +
			"required inputs first, each with its type and documentation. Without --output\n" +
			"the template goes to stdout, so it can be redirected to a file.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "workflow",
				Aliases:  []string{"w"},
				Usage:    "[required] Path to the WDL workflow file",
				Required: true,
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "[optional] Write the template to this file instead of stdout",
			},
			&cli.BoolFlag{
				Name:  "all",
				Usage: "[optional] Include optional inputs, with their default values",
			},
			&cli.BoolFlag{
				Name:  "force",
				Usage: "[optional] Overwrite the output file if it already exists",
			},
		},
		Action: h.handle,
	}
}

func (h *ScaffoldHandler) handle(c *cli.Context) error {
	workflowFile := c.String("workflow")
	outputFile := c.String("output")

	output, err := h.useCase.Execute(context.Background(), workflow.ScaffoldInputsInput{
		WorkflowFile:    workflowFile,
		IncludeOptional: c.Bool("all"),
	})
	if err != nil {
		return err
	}

	// No destination: the template is the output, so it can be piped.
	if outputFile == "" {
		h.presenter.Print("%s", output.Template)
		return nil
	}

	if err := writeTemplate(outputFile, output.Template, c.Bool("force")); err != nil {
		return err
	}

	h.presenter.Success("Wrote %s for workflow %s", outputFile, output.WorkflowName)
	h.renderInputs(output.Inputs)
	h.renderNextSteps(workflowFile, outputFile, output.Inputs)
	return nil
}

// writeTemplate writes the template, refusing to clobber an existing file
// unless asked: a filled-in inputs file is expensive to lose.
func writeTemplate(path string, content []byte, force bool) error {
	if !force {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("%s already exists (use --force to overwrite)", path)
		} else if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("failed to check %s: %w", path, err)
		}
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	return nil
}

// renderInputs explains every input, including the optional ones left out of
// the template — the teaching moment of the command.
func (h *ScaffoldHandler) renderInputs(inputs []wdl.InputSpec) {
	if len(inputs) == 0 {
		h.presenter.Info("This workflow declares no inputs.")
		return
	}

	h.presenter.Newline()
	table := h.presenter.NewTable([]string{"INPUT", "TYPE", "REQUIRED", "DEFAULT", "DESCRIPTION"})
	for _, in := range inputs {
		required := "no"
		if in.Required() {
			required = "yes"
		}
		_ = table.Append([]string{in.Name, in.Type, required, in.Default, in.Description})
	}
	_ = table.Render()
}

func (h *ScaffoldHandler) renderNextSteps(workflowFile, outputFile string, inputs []wdl.InputSpec) {
	required := 0
	for _, in := range inputs {
		if in.Required() {
			required++
		}
	}

	h.presenter.Newline()
	h.presenter.Title("Next steps")
	h.presenter.Println(fmt.Sprintf("  1. Replace the %d placeholder value(s) in %s", required, outputFile))
	h.presenter.Println(fmt.Sprintf("  2. pumbaa workflow preflight -w %s -i %s", workflowFile, outputFile))
	h.presenter.Println(fmt.Sprintf("  3. pumbaa workflow submit -w %s -i %s", workflowFile, outputFile))
}
