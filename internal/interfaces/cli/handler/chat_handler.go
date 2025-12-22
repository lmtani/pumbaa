package handler

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmtani/pumbaa/internal/config"
	"github.com/lmtani/pumbaa/internal/infrastructure/agent/tools"
	"github.com/lmtani/pumbaa/internal/infrastructure/llm"
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
			&cli.StringFlag{
				Name:    "provider",
				Aliases: []string{"p"},
				Usage:   "LLM provider: ollama or vertex (default: ollama)",
				EnvVars: []string{"PUMBAA_LLM_PROVIDER"},
			},
			&cli.StringFlag{
				Name:    "vertex-project",
				Usage:   "Google Cloud project for Vertex AI",
				EnvVars: []string{"VERTEX_PROJECT"},
			},
			&cli.StringFlag{
				Name:    "vertex-location",
				Usage:   "Vertex AI location (default: us-central1)",
				EnvVars: []string{"VERTEX_LOCATION"},
			},
			&cli.StringFlag{
				Name:    "vertex-model",
				Usage:   "Vertex AI model (default: gemini-2.0-flash)",
				EnvVars: []string{"VERTEX_MODEL"},
			},
		},
		Action: func(c *cli.Context) error {
			// Apply flag overrides
			if p := c.String("provider"); p != "" {
				h.config.LLMProvider = p
			}
			if vp := c.String("vertex-project"); vp != "" {
				h.config.VertexProject = vp
			}
			if vl := c.String("vertex-location"); vl != "" {
				h.config.VertexLocation = vl
			}
			if vm := c.String("vertex-model"); vm != "" {
				h.config.VertexModel = vm
			}

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

	// Initialize LLM using factory
	llmModel, err := llm.NewLLM(h.config)
	if err != nil {
		return fmt.Errorf("failed to initialize LLM: %w", err)
	}
	fmt.Printf("Using LLM provider: %s\n", llmModel.Name())

	// Initialize Tools
	agentTools := tools.GetAllTools()

	// Initialize Chat Model with session
	m := chat.NewModel(llmModel, agentTools, systemInstruction, svc, sess)

	// Run Bubble Tea Program
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
	return nil
}
