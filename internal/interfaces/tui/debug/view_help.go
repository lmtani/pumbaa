package debug

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

// renderHelpOverlay renders a contextual help overlay.
func (m Model) renderHelpOverlay() string {
	width := minInt(60, m.width-10)
	height := minInt(38, m.height-6)

	var content strings.Builder

	// Title
	title := common.TitleStyle.Render("? Help")
	content.WriteString(title + "\n\n")

	// Navigation section
	content.WriteString(helpSectionTitle("Navigation") + "\n")
	content.WriteString(helpLine("↑↓ j/k", "Move up/down"))
	content.WriteString(helpLine("←→ h/l", "Collapse/Expand or pan"))
	content.WriteString(helpLine("Tab", "Switch panel focus"))
	content.WriteString(helpLine("Enter", "Toggle expand or open log"))
	content.WriteString(helpLine("g / G", "Go to first / last item"))
	content.WriteString(helpLine("PgUp/Dn", "Page up/down"))
	content.WriteString("\n")

	// Global actions section
	content.WriteString(helpSectionTitle("Actions") + "\n")
	content.WriteString(helpLine("E / C", "Expand / Collapse all"))
	content.WriteString(helpLine("d", "Return to details view"))
	content.WriteString(helpLine("y", "Copy to clipboard"))
	content.WriteString(helpLine("Esc", "Close modal / back"))
	content.WriteString(helpLine("q", "Quit"))
	content.WriteString("\n")

	// Quick actions section header
	content.WriteString(helpSectionTitle("Quick Actions (1-5)") + "\n")
	content.WriteString(common.MutedStyle.Render("Actions depend on node type") + "\n\n")

	// Workflow actions
	wfStyle := lipgloss.NewStyle().Foreground(common.PrimaryColor)
	content.WriteString(wfStyle.Render(common.IconWorkflow+" Workflow / Subworkflow") + "\n")
	content.WriteString(helpLine("1", "Inputs"))
	content.WriteString(helpLine("2", "Outputs"))
	content.WriteString(helpLine("3", "Options"))
	content.WriteString(helpLine("4", "Timeline"))
	content.WriteString(helpLine("5", "Workflow log"))
	content.WriteString("\n")

	// Task actions
	taskStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#4CAF50"))
	content.WriteString(taskStyle.Render(common.IconTask+" Task / Shard") + "\n")
	content.WriteString(helpLine("1", "Inputs"))
	content.WriteString(helpLine("2", "Outputs"))
	content.WriteString(helpLine("3", "Command"))
	content.WriteString(helpLine("4", "Logs (inline)"))
	content.WriteString(helpLine("5", "Efficiency (inline)"))
	content.WriteString("\n")

	// In Modals section
	content.WriteString(helpSectionTitle("In Modals") + "\n")
	content.WriteString(helpLine("↑↓", "Scroll content"))
	content.WriteString(helpLine("←→", "Pan horizontally"))
	content.WriteString(helpLine("y", "Copy content"))
	content.WriteString(helpLine("Esc", "Close modal"))

	// Footer
	content.WriteString("\n" + common.MutedStyle.Render("Press ? or Esc to close"))

	// Create modal box
	modalStyle := common.ModalStyle.
		Width(width).
		Height(height)

	modal := modalStyle.Render(content.String())

	// Center the modal
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		modal,
	)
}

// helpSectionTitle formats a section title for help.
func helpSectionTitle(title string) string {
	return common.TitleStyle.Render(title)
}

// helpLine formats a help line with key and description.
func helpLine(key, desc string) string {
	return "  " + common.KeyStyle.Width(12).Render(key) + " " + common.MutedStyle.Render(desc) + "\n"
}

// minInt returns the minimum of two integers.
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
