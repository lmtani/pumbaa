package debug

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// renderBatchLogsModal renders the batch logs modal.
func (m Model) renderBatchLogsModal() string {
	modalWidth := m.width - 6
	modalHeight := m.height - 4

	// Modal title
	titleText := "📊 Google Batch Logs"
	if m.batchLogsHScrollOffset > 0 {
		titleText += " ◀"
	}
	title := titleStyle.Render(titleText)

	// Modal content
	var content string
	if m.batchLogsError != "" {
		content = errorStyle.Render("Error: " + m.batchLogsError)
	} else if m.batchLogsLoading {
		content = mutedStyle.Render("Loading...")
	} else {
		// Get viewport content and truncate lines to prevent wrap
		viewportContent := m.batchLogsViewport.View()
		content = truncateLinesToWidth(viewportContent, m.batchLogsViewport.Width)
	}

	// Footer with instructions
	footer := m.batchLogsModalFooter()

	// Build modal box
	modalContent := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		content,
		"",
		footer,
	)

	modal := modalStyle.
		Width(modalWidth).
		Height(modalHeight).
		Render(modalContent)

	// Center the modal
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		modal,
	)
}

// batchLogsModalFooter generates the footer for batch logs modal with scroll hint
func (m Model) batchLogsModalFooter() string {
	baseFooter := "↑↓ scroll • ←→ pan • y copy • esc close"
	if m.statusMessage != "" {
		return mutedStyle.Render(baseFooter) + "  " + temporaryStatusStyle.Render(m.statusMessage)
	}
	return mutedStyle.Render(baseFooter)
}

// handleBatchLogsModalKeys handles keyboard input in batch logs modal
func (m Model) handleBatchLogsModalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	viewportWidth := m.batchLogsViewport.Width

	switch {
	case key.Matches(msg, m.keys.Escape), key.Matches(msg, m.keys.Quit):
		m.showBatchLogsModal = false
		m.batchLogsContent = ""
		m.batchLogsRawContent = ""
		m.batchLogsError = ""
		m.batchLogsHScrollOffset = 0

	case key.Matches(msg, m.keys.Copy):
		if m.batchLogsRawContent != "" {
			return m, copyToClipboard(m.batchLogsRawContent, "batch logs")
		}

	case key.Matches(msg, m.keys.Up):
		m.batchLogsViewport.ScrollUp(1)

	case key.Matches(msg, m.keys.Down):
		m.batchLogsViewport.ScrollDown(1)

	case key.Matches(msg, m.keys.Left):
		if m.batchLogsHScrollOffset > 0 {
			m.batchLogsHScrollOffset -= 10
			if m.batchLogsHScrollOffset < 0 {
				m.batchLogsHScrollOffset = 0
			}
			scrolledContent := applyHorizontalScroll(m.batchLogsContent, m.batchLogsHScrollOffset, viewportWidth)
			truncatedContent := truncateLinesToWidth(scrolledContent, viewportWidth)
			m.batchLogsViewport.SetContent(truncatedContent)
		}

	case key.Matches(msg, m.keys.Right):
		m.batchLogsHScrollOffset += 10
		scrolledContent := applyHorizontalScroll(m.batchLogsContent, m.batchLogsHScrollOffset, viewportWidth)
		truncatedContent := truncateLinesToWidth(scrolledContent, viewportWidth)
		m.batchLogsViewport.SetContent(truncatedContent)

	case key.Matches(msg, m.keys.PageUp):
		m.batchLogsViewport.PageUp()

	case key.Matches(msg, m.keys.PageDown):
		m.batchLogsViewport.PageDown()

	case key.Matches(msg, m.keys.Home):
		m.batchLogsViewport.GotoTop()
		m.batchLogsHScrollOffset = 0
		truncatedContent := truncateLinesToWidth(m.batchLogsContent, viewportWidth)
		m.batchLogsViewport.SetContent(truncatedContent)

	case key.Matches(msg, m.keys.End):
		m.batchLogsViewport.GotoBottom()
	}

	return m, nil
}

// formatBatchLogsForDisplay formats log entries for display in the TUI.
// Format: "[timestamp] [SEVERITY] message"
// Returns colored output suitable for viewport.
func formatBatchLogsForDisplay(entries []interface{}, maxMessageLen int) string {
	var sb strings.Builder

	for _, e := range entries {
		entry, ok := e.(map[string]interface{})
		if !ok {
			continue
		}

		// Extract fields (assuming they're already formatted from use case)
		timestamp := entry["timestamp"]
		severity := entry["severity"]
		message := entry["message"]

		// Color by severity
		severityStr := fmt.Sprintf("%v", severity)
		var coloredSeverity string
		switch severityStr {
		case "ERROR", "CRITICAL":
			coloredSeverity = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FF6B6B")).
				Bold(true).
				Render(severityStr)
		case "WARNING":
			coloredSeverity = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFF00")).
				Render(severityStr)
		case "DEBUG":
			coloredSeverity = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#808080")).
				Render(severityStr)
		default:
			coloredSeverity = severityStr
		}

		// Truncate message if too long
		msgStr := fmt.Sprintf("%v", message)
		if len(msgStr) > maxMessageLen && maxMessageLen > 0 {
			msgStr = msgStr[:maxMessageLen-3] + "..."
		}

		// Format line: "[timestamp] [SEVERITY] message"
		line := fmt.Sprintf("[%v] [%s] %s\n", timestamp, coloredSeverity, msgStr)
		sb.WriteString(line)
	}

	return sb.String()
}
