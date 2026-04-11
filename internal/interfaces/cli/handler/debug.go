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
	"github.com/lmtani/pumbaa/internal/infrastructure/session"
	"github.com/lmtani/pumbaa/internal/infrastructure/telemetry"
	"github.com/lmtani/pumbaa/internal/interfaces/tui"
)

// DebugHandler handles workflow debug TUI commands.
type DebugHandler struct {
	repository   ports.WorkflowRepository
	telemetry    telemetry.Service
	monitoringUC *workflowapp.MonitoringUseCase
	fileProvider ports.FileProvider
	batchLogsUC  *workflowapp.GetBatchLogsUseCase
	config       *config.Config
}

// NewDebugHandler creates a new debug handler.
func NewDebugHandler(
	client ports.WorkflowRepository,
	ts telemetry.Service,
	muc *workflowapp.MonitoringUseCase,
	fp ports.FileProvider,
	bluc *workflowapp.GetBatchLogsUseCase,
	cfg *config.Config,
) *DebugHandler {
	return &DebugHandler{
		repository:   client,
		telemetry:    ts,
		monitoringUC: muc,
		fileProvider: fp,
		batchLogsUC:  bluc,
		config:       cfg,
	}
}

// Command returns the CLI command for debug.
func (h *DebugHandler) Command() *cli.Command {
	return &cli.Command{
		Name:  "debug",
		Usage: "Interactive TUI for debugging workflow execution",
		Description: `Opens an interactive terminal UI to explore workflow metadata.

Navigate through the call tree, view task details, commands, inputs,
and outputs.

USAGE EXAMPLES:
  # Debug a workflow by ID (fetches metadata from Cromwell)
  pumbaa workflow debug --id abc123

  # Debug from a local metadata JSON file
  pumbaa workflow debug --file metadata.json

KEY BINDINGS:
  ↑/↓ or j/k    Navigate through the tree
  ←/→ or h/l    Collapse/expand nodes
  Enter/Space   Toggle expand
  Tab           Switch between tree and details panel
  d             View details (default view)
  c             View task command
  L             View log paths (stdout/stderr)
  i             View task inputs
  o             View task outputs
  t             View tasks duration (workflows/subworkflows)
  E             Expand all nodes
  C             Collapse all nodes
  ?             Show help
  q             Quit`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "id",
				Aliases: []string{"i"},
				Usage:   "[optional] Workflow ID to debug",
			},
			&cli.StringFlag{
				Name:    "file",
				Aliases: []string{"f"},
				Usage:   "[optional] Path to metadata JSON file",
			},
			&cli.BoolFlag{
				Name:    "expand-subworkflows",
				Aliases: []string{"e"},
				Usage:   "Pre-expand all subworkflows metadata (may be slow for large workflows)",
				Value:   false,
			},
		},
		Action: h.handle,
	}
}

func (h *DebugHandler) handle(c *cli.Context) error {
	workflowID := c.String("id")
	filePath := c.String("file")
	expandSubWorkflows := c.Bool("expand-subworkflows")

	if workflowID == "" && filePath == "" {
		return fmt.Errorf("either --id or --file must be provided")
	}

	var metadataBytes []byte
	var err error

	if filePath != "" {
		// Load from file
		h.telemetry.AddBreadcrumb("navigation", fmt.Sprintf("debug from file: %s", filePath))
		metadataBytes, err = os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
	} else {
		// Fetch from Cromwell
		h.telemetry.AddBreadcrumb("navigation", fmt.Sprintf("debug workflow: %s", workflowID[:8]))
		metadataBytes, err = h.repository.GetRawMetadataWithOptions(c.Context, workflowID, expandSubWorkflows)
		if err != nil {
			return fmt.Errorf("failed to fetch metadata: %w", err)
		}
	}

	// Parse metadata
	wf, err := h.repository.ParseMetadata(metadataBytes)
	if err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	// Create shared dependencies
	deps := h.createDependencies()

	// Create the unified app model starting at debug screen with workflow
	model := tui.NewAppModelWithWorkflow(deps, wf)

	// Create and run the program
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err = p.Run()
	if err != nil {
		h.telemetry.CaptureError("debug.tui", err)
		return fmt.Errorf("error running debug TUI: %w", err)
	}

	return nil
}

// createDependencies creates the shared dependencies for the TUI.
func (h *DebugHandler) createDependencies() *tui.Dependencies {
	deps := &tui.Dependencies{
		Repository:   h.repository,
		FileProvider: h.fileProvider,
		MonitoringUC: h.monitoringUC,
		BatchLogsUC:  h.batchLogsUC,
	}

	// Initialize chat dependencies if LLM is configured
	if h.config != nil && h.config.LLMProvider != "" {
		deps.ChatDeps = h.initializeChatDependencies()
	}

	return deps
}

// initializeChatDependencies creates the chat dependencies for the TUI.
func (h *DebugHandler) initializeChatDependencies() *tui.ChatDependencies {
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

	// Initialize tools using the existing repository (without WDL for now)
	agentTools := tools.GetAllTools(h.repository, nil)

	return &tui.ChatDependencies{
		LLM:        llmModel,
		Tools:      agentTools,
		SessionSvc: svc,
	}
}
