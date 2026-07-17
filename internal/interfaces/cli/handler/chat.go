package handler

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/urfave/cli/v2"
	adksession "google.golang.org/adk/session"
	"google.golang.org/adk/tool"

	"github.com/lmtani/pumbaa/internal/application/ports"
	"github.com/lmtani/pumbaa/internal/config"
	"github.com/lmtani/pumbaa/internal/interfaces/tui"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/chat"
	"github.com/lmtani/pumbaa/internal/prompts"
)

// ChatDepsProvider builds the chat dependency bundle (LLM, agent tools,
// session service). rebuildWDLIndex forces a rebuild of the WDL index cache,
// and extraTools is the extension point for standalone ADK tools. It returns
// nil deps and no error when the LLM is not configured, and an error when
// initialization fails. Wired to Container.ChatDependencies.
type ChatDepsProvider func(rebuildWDLIndex bool, extraTools ...tool.Tool) (*tui.ChatDependencies, error)

// SessionStoreProvider opens the chat session store; it does not require an
// LLM, so session listing works even when chat is not configured. Wired to
// Container.SessionStore.
type SessionStoreProvider func() (ports.ChatSessionStore, error)

// Session scope shared with the embedded TUI chat (see ports package).
const appName = ports.DefaultChatAppName
const defaultUserID = ports.DefaultChatUserID

type ChatHandler struct {
	config       *config.Config
	telemetry    ports.Telemetry
	chatDeps     ChatDepsProvider
	sessionStore SessionStoreProvider
}

func NewChatHandler(cfg *config.Config, ts ports.Telemetry, chatDeps ChatDepsProvider, sessionStore SessionStoreProvider) *ChatHandler {
	return &ChatHandler{config: cfg, telemetry: ts, chatDeps: chatDeps, sessionStore: sessionStore}
}

func (h *ChatHandler) Command() *cli.Command {
	return &cli.Command{
		Name:  "chat",
		Usage: "Interact with the Pumbaa agent",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "session",
				Aliases: []string{"s"},
				Usage:   "Session ID to resume",
			},
			&cli.BoolFlag{
				Name:    "list",
				Aliases: []string{"l"},
				Usage:   "List available sessions",
			},
			&cli.StringFlag{
				Name:    "provider",
				Aliases: []string{"p"},
				Usage:   "LLM provider: ollama, vertex, or gemini",
				EnvVars: []string{"PUMBAA_LLM_PROVIDER"},
			},
			&cli.StringFlag{
				Name:    "vertex-project",
				Usage:   "Google Cloud project for Vertex AI",
				EnvVars: []string{"VERTEX_PROJECT"},
			},
			&cli.StringFlag{
				Name:    "vertex-location",
				Usage:   "Vertex AI location",
				EnvVars: []string{"VERTEX_LOCATION"},
			},
			&cli.StringFlag{
				Name:    "vertex-model",
				Usage:   "Vertex AI model",
				EnvVars: []string{"VERTEX_MODEL"},
			},
			&cli.StringFlag{
				Name:    "wdl-dir",
				Usage:   "Directory containing WDL workflows for context",
				EnvVars: []string{"PUMBAA_WDL_DIR"},
			},
			&cli.BoolFlag{
				Name:  "rebuild-index",
				Usage: "Force rebuild of WDL index cache",
			},
			&cli.StringFlag{
				Name:    "gemini-api-key",
				Usage:   "API key for Gemini (Google AI Studio)",
				EnvVars: []string{"GEMINI_API_KEY"},
			},
			&cli.StringFlag{
				Name:    "gemini-model",
				Usage:   "Gemini model name",
				EnvVars: []string{"GEMINI_MODEL"},
			},
		},
		Action: func(c *cli.Context) error {
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
			// Handle WDL dir
			if wd := c.String("wdl-dir"); wd != "" {
				h.config.WDLDirectory = wd
			}
			// Handle Gemini flags
			if gk := c.String("gemini-api-key"); gk != "" {
				h.config.GeminiAPIKey = gk
			}
			if gm := c.String("gemini-model"); gm != "" {
				h.config.GeminiModel = gm
			}
			return h.Run(c.String("session"), c.Bool("rebuild-index"))
		},
	}
}

func (h *ChatHandler) ListSessions() error {
	store, err := h.sessionStore()
	if err != nil {
		return fmt.Errorf("failed to initialize session service: %w", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	sessions, err := store.ListWithSummaries(ctx, appName, defaultUserID)
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	if len(sessions) == 0 {
		fmt.Println("No sessions found.")
		return nil
	}

	fmt.Println("Available sessions:")
	for _, s := range sessions {
		summary := s.Summary
		if summary == "" {
			summary = "(no summary)"
		}
		if s.ContextLabel != "" {
			summary = s.ContextLabel + ": " + summary
		}
		// Truncate summary to 60 chars for display
		if len(summary) > 60 {
			summary = summary[:57] + "..."
		}
		fmt.Printf("  - %-20s │ %s │ %s\n",
			s.ID[:20],
			s.UpdatedAt.Format("2006-01-02 15:04"),
			summary,
		)
	}
	return nil
}

func (h *ChatHandler) Run(sessionID string, rebuildIndex bool) error {
	deps, err := h.chatDeps(rebuildIndex)
	if err != nil {
		return err
	}
	if deps == nil {
		return fmt.Errorf("LLM not configured: set PUMBAA_LLM_PROVIDER (ollama, vertex or gemini)")
	}
	svc := deps.SessionSvc
	if store, ok := svc.(ports.ChatSessionStore); ok {
		defer func() { _ = store.Close() }()
	}

	ctx := context.Background()

	var sess adksession.Session
	if sessionID != "" {
		resp, err := svc.Get(ctx, &adksession.GetRequest{AppName: appName, UserID: defaultUserID, SessionID: sessionID})
		if err != nil {
			return fmt.Errorf("failed to get session %s: %w", sessionID, err)
		}
		sess = resp.Session
		h.telemetry.AddBreadcrumb("chat", "resumed existing session")
		fmt.Printf("Resuming session: %s\n", sessionID)
	} else {
		resp, err := svc.Create(ctx, &adksession.CreateRequest{AppName: appName, UserID: defaultUserID})
		if err != nil {
			return fmt.Errorf("failed to create session: %w", err)
		}
		sess = resp.Session
		h.telemetry.AddBreadcrumb("chat", "created new session")
		fmt.Printf("Created new session: %s\n", sess.ID())
	}

	h.telemetry.AddBreadcrumb("chat", fmt.Sprintf("using LLM provider: %s", h.config.LLMProvider))
	fmt.Printf("Using LLM: %s | Cromwell: %s\n", deps.LLM.Name(), h.config.CromwellHost)

	m := chat.NewModel(deps.LLM, deps.Tools, prompts.Chat, svc, sess)
	m.SetStandalone(true) // Running directly from CLI, not embedded in TUI

	p := tea.NewProgram(&m, tea.WithAltScreen())
	m.SetProgram(p)
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
	return nil
}
