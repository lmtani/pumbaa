package handler

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/urfave/cli/v2"

	workflowapp "github.com/lmtani/pumbaa/internal/application/workflow"
	"github.com/lmtani/pumbaa/internal/domain/ports"
	"github.com/lmtani/pumbaa/internal/infrastructure/storage"
	"github.com/lmtani/pumbaa/internal/infrastructure/telemetry"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/dashboard"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/debug"
)

// DashboardHandler handles the dashboard TUI command.
type DashboardHandler struct {
	repository ports.WorkflowRepository
	telemetry  telemetry.Service
}

// NewDashboardHandler creates a new dashboard handler.
func NewDashboardHandler(client ports.WorkflowRepository, ts telemetry.Service) *DashboardHandler {
	return &DashboardHandler{
		repository: client,
		telemetry:  ts,
	}
}

// Command returns the CLI command for dashboard.
func (h *DashboardHandler) Command() *cli.Command {
	return &cli.Command{
		Name:    "dashboard",
		Aliases: []string{"dash"},
		Usage:   "Interactive TUI dashboard for Cromwell workflows",
		Description: `Opens an interactive terminal UI to view and manage workflows.

Browse workflows, filter by status or name, and navigate to debug view.

KEY BINDINGS:
  ↑/↓           Navigate through workflows
  Enter         Open workflow in debug view
  a             Abort running workflow
  s             Cycle status filter (All/Running/Failed/Succeeded)
  /             Filter by workflow name
  Ctrl+X        Clear all filters
  r             Refresh workflow list
  q             Quit`,
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:    "status",
				Aliases: []string{"s"},
				Usage:   "Filter by status (Running, Succeeded, Failed)",
			},
			&cli.StringFlag{
				Name:    "name",
				Aliases: []string{"n"},
				Usage:   "Filter by workflow name",
			},
		},
		Action: h.handle,
	}
}

func (h *DashboardHandler) handle(c *cli.Context) error {
	h.telemetry.AddBreadcrumb("navigation", "entering dashboard")

	for {
		// Create dashboard model with TUI client
		model := dashboard.NewModelWithFetcher(h.repository)
		model.SetMetadataFetcher(h.repository)
		model.SetHealthChecker(h.repository)
		model.SetLabelManager(h.repository)

		// Create and run the program
		p := tea.NewProgram(model, tea.WithAltScreen())
		finalModel, err := p.Run()
		if err != nil {
			return fmt.Errorf("error running dashboard: %w", err)
		}

		// Check the final state
		dashModel, ok := finalModel.(dashboard.Model)
		if !ok {
			// Should not happen, but handle gracefully
			return nil
		}

		// Capture any TUI errors for telemetry before exiting
		if dashModel.LastError != nil {
			h.telemetry.CaptureError("dashboard.tui", dashModel.LastError)
		}

		// Check if user wants to quit
		if dashModel.ShouldQuit {
			h.telemetry.AddBreadcrumb("navigation", "exiting dashboard")
			return nil
		}

		// Check if we need to navigate to debug view
		if dashModel.NavigateToDebugID != "" {
			var metadataBytes []byte

			// Use pre-fetched metadata if available
			if dashModel.DebugMetadataReady != nil {
				metadataBytes = dashModel.DebugMetadataReady
			} else {
				// Fallback: fetch metadata (shouldn't happen with new flow)
				var err error
				metadataBytes, err = h.repository.GetRawMetadataWithOptions(c.Context, dashModel.NavigateToDebugID, false)
				if err != nil {
					fmt.Printf("Error fetching metadata: %v\n", err)
					h.telemetry.CaptureError("dashboard.fetchMetadata", err)
					continue
				}
			}
			h.telemetry.AddBreadcrumb("navigation", fmt.Sprintf("opening debug view for %s", dashModel.NavigateToDebugID[:8]))
			err := h.runDebugWithMetadata(metadataBytes)
			if err != nil {
				// Log error and send to telemetry
				fmt.Printf("Error opening debug view: %v\n", err)
				h.telemetry.CaptureError("dashboard.runDebugWithMetadata", err)
			}
			// After debug closes, loop back to restart dashboard
			h.telemetry.AddBreadcrumb("navigation", "returning to dashboard from debug")
			continue
		}

		// Normal exit
		return nil
	}
}

func (h *DashboardHandler) runDebugWithMetadata(metadataBytes []byte) error {
	// Build DebugInfo using usecase
	uc := workflowapp.NewUsecase()
	di, err := uc.GetDebugInfo(metadataBytes)
	if err != nil {
		return fmt.Errorf("failed to build debug info: %w", err)
	}

	// Initialize infrastructure and use cases
	fp := storage.NewFileProvider()
	muc := workflowapp.NewMonitoringUseCase(fp)

	// Create and run the debug TUI
	model := debug.NewModelWithDebugInfoAndMonitoring(di, h.repository, muc, fp)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running debug TUI: %w", err)
	}

	return nil
}
