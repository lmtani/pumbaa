package chat

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

// View rendering: header, transcript, input area and footer.
func (m *Model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	// Check for active modal first
	if modalView, ok := m.renderActiveModal(); ok {
		return modalView
	}

	header := m.renderHeader()
	content := m.renderContent()
	input := m.renderInput()
	footer := m.renderFooter()

	return lipgloss.JoinVertical(lipgloss.Left, header, content, input, footer)
}

// headerHeight is the fixed height of the chat header: the bar plus the
// session line. Keeping it constant lets the viewport size stay stable even
// when the session summary arrives asynchronously.
const headerHeight = 2

func (m Model) renderHeader() string {
	brand := common.HeaderBrandStyle.Render("Pumbaa")

	// Breadcrumb reflects how the chat was opened
	var screens []common.Screen
	if m.standalone {
		screens = []common.Screen{{Name: "Chat", Active: true}}
	} else {
		screens = []common.Screen{
			{Name: "Dashboard", Active: false},
			{Name: "Debug", Active: false},
			{Name: "Chat", Active: true},
		}
	}
	breadcrumbs := common.RenderBreadcrumbs(screens)

	left := brand + " " + breadcrumbs
	if m.contextLabel != "" {
		left += " " + contextBadgeStyle.Render(m.contextLabel)
	}

	// Right side: LLM provider and token usage
	var right []string
	if m.llm != nil {
		right = append(right, llmBadgeStyle.Render(m.llm.Name()))
	}
	if m.inputTokens > 0 || m.outputTokens > 0 {
		right = append(right, tokenBadgeStyle.Render(fmt.Sprintf("%s↑ %s↓", formatTokenCount(m.inputTokens), formatTokenCount(m.outputTokens))))
	}

	bar := common.RenderHeaderBar(m.width, left, strings.Join(right, " "))

	// Session line: summary when available, otherwise the session ID
	sessionLine := common.MutedStyle.Render("No session")
	if m.sessionSummary != "" {
		sessionLine = sessionSummaryStyle.Render(common.Truncate(m.sessionSummary, m.width-2))
	} else if m.session != nil {
		sessionID := m.session.ID()
		if len(sessionID) > 12 {
			sessionID = sessionID[:12] + "…"
		}
		sessionLine = common.MutedStyle.Render("Session: " + sessionID)
	}

	return lipgloss.JoinVertical(lipgloss.Left, bar, " "+sessionLine)
}

func (m Model) renderContent() string {
	return common.PanelStyle.
		Width(m.width - 2).
		Height(m.viewport.Height).
		Render(m.viewport.View())
}

func (m Model) renderInput() string {
	var inputBox string
	if m.loading {
		var loadingText string
		if m.toolNotification != "" {
			loadingText = fmt.Sprintf("%s Executing: %s", m.spinner.View(), m.toolNotification)
		} else {
			loadingText = fmt.Sprintf("%s Thinking...", m.spinner.View())
		}
		inputBox = inputStyle.
			Width(m.width - 4).
			Render(loadingText)
	} else {
		inputBox = focusedInputStyle.
			Width(m.width - 4).
			Render(m.textarea.View())
	}
	return inputBox
}

// scrollToSelectedMsg scrolls the viewport so the selected message is
// visible, using the real rendered line offsets tracked by renderMessages.
func (m *Model) scrollToSelectedMsg() {
	if m.msgs == nil || m.selectedMsg < 0 || m.selectedMsg >= len(*m.msgs) || m.selectedMsg >= len(m.msgOffsets) {
		return
	}

	// Place the message a third from the top: it reads naturally and leaves
	// room for the following context.
	offset := m.msgOffsets[m.selectedMsg] - m.viewport.Height/3
	if offset < 0 {
		offset = 0
	}
	m.viewport.SetYOffset(offset)
}

func (m Model) renderFooter() string {
	// Determine if ESC should show "back" or "quit"
	escAction := "back"
	if m.standalone {
		escAction = "quit"
	}

	hint := func(key, desc string) string {
		return common.KeyStyle.Render(key) + " " + common.DescStyle.Render(desc)
	}
	var hints []string
	switch {
	case m.loading:
		hints = []string{
			hint("esc", "cancel"),
			hint("↑↓", "scroll"),
		}
	case m.focusMode == FocusMessages:
		hints = []string{
			hint("↑↓", "navigate"),
			hint("pgup/pgdn", "page"),
			hint("g/G", "first/last"),
			hint("y", "copy"),
			hint("tab", "type"),
			hint("esc", escAction),
		}
	default:
		hints = []string{
			hint("enter", "send"),
			hint("ctrl+j", "newline"),
		}
		if m.resumableID != "" {
			hints = append(hints, hint("ctrl+r", "resume previous"))
		}
		hints = append(hints,
			hint("ctrl+s", "sessions"),
			hint("↑↓", "scroll"),
			hint("tab", "navigate msgs"),
			hint("esc", "messages"),
		)
	}

	// Show status message if present
	prefix := ""
	if m.statusMessage != "" {
		prefix = common.SuccessStyle.Render(m.statusMessage) + "  "
	}

	// Only as many hints as fit on one line, so the footer never wraps
	help := common.FitParts(m.width-2-lipgloss.Width(prefix), "  ", hints)

	return common.HelpBarStyle.
		Width(m.width).
		Render(prefix + help)
}

func (m *Model) renderMessages() string {
	hasMsgs := m.msgs != nil && len(*m.msgs) > 0
	if !hasMsgs && m.streamingText == "" {
		m.msgOffsets = nil
		return common.MutedStyle.Render("Welcome to Pumbaa Chat! 🐗\n\nType your message and press Enter to send (ctrl+j for a new line).")
	}

	var sb strings.Builder
	maxWidth := m.width - 8
	if maxWidth <= 0 {
		maxWidth = 80
	}

	// Track each message's rendered line offset for selection scrolling
	lineCount := 0
	offsets := make([]int, 0, 16)

	if hasMsgs {
		for i, msg := range *m.msgs {
			block := m.renderMessageBlock(i, msg, maxWidth)
			offsets = append(offsets, lineCount)
			sb.WriteString(block)
			lineCount += strings.Count(block, "\n")
		}
	}
	m.msgOffsets = offsets

	// Live response being streamed: plain text with a cursor block; the
	// final ResponseMsg replaces it with the markdown-rendered message.
	if m.loading && m.streamingText != "" {
		sb.WriteString(agentStyle.Render("Pumbaa") + "\n")
		sb.WriteString(messageStyle.Render(wrapText(m.streamingText, maxWidth) + " ▌"))
		sb.WriteString("\n\n")
	}

	return sb.String()
}

// renderMessageBlock renders a single transcript message, including its
// trailing spacing, so renderMessages can measure real line offsets.
func (m *Model) renderMessageBlock(i int, msg ChatMessage, maxWidth int) string {
	isSelected := m.focusMode == FocusMessages && i == m.selectedMsg
	selectedStyle := lipgloss.NewStyle().
		Background(common.SubtleColor).
		Padding(0, 1)

	// Compact one-line roles: tool records and notices have no header
	switch msg.Role {
	case "tool":
		line := common.MutedStyle.Render("🔧 " + msg.Content)
		if isSelected {
			line = selectedStyle.Render("▶ " + line)
		}
		return line + "\n\n"
	case "notice":
		line := common.MutedStyle.Italic(true).Render("· " + msg.Content)
		if isSelected {
			line = selectedStyle.Render("▶ " + line)
		}
		return line + "\n\n"
	}

	var roleStyle lipgloss.Style
	var roleName string
	contentStyle := messageStyle

	switch msg.Role {
	case "user":
		roleStyle = userStyle
		roleName = "You"
	case "agent":
		roleStyle = agentStyle
		roleName = "Pumbaa"
	case "info":
		roleStyle = infoStyle
		roleName = "Context"
		contentStyle = infoMessageStyle
	default:
		roleStyle = errorStyle
		roleName = "Error"
	}

	var sb strings.Builder

	// Render role with selection indicator
	if isSelected {
		sb.WriteString(selectedStyle.Render("▶ "+roleStyle.Render(roleName)) + "\n")
	} else {
		sb.WriteString(roleStyle.Render(roleName) + "\n")
	}

	// Render content
	var content string
	if msg.Role == "agent" && msg.Rendered != "" {
		content = msg.Rendered
	} else {
		content = contentStyle.Render(wrapText(msg.Content, maxWidth))
	}

	if isSelected {
		sb.WriteString(selectedStyle.Render(content))
	} else {
		sb.WriteString(content)
	}
	sb.WriteString("\n\n")

	return sb.String()
}
