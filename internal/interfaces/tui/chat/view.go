package chat

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

// View renders the chat interface.
func (m *Model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	header := m.renderHeader()
	content := m.renderContent()
	input := m.renderInput()
	footer := m.renderFooter()

	return lipgloss.JoinVertical(lipgloss.Left, header, content, input, footer)
}

func (m Model) renderHeader() string {
	title := common.HeaderTitleStyle.Render("🐗 Pumbaa Chat")

	// LLM provider badge
	llmBadge := ""
	if m.llm != nil {
		llmStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000000")).
			Background(lipgloss.Color("#87CEEB")).
			Padding(0, 1)
		llmBadge = llmStyle.Render("🤖 " + m.llm.Name())
	}

	sessionInfo := ""
	if m.session != nil {
		sessionInfo = common.MutedStyle.Render(fmt.Sprintf("Session: %s", m.session.ID()))
	}

	// Layout: Title | LLM Badge | Session (right aligned)
	leftContent := lipgloss.JoinHorizontal(lipgloss.Center, title, "  ", llmBadge)
	headerContent := lipgloss.JoinHorizontal(lipgloss.Center, leftContent, "  ", sessionInfo)

	return common.HeaderStyle.
		Width(m.width - 2).
		Render(headerContent)
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
			loadingText = fmt.Sprintf("%s 🔧 Executing: %s", m.spinner.View(), m.toolNotification)
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

func (m Model) renderFooter() string {
	var help string
	if m.focusMode == FocusMessages {
		help = fmt.Sprintf(
			"%s %s  %s %s  %s %s  %s %s",
			common.KeyStyle.Render("↑↓"),
			common.DescStyle.Render("navigate"),
			common.KeyStyle.Render("y"),
			common.DescStyle.Render("copy"),
			common.KeyStyle.Render("tab"),
			common.DescStyle.Render("type"),
			common.KeyStyle.Render("esc"),
			common.DescStyle.Render("exit"),
		)
	} else {
		help = fmt.Sprintf(
			"%s %s  %s %s  %s %s  %s %s",
			common.KeyStyle.Render("ctrl+d"),
			common.DescStyle.Render("send"),
			common.KeyStyle.Render("↑↓"),
			common.DescStyle.Render("scroll"),
			common.KeyStyle.Render("tab"),
			common.DescStyle.Render("navigate msgs"),
			common.KeyStyle.Render("esc"),
			common.DescStyle.Render("exit"),
		)
	}

	// Show status message if present
	if m.statusMessage != "" {
		help = common.SuccessStyle.Render(m.statusMessage) + "  " + help
	}

	return common.HelpBarStyle.
		Width(m.width - 2).
		Render(help)
}

func (m Model) renderMessages() string {
	if m.msgs == nil || len(*m.msgs) == 0 {
		return common.MutedStyle.Render("Welcome to Pumbaa Chat!\n\nType your message and press Ctrl+D to send.")
	}

	var sb strings.Builder
	maxWidth := m.width - 8
	if maxWidth <= 0 {
		maxWidth = 80
	}

	for i, msg := range *m.msgs {
		var roleStyle lipgloss.Style
		var roleName string

		switch msg.Role {
		case "user":
			roleStyle = userStyle
			roleName = "You"
		case "agent":
			roleStyle = agentStyle
			roleName = "Pumbaa"
		default:
			roleStyle = errorStyle
			roleName = "Error"
		}

		// Highlight if selected
		isSelected := m.focusMode == FocusMessages && i == m.selectedMsg

		// Render role with selection indicator
		if isSelected {
			sb.WriteString(selectedMsgStyle.Render("▶ "+roleStyle.Render(roleName)) + "\n")
		} else {
			sb.WriteString(roleStyle.Render(roleName) + "\n")
		}

		// Render content
		var content string
		if msg.Role == "agent" && msg.Rendered != "" {
			content = msg.Rendered
		} else {
			content = messageStyle.Render(wrapText(msg.Content, maxWidth))
		}

		if isSelected {
			sb.WriteString(selectedMsgStyle.Render(content))
		} else {
			sb.WriteString(content)
		}
		sb.WriteString("\n\n")
	}

	return sb.String()
}

// scrollToSelectedMsg scrolls the viewport to center the selected message.
func (m *Model) scrollToSelectedMsg() {
	if m.msgs == nil || m.selectedMsg < 0 || m.selectedMsg >= len(*m.msgs) {
		return
	}

	// Calculate approximate position based on message index
	// Each message is roughly: role line (1) + content lines + spacing (2)
	// We estimate ~6 lines per message on average for a good approximation
	linesPerMsg := 6
	targetLine := m.selectedMsg * linesPerMsg

	// Center the message in the viewport
	viewportHeight := m.viewport.Height
	centeredOffset := targetLine - (viewportHeight / 2)

	// Clamp to valid range
	if centeredOffset < 0 {
		centeredOffset = 0
	}

	m.viewport.SetYOffset(centeredOffset)
}
