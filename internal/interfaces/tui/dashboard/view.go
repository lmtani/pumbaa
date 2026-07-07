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

	if m.showHelp {
		return m.renderHelpModal()
	}

	if m.showError {
		return m.renderErrorModal()
	}

	if m.showDiff {
		return m.renderDiffModal()
	}

	sections := []string{m.renderHeader()}
	if m.filterBarVisible() {
		sections = append(sections, m.renderFilterBar())
	}
	sections = append(sections, m.renderContent(), m.renderFooter())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}
