package dashboard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

// renderContent renders the main content area (error, empty state, or table).
// The UUID prompt is the only filter that still replaces the content area;
// name/label filtering happens live in the inline bar above the table.
func (m Model) renderContent() string {
	if m.showFilter && m.filterType == "uuid" {
		return m.renderFilterInput()
	}

	if m.error != "" {
		// Determine error category and appropriate title/tips
		errorTitle, troubleshootingTips := categorizeErrorForDisplay(m.error)

		// Build a compact, helpful error display
		var errorContent strings.Builder
		errorContent.WriteString(common.ErrorStyle.Render(errorTitle) + "\n\n")

		errorContent.WriteString(common.MutedStyle.Render("Backend response:") + "\n")
		errorContent.WriteString(lipgloss.NewStyle().
			Foreground(common.ErrorSoftColor).
			Render(m.error) + "\n\n")

		errorContent.WriteString(common.MutedStyle.Render("Troubleshooting:") + "\n")
		for _, tip := range troubleshootingTips {
			errorContent.WriteString("  • " + tip + "\n")
		}

		errorBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(common.StatusFailed).
			Padding(1, 2).
			Width(maxInt(60, m.width/2)).
			Render(errorContent.String())

		return common.PanelStyle.
			Width(m.width - 2).
			Height(m.contentHeight()).
			Render(lipgloss.Place(m.width-4, m.contentHeight()-2, lipgloss.Center, lipgloss.Center, errorBox))
	}

	if len(m.workflows) == 0 && !m.loading {
		emptyMsg := common.MutedStyle.Render("No workflows found\n\nPress 'r' to refresh or '/' to filter")
		return common.PanelStyle.
			Width(m.width - 2).
			Height(m.contentHeight()).
			Render(lipgloss.Place(m.width-4, m.contentHeight()-2, lipgloss.Center, lipgloss.Center, emptyMsg))
	}

	return m.renderTable()
}

// renderFilterBar renders the single-line live filter bar above the table.
func (m Model) renderFilterBar() string {
	label := "Name"
	if m.filterType == "label" {
		label = "Label"
	}
	left := " " + common.KeyStyle.Render("/") + " " + common.LabelStyle.Render(label+":") + " " + m.filterInput.View()
	count := common.ValueStyle.Render(fmt.Sprintf("%d/%d", len(m.workflows), len(m.allWorkflows)))
	hint := common.MutedStyle.Render("enter apply on server · esc cancel")
	return common.RenderHeaderBar(m.width, left, count+"  "+hint+" ")
}

// renderFilterInput renders the filter input modal.
func (m Model) renderFilterInput() string {
	title := "Search by Name"
	switch m.filterType {
	case "label":
		title = "Search by Label"
	case "uuid":
		title = "Go to Workflow"
	}

	filterBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(common.PrimaryColor).
		Padding(1, 2).
		Width(50).
		Render(
			lipgloss.JoinVertical(lipgloss.Left,
				common.TitleStyle.Render(title),
				"",
				m.filterInput.View(),
				"",
				common.MutedStyle.Render("Enter to apply • Esc to cancel"),
			),
		)

	return common.PanelStyle.
		Width(m.width - 2).
		Height(m.contentHeight()).
		Render(lipgloss.Place(m.width-4, m.contentHeight()-2, lipgloss.Center, lipgloss.Center, filterBox))
}

// renderConfirmModal renders the abort confirmation modal.
func (m Model) renderConfirmModal() string {
	modalContent := lipgloss.JoinVertical(lipgloss.Center,
		common.TitleStyle.Render("⚠  Confirm Abort"),
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
