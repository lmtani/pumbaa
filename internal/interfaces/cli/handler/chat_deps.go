package handler

import (
	"fmt"
	"os"

	"github.com/lmtani/pumbaa/internal/application/ports"
	"github.com/lmtani/pumbaa/internal/config"
	"github.com/lmtani/pumbaa/internal/infrastructure/agents/llm"
	"github.com/lmtani/pumbaa/internal/infrastructure/agents/tools"
	"github.com/lmtani/pumbaa/internal/infrastructure/session"
	"github.com/lmtani/pumbaa/internal/interfaces/tui"
)

// initChatDependencies creates the optional chat dependencies for TUI screens.
// Returns nil if LLM or session initialization fails (chat is silently disabled).
func initChatDependencies(cfg *config.Config, repo ports.WorkflowReader) *tui.ChatDependencies {
	llmModel, err := llm.NewLLM(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Chat disabled - LLM initialization failed: %v\n", err)
		return nil
	}

	svc, err := session.NewSQLiteService(cfg.SessionDBPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Chat disabled - Session service failed: %v\n", err)
		return nil
	}

	agentTools := tools.GetAllTools(repo, nil)

	return &tui.ChatDependencies{
		LLM:        llmModel,
		Tools:      agentTools,
		SessionSvc: svc,
	}
}
