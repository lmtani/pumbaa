// Package handler provides CLI command handlers.
package handler

import (
	"context"
	"path/filepath"

	"github.com/urfave/cli/v2"

	"github.com/lmtani/pumbaa/internal/application/bundle/create"
	"github.com/lmtani/pumbaa/internal/interfaces/cli/presenter"
)

// BundleHandler handles the WDL bundle command.
type BundleHandler struct {
	useCase   *create.UseCase
	presenter *presenter.Presenter
}

// NewBundleHandler creates a new BundleHandler.
func NewBundleHandler(uc *create.UseCase, p *presenter.Presenter) *BundleHandler {
	return &BundleHandler{
		useCase:   uc,
		presenter: p,
	}
}

// Command returns the CLI command for creating WDL bundles.
func (h *BundleHandler) Command() *cli.Command {
	return &cli.Command{
		Name:    "bundle",
		Aliases: []string{"b", "pack"},
		Usage:   "Create a WDL bundle with all dependencies",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "workflow",
				Aliases:  []string{"w"},
				Usage:    "[required] Path to the main WDL workflow file",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "output",
				Aliases:  []string{"o"},
				Usage:    "[required] Output path for the bundle ZIP file",
				Required: true,
			},
		},
		Action: h.handle,
	}
}

func (h *BundleHandler) handle(c *cli.Context) error {
	ctx := context.Background()

	input := create.Input{
		MainWorkflowPath: c.String("workflow"),
		OutputPath:       c.String("output"),
	}

	h.presenter.Info("Analyzing workflow dependencies...")

	output, err := h.useCase.Execute(ctx, input)
	if err != nil {
		h.presenter.Error("Failed to create bundle: %v", err)
		return err
	}

	h.presenter.Success("Bundle created successfully!")
	h.presenter.Newline()
	h.presenter.Title("Output Files")
	h.presenter.KeyValue("Main WDL", output.MainWDLPath)
	if output.DependenciesZipPath != "" {
		h.presenter.KeyValue("Dependencies ZIP", output.DependenciesZipPath)
	}
	h.presenter.KeyValue("Total Files", output.TotalFiles)

	if len(output.Dependencies) > 0 {
		h.presenter.Newline()
		h.presenter.Title("Dependencies (included in ZIP)")
		for _, dep := range output.Dependencies {
			h.presenter.Print("  • %s\n", filepath.Base(dep))
		}
	} else {
		h.presenter.Newline()
		h.presenter.Info("No dependencies found - only main WDL generated")
	}

	return nil
}
