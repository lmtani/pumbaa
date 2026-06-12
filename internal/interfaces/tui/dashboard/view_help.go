package dashboard

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

// renderHelpModal renders the full keyboard reference overlay (opened with ?).
func (m Model) renderHelpModal() string {
	var content strings.Builder

	helpLine := func(keys, desc string) string {
		return "  " + common.KeyStyle.Render(common.PadRight(keys, 10)) + common.DescStyle.Render(desc) + "\n"
	}
	section := func(title string) string {
		return common.TitleStyle.Render(title) + "\n"
	}

	content.WriteString(section("Navigation"))
	content.WriteString(helpLine("↑↓", "Move selection"))
	content.WriteString(helpLine("pgup/pgdn", "Move 10 rows"))
	content.WriteString(helpLine("home/end", "First / last row"))
	content.WriteString(helpLine("enter", "Open debug view"))
	content.WriteString("\n")

	content.WriteString(section("Filtering"))
	content.WriteString(helpLine("/", "Filter by name (live)"))
	content.WriteString(helpLine("l", "Filter by label (live)"))
	content.WriteString(helpLine("s", "Cycle status filter"))
	content.WriteString(helpLine("u", "Go to workflow by UUID"))
	content.WriteString(helpLine("ctrl+x", "Clear all filters"))
	content.WriteString("\n")

	content.WriteString(section("Actions"))
	content.WriteString(helpLine("a", "Abort selected workflow"))
	content.WriteString(helpLine("L", "Edit labels"))
	content.WriteString(helpLine("r", "Refresh list"))
	content.WriteString("\n")

	content.WriteString(common.MutedStyle.Render("Press any key to close"))

	modal := common.ModalStyle.
		Width(44).
		Render(lipgloss.JoinVertical(lipgloss.Left,
			common.HeaderTitleStyle.Render("Dashboard Keys"),
			"",
			content.String(),
		))

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		modal,
		lipgloss.WithWhitespaceChars(" "),
	)
}
