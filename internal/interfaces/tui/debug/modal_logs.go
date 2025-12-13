package debug

import (
	"github.com/charmbracelet/lipgloss"
)

// renderLogModal renders the log modal.
func (m Model) renderLogModal() string {
	modalWidth := m.width - 6
	modalHeight := m.height - 4

	// Modal title
	title := titleStyle.Render("📄 " + m.logModalTitle)

	// Modal content
	var content string
	if m.logModalError != "" {
		content = errorStyle.Render("Error: " + m.logModalError)
	} else if m.logModalLoading {
		content = mutedStyle.Render("Loading...")
	} else {
		content = m.logModalViewport.View()
	}

	// Footer with instructions
	footer := m.modalFooter()

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
