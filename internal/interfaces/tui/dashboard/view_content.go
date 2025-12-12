package dashboard

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

// renderContent renders the main content area (filter input, error, empty state, or table).
func (m Model) renderContent() string {
	if m.showFilter {
		return m.renderFilterInput()
	}

	if m.error != "" {
		// Parse error to make it more user-friendly
		errorMsg := m.error

		// Build a compact, helpful error display
		var errorContent strings.Builder
		errorContent.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF6B6B")).
			Bold(true).
			Render("⚠ Query Failed") + "\n\n")

		errorContent.WriteString(common.MutedStyle.Render("Backend response:") + "\n")
		errorContent.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF8E8E")).
			Render(errorMsg) + "\n\n")

		errorContent.WriteString(common.MutedStyle.Render("Troubleshooting:") + "\n")
		errorContent.WriteString("  • Check your filter values\n")
		errorContent.WriteString("  • Press " + common.KeyStyle.Render("ctrl+x") + " to clear all filters\n")

		errorBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#FF6B6B")).
			Padding(1, 2).
			Width(maxInt(60, m.width/2)).
			Render(errorContent.String())

		return common.PanelStyle.
			Width(m.width - 2).
			Height(m.height - 8).
			Render(lipgloss.Place(m.width-4, m.height-10, lipgloss.Center, lipgloss.Center, errorBox))
	}

	if len(m.workflows) == 0 && !m.loading {
		emptyMsg := common.MutedStyle.Render("No workflows found\n\nPress 'r' to refresh or '/' to filter")
		return common.PanelStyle.
			Width(m.width - 2).
			Height(m.height - 8).
			Render(lipgloss.Place(m.width-4, m.height-10, lipgloss.Center, lipgloss.Center, emptyMsg))
	}

	return m.renderTable()
}

// renderFilterInput renders the filter input modal.
func (m Model) renderFilterInput() string {
	filterBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(common.PrimaryColor).
		Padding(1, 2).
		Width(50).
		Render(
			lipgloss.JoinVertical(lipgloss.Left,
				common.TitleStyle.Render("Filter Workflows"),
				"",
				m.filterInput.View(),
				"",
				common.MutedStyle.Render("Enter to apply • Esc to cancel"),
			),
		)

	return common.PanelStyle.
		Width(m.width - 2).
		Height(m.height - 8).
		Render(lipgloss.Place(m.width-4, m.height-10, lipgloss.Center, lipgloss.Center, filterBox))
}

// renderConfirmModal renders the abort confirmation modal.
func (m Model) renderConfirmModal() string {
	modalContent := lipgloss.JoinVertical(lipgloss.Center,
		common.TitleStyle.Render("⚠️  Confirm Abort"),
		"",
		"Are you sure you want to abort workflow",
		common.MutedStyle.Render(truncateID(m.confirmID)),
		"",
		lipgloss.JoinHorizontal(lipgloss.Center,
			common.KeyStyle.Render("y")+" Yes  ",
			common.KeyStyle.Render("n")+" No",
		),
	)

	modal := common.ModalStyle.
		Width(50).
		Render(modalContent)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		modal,
		lipgloss.WithWhitespaceChars(" "),
	)
}
