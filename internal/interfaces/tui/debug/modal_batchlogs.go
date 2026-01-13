package debug

import (
	tea "github.com/charmbracelet/bubbletea"
)

// renderBatchLogsModal renders the batch logs modal.
func (m Model) renderBatchLogsModal() string {
	// Modal title
	titleText := "📊 Google Batch Logs"
	if m.batchLogsHScrollOffset > 0 {
		titleText += " ◀"
	}
	title := titleStyle.Render(titleText)

	// Modal content
	viewportContent := m.batchLogsViewport.View()
	content := renderModalViewportContent(viewportContent, m.batchLogsViewport.Width, m.batchLogsLoading, m.batchLogsError)

	// Footer with instructions
	footer := m.batchLogsModalFooter()

	return m.renderStandardModal(title, content, footer)
}

// batchLogsModalFooter generates the footer for batch logs modal with scroll hint
func (m Model) batchLogsModalFooter() string {
	return m.modalFooterWithHints("↑↓ scroll", "←→ pan", "y copy", "esc close")
}

// handleBatchLogsModalKeys handles keyboard input in batch logs modal
func (m Model) handleBatchLogsModalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	viewportWidth := m.batchLogsViewport.Width
	actions := viewportModalActions{
		onClose: func(m *Model) {
			m.activeModal = ModalNone
			m.batchLogsContent = ""
			m.batchLogsRawContent = ""
			m.batchLogsError = ""
			m.batchLogsHScrollOffset = 0
		},
		onCopy: func(m *Model) tea.Cmd {
			if m.batchLogsRawContent != "" {
				return copyToClipboard(m.batchLogsRawContent, "batch logs")
			}
			return nil
		},
		onHome: func(m *Model) {
			m.batchLogsViewport.GotoTop()
			m.batchLogsHScrollOffset = 0
			truncatedContent := truncateLinesToWidth(m.batchLogsContent, viewportWidth)
			m.batchLogsViewport.SetContent(truncatedContent)
		},
		onLeft: func(m *Model) {
			if m.batchLogsHScrollOffset > 0 {
				m.batchLogsHScrollOffset -= modalHorizontalStep
				if m.batchLogsHScrollOffset < 0 {
					m.batchLogsHScrollOffset = 0
				}
				scrolledContent := applyHorizontalScroll(m.batchLogsContent, m.batchLogsHScrollOffset, viewportWidth)
				truncatedContent := truncateLinesToWidth(scrolledContent, viewportWidth)
				m.batchLogsViewport.SetContent(truncatedContent)
			}
		},
		onRight: func(m *Model) {
			m.batchLogsHScrollOffset += modalHorizontalStep
			scrolledContent := applyHorizontalScroll(m.batchLogsContent, m.batchLogsHScrollOffset, viewportWidth)
			truncatedContent := truncateLinesToWidth(scrolledContent, viewportWidth)
			m.batchLogsViewport.SetContent(truncatedContent)
		},
	}
	cmd, handled := m.handleViewportModalKeys(msg, &m.batchLogsViewport, actions)
	if handled {
		return m, cmd
	}
	return m, nil
}
