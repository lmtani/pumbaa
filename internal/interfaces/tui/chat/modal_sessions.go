package chat

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"google.golang.org/adk/model"
	"google.golang.org/adk/session"
	"google.golang.org/genai"

	infraSession "github.com/lmtani/pumbaa/internal/infrastructure/session"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

const summaryPrompt = `Summarize this conversation in 1-2 sentences (max 80 chars):
- Main topic
- Actions taken
- Outcome

Conversation:
%s

Summary:`

// openSessionsModal initializes and opens the sessions modal (pointer receiver needed to modify state)
func (m *Model) openSessionsModal() (tea.Model, tea.Cmd) {
	m.activeModal = ModalSessions
	m.sessionsCursor = 0
	m.sessionsLoading = true
	m.sessionsError = ""
	return m, m.loadSessionsList()
}

// loadSessionsList returns a command that fetches all sessions
func (m *Model) loadSessionsList() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Cast sessionService to SQLiteService to access ListWithSummaries
		if svc, ok := m.sessionService.(*infraSession.SQLiteService); ok {
			const appName = "pumbaa"
			const defaultUserID = "default"

			sessions, err := svc.ListWithSummaries(ctx, appName, defaultUserID)
			if err != nil {
				return sessionListErrorMsg{err: err}
			}
			return sessionListLoadedMsg{sessions: sessions}
		}
		return sessionListErrorMsg{err: fmt.Errorf("session service doesn't support ListWithSummaries")}
	}
}

// renderSessionsModal renders the sessions list modal
func (m *Model) renderSessionsModal() string {
	modalWidth := 70
	modalHeight := 18

	title := common.HeaderTitleStyle.Render("🔄 Chat Sessions")

	if m.sessionsLoading {
		content := common.MutedStyle.Render("Loading sessions...")
		footer := common.MutedStyle.Render("esc close")
		return m.renderCenteredSessionsModal(modalWidth, modalHeight, title, content, footer)
	}

	if m.sessionsError != "" {
		content := common.ErrorStyle.Render("Error: " + m.sessionsError)
		footer := common.MutedStyle.Render("esc close")
		return m.renderCenteredSessionsModal(modalWidth, modalHeight, title, content, footer)
	}

	if len(m.sessionsList) == 0 {
		content := common.MutedStyle.Render("No sessions found. Press 'n' to create a new session.")
		footer := common.MutedStyle.Render("n new • esc close")
		return m.renderCenteredSessionsModal(modalWidth, modalHeight, title, content, footer)
	}

	// Build session list content
	var lines []string
	for i, sess := range m.sessionsList {
		isSelected := i == m.sessionsCursor
		line := m.formatSessionLine(sess, isSelected, modalWidth-4)
		lines = append(lines, line)
	}

	content := strings.Join(lines, "\n")
	footer := common.MutedStyle.Render("↑↓ navigate • enter select • n new • esc close")

	return m.renderCenteredSessionsModal(modalWidth, modalHeight, title, content, footer)
}

// formatSessionLine formats a single session entry for display
func (m Model) formatSessionLine(sess infraSession.SessionInfo, selected bool, maxWidth int) string {
	// Truncate ID to 16 chars
	idDisplay := sess.ID
	if len(idDisplay) > 16 {
		idDisplay = idDisplay[:13] + "..."
	}

	// Truncate summary to 30 chars
	summary := sess.Summary
	if summary == "" {
		summary = "(no summary)"
	}
	if len(summary) > 30 {
		summary = summary[:27] + "..."
	}

	// Format date
	dateStr := sess.UpdatedAt.Format("01-02")

	// Format tokens
	totalTokens := sess.InputTokens + sess.OutputTokens
	tokenStr := fmt.Sprintf("%dK", (totalTokens+999)/1000) // Round up
	if totalTokens < 1000 {
		tokenStr = fmt.Sprintf("%d", totalTokens)
	}

	// Build line
	line := fmt.Sprintf("%-16s │ %-30s │ %s │ %s", idDisplay, summary, dateStr, tokenStr)

	if selected {
		// Highlight selected line
		selectedStyle := lipgloss.NewStyle().
			Background(common.PrimaryColor).
			Foreground(lipgloss.Color("#000000"))
		return selectedStyle.Render("▶ " + line)
	}
	return "  " + line
}

// renderCenteredSessionsModal renders a centered modal with the given dimensions
func (m Model) renderCenteredSessionsModal(width, height int, title, content, footer string) string {
	// Ensure dimensions don't exceed terminal
	if width > m.width-4 {
		width = m.width - 4
	}
	if height > m.height-4 {
		height = m.height - 4
	}

	// Build the modal
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(common.PrimaryColor).
		Padding(1, 2).
		Width(width).
		Height(height)

	// Combine content and footer
	fullContent := content
	if footer != "" {
		fullContent = content + "\n\n" + footer
	}

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		modalStyle.Render(lipgloss.JoinVertical(lipgloss.Left, title, "", fullContent)),
	)
}

// handleSessionsModalKeys handles keyboard input in the sessions modal
func (m *Model) handleSessionsModalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c", "q":
		m.activeModal = ModalNone
		return m, nil

	case "up":
		if m.sessionsCursor > 0 {
			m.sessionsCursor--
		}
		return m, nil

	case "down":
		if m.sessionsCursor < len(m.sessionsList)-1 {
			m.sessionsCursor++
		}
		return m, nil

	case "enter":
		if len(m.sessionsList) > 0 {
			selected := m.sessionsList[m.sessionsCursor]
			return m.switchToSession(selected.ID)
		}
		return m, nil

	case "n":
		return m.createNewSession()
	}

	return m, nil
}

// switchToSession switches to a different session
func (m *Model) switchToSession(sessionID string) (tea.Model, tea.Cmd) {
	m.sessionsLoading = true
	return m, func() tea.Msg {
		ctx := context.Background()

		// Generate summary for current session before switching
		if m.session != nil && m.msgs != nil && len(*m.msgs) > 0 {
			m.generateAndSaveSummary(ctx)
		}

		// Load new session
		if svc, ok := m.sessionService.(*infraSession.SQLiteService); ok {
			const appName = "pumbaa"
			const defaultUserID = "default"

			resp, err := svc.Get(ctx, &session.GetRequest{
				AppName:   appName,
				UserID:    defaultUserID,
				SessionID: sessionID,
			})
			if err != nil {
				return sessionListErrorMsg{err: err}
			}

			// Load history from session
			history := make([]*genai.Content, 0)
			if resp.Session.Events() != nil {
				for ev := range resp.Session.Events().All() {
					if ev.Content != nil {
						history = append(history, ev.Content)
					}
				}
			}

			return sessionSwitchedMsg{
				session: resp.Session,
				history: history,
			}
		}
		return sessionListErrorMsg{err: fmt.Errorf("session service doesn't support Get")}
	}
}

// createNewSession creates a new chat session
func (m *Model) createNewSession() (tea.Model, tea.Cmd) {
	m.sessionsLoading = true
	return m, func() tea.Msg {
		ctx := context.Background()

		// Generate summary for current session before creating new
		if m.session != nil && m.msgs != nil && len(*m.msgs) > 0 {
			m.generateAndSaveSummary(ctx)
		}

		// Create new session
		if svc, ok := m.sessionService.(*infraSession.SQLiteService); ok {
			const appName = "pumbaa"
			const defaultUserID = "default"

			resp, err := svc.Create(ctx, &session.CreateRequest{
				AppName: appName,
				UserID:  defaultUserID,
			})
			if err != nil {
				return sessionListErrorMsg{err: err}
			}

			return sessionSwitchedMsg{
				session: resp.Session,
				history: make([]*genai.Content, 0),
			}
		}
		return sessionListErrorMsg{err: fmt.Errorf("session service doesn't support Create")}
	}
}

// generateAndSaveSummary generates a summary and saves it to the database
func (m *Model) generateAndSaveSummary(ctx context.Context) {
	if m.llm == nil || m.msgs == nil || m.session == nil || len(*m.msgs) < 2 {
		return
	}

	// Build conversation text
	var sb strings.Builder
	for _, msg := range *m.msgs {
		if msg.Role == "info" {
			continue
		}
		content := msg.Content
		if len(content) > 200 {
			content = content[:197] + "..."
		}
		sb.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, content))
	}

	// Generate summary using LLM
	prompt := fmt.Sprintf(summaryPrompt, sb.String())
	userContent := &genai.Content{
		Role: "user",
		Parts: []*genai.Part{
			genai.NewPartFromText(prompt),
		},
	}

	req := &model.LLMRequest{
		Contents: []*genai.Content{userContent},
	}

	respSeq := m.llm.GenerateContent(ctx, req, false)

	var summary string
	for r, e := range respSeq {
		if e != nil {
			return // Silently fail
		}
		if r != nil && r.Content != nil {
			for _, part := range r.Content.Parts {
				if part.Text != "" {
					summary = strings.TrimSpace(part.Text)
					// Truncate to reasonable length
					if len(summary) > 200 {
						summary = summary[:197] + "..."
					}
					break
				}
			}
		}
	}

	// Save summary to database
	if svc, ok := m.sessionService.(*infraSession.SQLiteService); ok {
		go svc.UpdateSummary(context.Background(), m.session.ID(), summary)
	}
}
