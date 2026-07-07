package handler

import (
	"fmt"
	"os"

	"google.golang.org/adk/tool"

	"github.com/lmtani/pumbaa/internal/application/ports"
	"github.com/lmtani/pumbaa/internal/config"
	"github.com/lmtani/pumbaa/internal/infrastructure/agents/llm"
	"github.com/lmtani/pumbaa/internal/infrastructure/agents/tools"
	"github.com/lmtani/pumbaa/internal/infrastructure/agents/tools/wdl"
	"github.com/lmtani/pumbaa/internal/infrastructure/session"
	wdlindexer "github.com/lmtani/pumbaa/internal/infrastructure/wdl"
	"github.com/lmtani/pumbaa/internal/interfaces/tui"
)

// initWDLRepository builds the WDL index repository from config. Returns nil
// (WDL actions disabled) when no directory is configured or indexing fails.
// The index is cached at cfg.WDLIndexPath, so subsequent startups are instant.
func initWDLRepository(cfg *config.Config, forceRebuild bool) wdl.Repository {
	if cfg.WDLDirectory == "" {
		return nil
	}
	indexer, err := wdlindexer.NewIndexer(cfg.WDLDirectory, cfg.WDLIndexPath, forceRebuild)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: WDL tools disabled - indexing failed: %v\n", err)
		return nil
	}
	if idx, err := indexer.List(); err == nil {
		fmt.Printf("WDL index: %d tasks, %d workflows\n", len(idx.Tasks), len(idx.Workflows))
	}
	return indexer
}

// initChatDependencies creates the optional chat dependencies for TUI screens.
// Returns nil if LLM or session initialization fails (chat is silently disabled).
//
// extraTools is the extension point for adding standalone ADK tools to the
// chat agent beyond the built-in pumbaa tool; see the tools package docs.
func initChatDependencies(cfg *config.Config, repo ports.WorkflowReader, extraTools ...tool.Tool) *tui.ChatDependencies {
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

	agentTools := tools.GetAllTools(repo, initWDLRepository(cfg, false), extraTools...)

	return &tui.ChatDependencies{
		LLM:        llmModel,
		Tools:      agentTools,
		SessionSvc: svc,
	}
}
