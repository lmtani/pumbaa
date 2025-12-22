package handler

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmtani/pumbaa/internal/config"
	"github.com/lmtani/pumbaa/internal/infrastructure/agent/tools"
	"github.com/lmtani/pumbaa/internal/infrastructure/ollama"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/chat"
	"github.com/urfave/cli/v2"
)

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
		Action: func(c *cli.Context) error {
			return h.Run()
		},
	}
}

func (h *ChatHandler) Run() error {
	// Initialize LLM
	llm := ollama.NewModel(h.config.OllamaHost, h.config.OllamaModel)

	// Initialize Tools
	agentTools := tools.GetAllTools()

	// Initialize Chat Model with system instruction
	m := chat.NewModel(llm, agentTools, systemInstruction)

	// Run Bubble Tea Program
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
	return nil
}
