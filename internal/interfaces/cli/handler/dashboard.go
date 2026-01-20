package handler

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/urfave/cli/v2"

	"github.com/lmtani/pumbaa/internal/application/ports"
	workflowapp "github.com/lmtani/pumbaa/internal/application/workflow"
	"github.com/lmtani/pumbaa/internal/config"
	"github.com/lmtani/pumbaa/internal/infrastructure/agents/llm"
	"github.com/lmtani/pumbaa/internal/infrastructure/agents/tools"
	cromwellclient "github.com/lmtani/pumbaa/internal/infrastructure/cromwell"
	"github.com/lmtani/pumbaa/internal/infrastructure/session"
	"github.com/lmtani/pumbaa/internal/infrastructure/telemetry"
	"github.com/lmtani/pumbaa/internal/interfaces/tui"
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
	version        string
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
	version string,
) *DashboardHandler {
	return &DashboardHandler{
		repository:     client,
		telemetry:      ts,
		monitoringUC:   muc,
		fileProvider:   fp,
		metadataParser: mp,
		batchLogsUC:    bluc,
		config:         cfg,
		version:        version,
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

	// Create shared dependencies
	deps := h.createDependencies()

	// Create the unified app model starting at dashboard
	model := tui.NewAppModel(deps, tui.ScreenDashboard)

	// Create and run the program
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err := p.Run()
	if err != nil {
		h.telemetry.CaptureError("dashboard.tui", err)
		return fmt.Errorf("error running dashboard: %w", err)
	}

	h.telemetry.AddBreadcrumb("navigation", "exiting dashboard")
	return nil
}

// createDependencies creates the shared dependencies for the TUI.
func (h *DashboardHandler) createDependencies() *tui.Dependencies {
	deps := &tui.Dependencies{
		Repository:     h.repository,
		FileProvider:   h.fileProvider,
		MetadataParser: h.metadataParser,
		MonitoringUC:   h.monitoringUC,
		BatchLogsUC:    h.batchLogsUC,
		CurrentVersion: h.version,
	}

	// Initialize chat dependencies if LLM is configured
	if h.config != nil && h.config.LLMProvider != "" {
		deps.ChatDeps = h.initializeChatDependencies()
	}

	return deps
}

// initializeChatDependencies creates the chat dependencies for the TUI.
func (h *DashboardHandler) initializeChatDependencies() *tui.ChatDependencies {
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

	return &tui.ChatDependencies{
		LLM:        llmModel,
		Tools:      agentTools,
		SessionSvc: svc,
	}
}
