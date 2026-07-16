package chat

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"google.golang.org/adk/session"

	"github.com/lmtani/pumbaa/internal/application/ports"
)

// Session lifecycle commands: lazy creation on first send and
// resume-by-task lookup (ctrl+r).
// findResumableCmd looks up a previous session for the same task context, so
// the user can resume it with ctrl+r instead of starting over.
func (m *Model) findResumableCmd() tea.Cmd {
	if m.session != nil || m.contextLabel == "" {
		return nil
	}
	svc, ok := m.sessionService.(ports.ChatSessionStore)
	if !ok {
		return nil
	}
	label := m.contextLabel
	owner := m.msgs
	return func() tea.Msg {
		info, err := svc.FindLatestByContextLabel(context.Background(), ports.DefaultChatAppName, ports.DefaultChatUserID, label)
		if err != nil || info == nil {
			return nil
		}
		return resumableFoundMsg{owner: owner, info: *info}
	}
}

// createSessionCmd creates the persistent session on first use and tags it
// with the task context for later resume-by-task lookups.
func (m *Model) createSessionCmd() tea.Cmd {
	svc := m.sessionService
	label := m.contextLabel
	owner := m.msgs
	return func() tea.Msg {
		ctx := context.Background()
		resp, err := svc.Create(ctx, &session.CreateRequest{
			AppName: ports.DefaultChatAppName,
			UserID:  ports.DefaultChatUserID,
		})
		if err != nil {
			return sessionCreatedMsg{owner: owner, err: err}
		}
		if label != "" {
			if store, ok := svc.(ports.ChatSessionStore); ok {
				_ = store.SetContextLabel(ctx, resp.Session.ID(), label)
			}
		}
		return sessionCreatedMsg{owner: owner, session: resp.Session}
	}
}
