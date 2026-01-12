package handler

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/urfave/cli/v2"

	workflowapp "github.com/lmtani/pumbaa/internal/application/workflow"
	"github.com/lmtani/pumbaa/internal/config"
	"github.com/lmtani/pumbaa/internal/domain/ports"
	"github.com/lmtani/pumbaa/internal/infrastructure/chat/agent/tools"
	"github.com/lmtani/pumbaa/internal/infrastructure/chat/llm"
	cromwellclient "github.com/lmtani/pumbaa/internal/infrastructure/cromwell"
	"github.com/lmtani/pumbaa/internal/infrastructure/session"
	"github.com/lmtani/pumbaa/internal/infrastructure/telemetry"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/dashboard"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/debug"
)

// DashboardHandler handles the dashboard TUI command.
type DashboardHandler struct {
	repository     ports.WorkflowRepository
	telemetry      telemetry.Service
	monitoringUC   *workflowapp.MonitoringUseCase
	fileProvider   ports.FileProvider
	metadataParser ports.MetadataParser
	batchLogsUC    *workflowapp.GetBatchLogsUseCase
	config         *config.Config
}

// NewDashboardHandler creates a new dashboard handler.
func NewDashboardHandler(
	client ports.WorkflowRepository,
	ts telemetry.Service,
	muc *workflowapp.MonitoringUseCase,
	fp ports.FileProvider,
	mp ports.MetadataParser,
	bluc *workflowapp.GetBatchLogsUseCase,
	cfg *config.Config,
) *DashboardHandler {
	return &DashboardHandler{
		repository:     client,
		telemetry:      ts,
		monitoringUC:   muc,
		fileProvider:   fp,
		metadataParser: mp,
		batchLogsUC:    bluc,
		config:         cfg,
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
	// Parse metadata using injected parser
	wf, err := h.metadataParser.ParseMetadata(metadataBytes)
	if err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	// Initialize chat dependencies if LLM is configured
	var chatDeps *debug.ChatDependencies
	if h.config != nil && h.config.LLMProvider != "" {
		chatDeps = h.initializeChatDependencies()
	}

	for {
		// Create and run the debug TUI (tree building happens inside NewModel)
		model := debug.NewModelWithChat(wf, h.repository, h.monitoringUC, h.fileProvider, h.batchLogsUC, chatDeps)
		p := tea.NewProgram(model, tea.WithAltScreen())

		finalModel, err := p.Run()
		if err != nil {
			return fmt.Errorf("error running debug TUI: %w", err)
		}

		debugModel, ok := finalModel.(debug.Model)
		if !ok {
			return nil
		}

		if debugModel.NavigateToChatSystemInstruction != "" {
			if err := runDebugChat(debugModel.NavigateToChatSystemInstruction, chatDeps); err != nil {
				return err
			}
			continue
		}

		return nil
	}
}

// initializeChatDependencies creates the chat dependencies for the debug TUI.
func (h *DashboardHandler) initializeChatDependencies() *debug.ChatDependencies {
	// Try to initialize LLM
	llmModel, err := llm.NewLLM(h.config)
	if err != nil {
		// Log the error so user knows why chat is disabled
		fmt.Fprintf(os.Stderr, "Warning: Chat disabled - LLM initialization failed: %v\n", err)
		return nil
	}

	// Initialize session service
	svc, err := session.NewSQLiteService(h.config.SessionDBPath)
	if err != nil {
		// Log the error so user knows why chat is disabled
		fmt.Fprintf(os.Stderr, "Warning: Chat disabled - Session service failed: %v\n", err)
		return nil
	}

	// Create Cromwell client for tools
	cromwellClient := cromwellclient.NewClient(cromwellclient.Config{
		Host:    h.config.CromwellHost,
		Timeout: h.config.CromwellTimeout,
	})

	// Initialize tools (without WDL for now)
	agentTools := tools.GetAllTools(cromwellClient, nil)

	return &debug.ChatDependencies{
		LLM:        llmModel,
		Tools:      agentTools,
		SessionSvc: svc,
	}
}
