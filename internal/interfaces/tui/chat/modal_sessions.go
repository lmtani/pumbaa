package chat

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
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

// openSessionsModal initializes and opens the sessions modal
func (m *Model) openSessionsModal() (tea.Model, tea.Cmd) {
	m.activeModal = ModalSessions
	m.sessionsCursor = 0
	m.sessionsLoading = true
	m.sessionsError = ""
	m.sessionsSearch = ""
	m.sessionsSearching = false

	// Initialize search input
	ti := textinput.New()
	ti.Placeholder = "Type to filter..."
	ti.CharLimit = 50
	ti.Width = 40
	m.sessionsSearchInput = ti

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

// getFilteredSessions returns sessions filtered by search term
func (m *Model) getFilteredSessions() []infraSession.SessionInfo {
	if m.sessionsSearch == "" {
		return m.sessionsList
	}

	search := strings.ToLower(m.sessionsSearch)
	var filtered []infraSession.SessionInfo
	for _, sess := range m.sessionsList {
		if strings.Contains(strings.ToLower(sess.ID), search) ||
			strings.Contains(strings.ToLower(sess.Summary), search) {
			filtered = append(filtered, sess)
		}
	}
	return filtered
}

// renderSessionsModal renders the sessions list modal
func (m *Model) renderSessionsModal() string {
	modalWidth := 75
	modalHeight := 20

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

	filteredSessions := m.getFilteredSessions()

	if len(m.sessionsList) == 0 {
		content := common.MutedStyle.Render("No sessions found. Press 'n' to create a new session.")
		footer := common.MutedStyle.Render("n new • esc close")
		return m.renderCenteredSessionsModal(modalWidth, modalHeight, title, content, footer)
	}

	// Search bar
	searchBar := ""
	if m.sessionsSearching {
		searchBar = "🔍 " + m.sessionsSearchInput.View()
	} else if m.sessionsSearch != "" {
		searchBar = common.MutedStyle.Render(fmt.Sprintf("🔍 Filter: %s (/ to edit, esc to clear)", m.sessionsSearch))
	} else {
		searchBar = common.MutedStyle.Render("/ to search")
	}

	// Build session list content
	var lines []string
	for i, sess := range filteredSessions {
		isSelected := i == m.sessionsCursor
		line := m.formatSessionLine(sess, isSelected)
		lines = append(lines, line)
	}

	// Calculate visible area (modal height minus header, search bar, footer, borders)
	visibleHeight := modalHeight - 8
	if visibleHeight < 3 {
		visibleHeight = 3
	}

	// Scroll window
	startIdx := 0
	if m.sessionsCursor >= visibleHeight {
		startIdx = m.sessionsCursor - visibleHeight + 1
	}
	endIdx := startIdx + visibleHeight
	if endIdx > len(lines) {
		endIdx = len(lines)
	}

	visibleLines := lines[startIdx:endIdx]
	content := strings.Join(visibleLines, "\n")

	// Show scroll indicators
	scrollInfo := ""
	if len(filteredSessions) > visibleHeight {
		scrollInfo = common.MutedStyle.Render(fmt.Sprintf(" [%d/%d]", m.sessionsCursor+1, len(filteredSessions)))
	}

	var footer string
	if m.sessionsSearching {
		footer = common.MutedStyle.Render("enter confirm • esc cancel search")
	} else {
		footer = common.MutedStyle.Render("↑↓ navigate • enter select • n new • / search • esc close") + scrollInfo
	}

	// Combine search bar with content
	fullContent := searchBar + "\n\n" + content

	return m.renderCenteredSessionsModal(modalWidth, modalHeight, title, fullContent, footer)
}

// formatSessionLine formats a single session entry for display
func (m *Model) formatSessionLine(sess infraSession.SessionInfo, selected bool) string {
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
	line := fmt.Sprintf("%-16s │ %-30s │ %s │ %5s", idDisplay, summary, dateStr, tokenStr)

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
func (m *Model) renderCenteredSessionsModal(width, height int, title, content, footer string) string {
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
	// If searching, handle search input
	if m.sessionsSearching {
		switch msg.String() {
		case "esc":
			m.sessionsSearching = false
			m.sessionsSearch = ""
			m.sessionsCursor = 0
			return m, nil
		case "enter":
			m.sessionsSearching = false
			m.sessionsSearch = m.sessionsSearchInput.Value()
			m.sessionsCursor = 0
			return m, nil
		default:
			var cmd tea.Cmd
			m.sessionsSearchInput, cmd = m.sessionsSearchInput.Update(msg)
			return m, cmd
		}
	}

	filteredSessions := m.getFilteredSessions()

	switch msg.String() {
	case "esc":
		if m.sessionsSearch != "" {
			// Clear search first
			m.sessionsSearch = ""
			m.sessionsCursor = 0
			return m, nil
		}
		m.activeModal = ModalNone
		return m, nil

	case "ctrl+c", "q":
		m.activeModal = ModalNone
		return m, nil

	case "up", "k":
		if m.sessionsCursor > 0 {
			m.sessionsCursor--
		}
		return m, nil

	case "down", "j":
		if m.sessionsCursor < len(filteredSessions)-1 {
			m.sessionsCursor++
		}
		return m, nil

	case "home", "g":
		m.sessionsCursor = 0
		return m, nil

	case "end", "G":
		if len(filteredSessions) > 0 {
			m.sessionsCursor = len(filteredSessions) - 1
		}
		return m, nil

	case "/":
		m.sessionsSearching = true
		m.sessionsSearchInput.SetValue(m.sessionsSearch)
		m.sessionsSearchInput.Focus()
		return m, nil

	case "enter":
		if len(filteredSessions) > 0 && m.sessionsCursor < len(filteredSessions) {
			selected := filteredSessions[m.sessionsCursor]
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

	// Capture current session info BEFORE async operation
	currentSessionID := ""
	var currentMsgs []ChatMessage
	if m.session != nil {
		currentSessionID = m.session.ID()
	}
	if m.msgs != nil && len(*m.msgs) > 0 {
		currentMsgs = make([]ChatMessage, len(*m.msgs))
		copy(currentMsgs, *m.msgs)
	}

	return m, func() tea.Msg {
		ctx := context.Background()

		// Generate summary for current session before switching (using captured values)
		if currentSessionID != "" && len(currentMsgs) >= 2 {
			m.generateAndSaveSummaryForSession(ctx, currentSessionID, currentMsgs)
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

	// Capture current session info BEFORE async operation
	currentSessionID := ""
	var currentMsgs []ChatMessage
	if m.session != nil {
		currentSessionID = m.session.ID()
	}
	if m.msgs != nil && len(*m.msgs) > 0 {
		currentMsgs = make([]ChatMessage, len(*m.msgs))
		copy(currentMsgs, *m.msgs)
	}

	return m, func() tea.Msg {
		ctx := context.Background()

		// Generate summary for current session before creating new (using captured values)
		if currentSessionID != "" && len(currentMsgs) >= 2 {
			m.generateAndSaveSummaryForSession(ctx, currentSessionID, currentMsgs)
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

// generateAndSaveSummaryForSession generates a summary for a specific session and saves it
func (m *Model) generateAndSaveSummaryForSession(ctx context.Context, sessionID string, msgs []ChatMessage) {
	if m.llm == nil || len(msgs) < 2 {
		return
	}

	// Build conversation text
	var sb strings.Builder
	for _, msg := range msgs {
		if msg.Role == "info" {
			continue
		}
		content := msg.Content
		if len(content) > 200 {
			content = content[:197] + "..."
		}
		fmt.Fprintf(&sb, "%s: %s\n", msg.Role, content)
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

	// Save summary to database (synchronously, not in goroutine)
	if summary != "" {
		if svc, ok := m.sessionService.(*infraSession.SQLiteService); ok {
			_ = svc.UpdateSummary(context.Background(), sessionID, summary)
		}
		// Notify UI to update header
		if m.program != nil {
			m.program.Send(sessionSummaryUpdatedMsg{sessionID: sessionID, summary: summary})
		}
	}
}
