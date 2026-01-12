package handler

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	adksession "google.golang.org/adk/session"

	"github.com/lmtani/pumbaa/internal/interfaces/tui/chat"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/debug"
)

const debugChatAppName = "pumbaa-debug"
const debugChatUserID = "default"

func runDebugChat(systemInstruction string, chatDeps *debug.ChatDependencies) error {
	if chatDeps == nil || chatDeps.LLM == nil {
		return fmt.Errorf("chat dependencies not configured")
	}

	ctx := context.Background()

	var sess adksession.Session
	if chatDeps.SessionSvc != nil {
		resp, err := chatDeps.SessionSvc.Create(ctx, &adksession.CreateRequest{
			AppName: debugChatAppName,
			UserID:  debugChatUserID,
		})
		if err != nil {
			return fmt.Errorf("failed to create chat session: %w", err)
		}
		sess = resp.Session
	}

	model := chat.NewModel(chatDeps.LLM, chatDeps.Tools, systemInstruction, chatDeps.SessionSvc, sess)
	p := tea.NewProgram(&model, tea.WithAltScreen())
	model.SetProgram(p)
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running chat: %w", err)
	}

	return nil
}
