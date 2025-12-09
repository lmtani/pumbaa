package handler

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmtani/pumbaa/internal/application/workflow/debuginfo"
	"github.com/lmtani/pumbaa/internal/domain/workflow/preemption"
	"github.com/lmtani/pumbaa/internal/infrastructure/cromwell"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/debug"
	"github.com/urfave/cli/v2"
)

// DebugHandler handles workflow debug TUI commands.
type DebugHandler struct {
	client *cromwell.Client
}

// NewDebugHandler creates a new debug handler.
func NewDebugHandler(client *cromwell.Client) *DebugHandler {
	return &DebugHandler{
		client: client,
	}
}

// Command returns the CLI command for debug.
func (h *DebugHandler) Command() *cli.Command {
	return &cli.Command{
		Name:  "debug",
		Usage: "Interactive TUI for debugging workflow execution",
		Description: `Opens an interactive terminal UI to explore workflow metadata.

Navigate through the call tree, view task details, commands, inputs,
outputs, and execution timeline.

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
  t             View execution timeline
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
		metadataBytes, err = os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
	} else {
		// Fetch from Cromwell
		metadataBytes, err = h.client.GetRawMetadataWithOptions(c.Context, workflowID, expandSubWorkflows)
		if err != nil {
			return fmt.Errorf("failed to fetch metadata: %w", err)
		}
	}

	uc := debuginfo.NewUsecase(preemption.NewAnalyzer())
	di, err := uc.GetDebugInfo(metadataBytes)
	if err != nil {
		return fmt.Errorf("failed to build debug info: %w", err)
	}

	// Create and run the TUI using precomputed DebugInfo
	var model debug.Model
	if workflowID != "" {
		model = debug.NewModelWithDebugInfo(di, h.client)
	} else {
		model = debug.NewModelWithDebugInfo(di, nil)
	}
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running TUI: %w", err)
	}

	return nil
}
