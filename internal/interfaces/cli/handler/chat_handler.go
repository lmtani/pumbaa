package handler

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/urfave/cli/v2"
	adksession "google.golang.org/adk/session"

	"github.com/lmtani/pumbaa/internal/config"
	"github.com/lmtani/pumbaa/internal/infrastructure/chat/agent/tools"
	"github.com/lmtani/pumbaa/internal/infrastructure/chat/agent/tools/wdl"
	"github.com/lmtani/pumbaa/internal/infrastructure/chat/llm"
	cromwellclient "github.com/lmtani/pumbaa/internal/infrastructure/cromwell"
	"github.com/lmtani/pumbaa/internal/infrastructure/session"
	"github.com/lmtani/pumbaa/internal/infrastructure/telemetry"
	wdlindexer "github.com/lmtani/pumbaa/internal/infrastructure/wdl"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/chat"
)

const appName = "pumbaa"
const defaultUserID = "default"

const systemInstruction = `You are Pumbaa, a helpful assistant specialized in bioinformatics workflows and Cromwell/WDL.

You have access to the "pumbaa" tool with these actions:

# Cromwell + WDL Agent

This agent operates in **two distinct domains**.  
**Never mix runtime operations with WDL definitions.**

---

## 1. Execution Operations (Cromwell Runtime)

Use **only** when the question is about workflows already submitted:
status, failures, logs, outputs, or runtime metadata.

### Actions
- action="query"  
  Search workflow executions  
  Optional: status (Running | Succeeded | Failed), name

- action="status"  
  Get execution status  
  Required: workflow_id

- action="metadata"  
  Get full execution metadata (calls, inputs, outputs)  
  Required: workflow_id

- action="outputs"  
  List output files  
  Required: workflow_id

- action="logs"  
  Get log file paths for debugging  
  Required: workflow_id

---

## 2. Files (Google Cloud Storage)

Use **only** to read real files produced by executions.

- action="gcs_download"  
  Read file from GCS  
  Required: path (gs://bucket/file)

---

## 3. Knowledge Base (Workflow WDL Context)

Use **only** to understand or explain WDL definitions.  
**Does not access runtime or real executions.**

### Actions
- action="wdl_list"  
  List indexed WDL tasks and workflows
d
- action="wdl_search"  
  Search by name or command content  
  Required: query

- action="wdl_info"  
  Get task or workflow details  
  Required: name, type (task | workflow)

---

## Decision Rules

- “Status / failed / logs / outputs?” → **Cromwell**
- “What does this task do / inputs / command?” → **WDL**
- Failure debugging:
  1. Cromwell (query → logs)
  2. GCS (gcs_download)
  3. WDL **only to explain the code**

---

## Guidelines

- Prefer query before using workflow_id
- Do not mix runtime (Cromwell) with definition (WDL)
- Be concise and technical
- Use markdown to format responses
- Respond in the user’s language (EN or PT)
`

type ChatHandler struct {
	config    *config.Config
	telemetry telemetry.Service
}

func NewChatHandler(cfg *config.Config, ts telemetry.Service) *ChatHandler {
	return &ChatHandler{config: cfg, telemetry: ts}
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

func (h *ChatHandler) Run(sessionID string, rebuildIndex bool) error {
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

	llmModel, err := llm.NewLLM(h.config)
	if err != nil {
		return fmt.Errorf("failed to initialize LLM: %w", err)
	}
	h.telemetry.AddBreadcrumb("chat", fmt.Sprintf("using LLM provider: %s", h.config.LLMProvider))
	fmt.Printf("Using LLM: %s | Cromwell: %s\n", llmModel.Name(), h.config.CromwellHost)

	cromwellClient := cromwellclient.NewClient(cromwellclient.Config{
		Host:    h.config.CromwellHost,
		Timeout: h.config.CromwellTimeout,
	})

	// Initialize WDL indexer if configured
	var wdlRepo wdl.Repository
	if h.config.WDLDirectory != "" {
		fmt.Printf("Indexing WDL workflows from: %s\n", h.config.WDLDirectory)
		indexer, err := wdlindexer.NewIndexer(h.config.WDLDirectory, h.config.WDLIndexPath, rebuildIndex)
		if err != nil {
			fmt.Printf("Warning: Failed to initialize WDL indexer: %v\n", err)
		} else {
			wdlRepo = indexer
			idx, _ := indexer.List()
			fmt.Printf("WDL index: %d tasks, %d workflows\n", len(idx.Tasks), len(idx.Workflows))
		}
	}

	agentTools := tools.GetAllTools(cromwellClient, wdlRepo)
	m := chat.NewModel(llmModel, agentTools, systemInstruction, svc, sess)

	p := tea.NewProgram(&m, tea.WithAltScreen())
	m.SetProgram(p)
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
	return nil
}
