package dashboard

import (
	"github.com/charmbracelet/lipgloss"
)

// View implements tea.Model.
func (m Model) View() string {
	if m.loadingDebug {
		return m.renderDebugLoadingScreen()
	}

	if m.showConfirm {
		return m.renderConfirmModal()
	}

	if m.showLabelsModal {
		return m.renderLabelsModal()
	}

	header := m.renderHeader()
	content := m.renderContent()
	footer := m.renderFooter()

	return lipgloss.JoinVertical(lipgloss.Left, header, content, footer)
}
