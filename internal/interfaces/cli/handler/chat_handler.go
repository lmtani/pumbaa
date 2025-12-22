package handler

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmtani/pumbaa/internal/config"
	"github.com/lmtani/pumbaa/internal/infrastructure/agent/tools"
	cromwellclient "github.com/lmtani/pumbaa/internal/infrastructure/cromwell"
	"github.com/lmtani/pumbaa/internal/infrastructure/llm"
	"github.com/lmtani/pumbaa/internal/infrastructure/session"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/chat"
	"github.com/urfave/cli/v2"
	adksession "google.golang.org/adk/session"
)

const appName = "pumbaa"
const defaultUserID = "default"

const systemInstruction = `You are Pumbaa, a helpful assistant specialized in bioinformatics workflows and Cromwell/WDL.

You have access to the "pumbaa" tool with these actions:

**Cromwell Server:**
- action="query" → Search workflows. Optional: status (Running, Succeeded, Failed), name
- action="status" → Get workflow status. Required: workflow_id
- action="metadata" → Get workflow details (calls, inputs, outputs). Required: workflow_id
- action="outputs" → Get workflow output files. Required: workflow_id
- action="logs" → Get log file paths for debugging. Required: workflow_id

**Google Cloud Storage:**
- action="gcs_download" → Read file from GCS. Required: path (gs://bucket/file)

Guidelines:
- Use action="query" to find workflows first
- Use action="logs" + action="gcs_download" to debug failures
- Be helpful and concise
- Respond in the user's language (Portuguese or English)`

type ChatHandler struct {
	config *config.Config
}

func NewChatHandler(cfg *config.Config) *ChatHandler {
	return &ChatHandler{config: cfg}
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
				Usage:   "LLM provider: ollama or vertex",
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
	resp, err := svc.List(ctx, &adksession.ListRequest{AppName: appName, UserID: defaultUserID})
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
	svc, err := session.NewSQLiteService(h.config.SessionDBPath)
	if err != nil {
		return fmt.Errorf("failed to initialize session service: %w", err)
	}
	defer svc.Close()

	ctx := context.Background()

	var sess adksession.Session
	if sessionID != "" {
		resp, err := svc.Get(ctx, &adksession.GetRequest{AppName: appName, UserID: defaultUserID, SessionID: sessionID})
		if err != nil {
			return fmt.Errorf("failed to get session %s: %w", sessionID, err)
		}
		sess = resp.Session
		fmt.Printf("Resuming session: %s\n", sessionID)
	} else {
		resp, err := svc.Create(ctx, &adksession.CreateRequest{AppName: appName, UserID: defaultUserID})
		if err != nil {
			return fmt.Errorf("failed to create session: %w", err)
		}
		sess = resp.Session
		fmt.Printf("Created new session: %s\n", sess.ID())
	}

	llmModel, err := llm.NewLLM(h.config)
	if err != nil {
		return fmt.Errorf("failed to initialize LLM: %w", err)
	}
	fmt.Printf("Using LLM: %s | Cromwell: %s\n", llmModel.Name(), h.config.CromwellHost)

	cromwellClient := cromwellclient.NewClient(cromwellclient.Config{
		Host:    h.config.CromwellHost,
		Timeout: h.config.CromwellTimeout,
	})

	agentTools := tools.GetAllTools(cromwellClient)
	m := chat.NewModel(llmModel, agentTools, systemInstruction, svc, sess)

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
	return nil
}
