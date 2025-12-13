package handler

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmtani/pumbaa/internal/application/workflow/debuginfo"
	"github.com/lmtani/pumbaa/internal/domain/workflow"
	"github.com/lmtani/pumbaa/internal/domain/workflow/preemption"
	"github.com/lmtani/pumbaa/internal/infrastructure/cromwell"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/dashboard"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/debug"
	"github.com/urfave/cli/v2"
)

// DashboardHandler handles the dashboard TUI command.
type DashboardHandler struct {
	client *cromwell.Client
}

// NewDashboardHandler creates a new dashboard handler.
func NewDashboardHandler(client *cromwell.Client) *DashboardHandler {
	return &DashboardHandler{
		client: client,
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
	fetcher := &dashboardFetcher{client: h.client}

	for {
		// Create dashboard model with metadata fetcher for smooth transitions
		model := dashboard.NewModelWithFetcher(fetcher)
		model.SetMetadataFetcher(h.client)

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

		// Check if user wants to quit
		if dashModel.ShouldQuit {
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
				metadataBytes, err = h.client.GetRawMetadataWithOptions(c.Context, dashModel.NavigateToDebugID, false)
				if err != nil {
					fmt.Printf("Error fetching metadata: %v\n", err)
					continue
				}
			}

			err := h.runDebugWithMetadata(metadataBytes)
			if err != nil {
				// Log error but continue - will restart dashboard
				fmt.Printf("Error opening debug view: %v\n", err)
			}
			// After debug closes, loop back to restart dashboard
			continue
		}

		// Normal exit
		return nil
	}
}

func (h *DashboardHandler) runDebugWithMetadata(metadataBytes []byte) error {
	// Build DebugInfo using usecase
	uc := debuginfo.NewUsecase(preemption.NewAnalyzer())
	di, err := uc.GetDebugInfo(metadataBytes)
	if err != nil {
		return fmt.Errorf("failed to build debug info: %w", err)
	}

	// Create and run the debug TUI
	model := debug.NewModelWithDebugInfo(di, h.client)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running debug TUI: %w", err)
	}

	return nil
}

// dashboardFetcher adapts the Cromwell client to the dashboard.WorkflowFetcher interface
type dashboardFetcher struct {
	client *cromwell.Client
}

func (f *dashboardFetcher) Query(ctx context.Context, filter workflow.QueryFilter) (*workflow.QueryResult, error) {
	return f.client.Query(ctx, filter)
}

func (f *dashboardFetcher) Abort(ctx context.Context, workflowID string) error {
	return f.client.Abort(ctx, workflowID)
}
