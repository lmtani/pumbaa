// Package handler provides CLI command handlers.
package handler

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/lmtani/pumbaa/internal/application/workflow"
	"github.com/lmtani/pumbaa/internal/interfaces/cli/presenter"
)

// AnalyzeHandler handles the analyze commands.
type AnalyzeHandler struct {
	visualizationUseCase *workflow.ResourceVisualizationUseCase
	presenter            *presenter.Presenter
}

// NewAnalyzeHandler creates a new AnalyzeHandler.
func NewAnalyzeHandler(uc *workflow.ResourceVisualizationUseCase, p *presenter.Presenter) *AnalyzeHandler {
	return &AnalyzeHandler{
		visualizationUseCase: uc,
		presenter:            p,
	}
}

// Command returns the CLI command for analyze operations.
func (h *AnalyzeHandler) Command() *cli.Command {
	return &cli.Command{
		Name:  "analyze",
		Usage: "Analyze collected workflow data",
		Subcommands: []*cli.Command{
			{
				Name:      "resources",
				Usage:     "Generate HTML visualization from resource report TSV files",
				ArgsUsage: "<directory>",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
						Usage:   "Output HTML file path",
						Value:   "resource_report.html",
					},
					&cli.BoolFlag{
						Name:  "no-llm",
						Usage: "Skip LLM-based recommendations (faster)",
					},
					&cli.IntFlag{
						Name:    "llm-batch-size",
						Usage:   "Number of tasks per LLM request",
						Value:   25,
						Aliases: []string{"b"},
					},
					&cli.StringFlag{
						Name:  "llm-debug",
						Usage: "File path to write LLM debug logs (prompts and responses)",
					},
				},
				Action: h.handleResources,
			},
		},
	}
}

func (h *AnalyzeHandler) handleResources(c *cli.Context) error {
	if c.NArg() < 1 {
		h.presenter.Error("Directory path is required")
		return cli.Exit("directory path required", 1)
	}

	ctx := context.Background()
	directory := c.Args().First()
	outputFile := c.String("output")

	input := workflow.ResourceVisualizationInput{
		Directory:    directory,
		OutputFile:   outputFile,
		SkipLLM:      c.Bool("no-llm"),
		LLMBatchSize: c.Int("llm-batch-size"),
		LLMDebugFile: c.String("llm-debug"),
	}

	h.presenter.Info("Scanning TSV files in %s...", directory)

	output, err := h.visualizationUseCase.Execute(ctx, input)
	if err != nil {
		h.presenter.Error("Failed to generate visualization: %v", err)
		return err
	}

	h.presenter.Newline()
	h.presenter.Title("Resource Analysis Report Generated")
	h.presenter.KeyValue("Workflows Analyzed", fmt.Sprintf("%d", output.WorkflowCount))
	h.presenter.KeyValue("Unique Tasks", fmt.Sprintf("%d", output.TaskCount))
	h.presenter.KeyValue("Output File", output.OutputFile)
	h.presenter.Newline()
	h.presenter.Success("✓ Open %s in your browser to view the report", output.OutputFile)

	return nil
}
