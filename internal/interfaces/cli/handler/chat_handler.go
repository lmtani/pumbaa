package handler

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmtani/pumbaa/internal/config"
	"github.com/lmtani/pumbaa/internal/infrastructure/agent/tools"
	"github.com/lmtani/pumbaa/internal/infrastructure/ollama"
	"github.com/lmtani/pumbaa/internal/infrastructure/session"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/chat"
	"github.com/urfave/cli/v2"
	adksession "google.golang.org/adk/session"
)

const appName = "pumbaa"
const defaultUserID = "default"

const systemInstruction = `You are Pumbaa, a helpful assistant specialized in bioinformatics workflows and Cromwell/WDL.

You help users understand and manage their Cromwell workflows. You have access to the following tools:

1. **gcs_download**: Downloads and reads content from Google Cloud Storage (gs:// paths). Use this when users provide GCS paths.

Guidelines:
- Always be helpful and concise.
- When users provide gs:// paths, use gcs_download to read the content.
- Explain workflow concepts clearly when asked.
- If you don't know something, say so.

Respond in the same language the user uses (Portuguese or English).`

type ChatHandler struct {
	config *config.Config
}

func NewChatHandler(cfg *config.Config) *ChatHandler {
	return &ChatHandler{
		config: cfg,
	}
}

func (h *ChatHandler) Command() *cli.Command {
	return &cli.Command{
		Name:  "chat",
		Usage: "Interact with the Pumbaa agent",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "session",
				Aliases: []string{"s"},
				Usage:   "Session ID to resume (leave empty for new session)",
			},
			&cli.BoolFlag{
				Name:    "list",
				Aliases: []string{"l"},
				Usage:   "List available sessions",
			},
		},
		Action: func(c *cli.Context) error {
			if c.Bool("list") {
				return h.ListSessions()
			}
			return h.Run(c.String("session"))
		},
	}
}

func (h *ChatHandler) ListSessions() error {
	svc, err := session.NewSQLiteService(h.config.SessionDBPath)
	if err != nil {
		return fmt.Errorf("failed to initialize session service: %w", err)
	}
	defer svc.Close()

	ctx := context.Background()
	resp, err := svc.List(ctx, &adksession.ListRequest{
		AppName: appName,
		UserID:  defaultUserID,
	})
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	if len(resp.Sessions) == 0 {
		fmt.Println("No sessions found.")
		return nil
	}

	fmt.Println("Available sessions:")
	for _, s := range resp.Sessions {
		fmt.Printf("  - %s (last updated: %s)\n", s.ID(), s.LastUpdateTime().Format("2006-01-02 15:04:05"))
	}
	return nil
}

func (h *ChatHandler) Run(sessionID string) error {
	// Initialize session service
	svc, err := session.NewSQLiteService(h.config.SessionDBPath)
	if err != nil {
		return fmt.Errorf("failed to initialize session service: %w", err)
	}
	defer svc.Close()

	ctx := context.Background()

	// Get or create session
	var sess adksession.Session
	if sessionID != "" {
		// Resume existing session
		resp, err := svc.Get(ctx, &adksession.GetRequest{
			AppName:   appName,
			UserID:    defaultUserID,
			SessionID: sessionID,
		})
		if err != nil {
			return fmt.Errorf("failed to get session %s: %w", sessionID, err)
		}
		sess = resp.Session
		fmt.Printf("Resuming session: %s\n", sessionID)
	} else {
		// Create new session
		resp, err := svc.Create(ctx, &adksession.CreateRequest{
			AppName: appName,
			UserID:  defaultUserID,
		})
		if err != nil {
			return fmt.Errorf("failed to create session: %w", err)
		}
		sess = resp.Session
		fmt.Printf("Created new session: %s\n", sess.ID())
	}

	// Initialize LLM
	llm := ollama.NewModel(h.config.OllamaHost, h.config.OllamaModel)

	// Initialize Tools
	agentTools := tools.GetAllTools()

	// Initialize Chat Model with session
	m := chat.NewModel(llm, agentTools, systemInstruction, svc, sess)

	// Run Bubble Tea Program
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
	return nil
}
