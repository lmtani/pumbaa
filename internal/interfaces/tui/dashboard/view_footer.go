package dashboard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

// renderFooter renders the status bar and help footer.
func (m Model) renderFooter() string {
	var parts []string

	// Status message
	if m.statusMsg != "" {
		// Determine notification type based on message prefix
		notifyType := common.NotifyInfo
		msg := m.statusMsg
		if strings.HasPrefix(m.statusMsg, "✓") {
			notifyType = common.NotifySuccess
			msg = strings.TrimPrefix(m.statusMsg, "✓ ")
		} else if strings.HasPrefix(m.statusMsg, "✗") {
			notifyType = common.NotifyError
			msg = strings.TrimPrefix(m.statusMsg, "✗ ")
		}
		parts = append(parts, common.RenderNotification(msg, notifyType))
		parts = append(parts, " • ")
	}

	// Filter indicators with clear option
	hasFilters := false
	if len(m.activeFilters.Status) > 0 {
		statusNames := make([]string, len(m.activeFilters.Status))
		for i, s := range m.activeFilters.Status {
			statusNames[i] = string(s)
		}
		parts = append(parts, common.BadgeStyle.
			Foreground(lipgloss.Color("#000000")).
			Background(lipgloss.Color("#FFD700")).
			Render(fmt.Sprintf("Status: %s", strings.Join(statusNames, "/"))))
		parts = append(parts, " ")
		hasFilters = true
	}

	if m.activeFilters.Name != "" {
		parts = append(parts, common.BadgeStyle.
			Foreground(lipgloss.Color("#000000")).
			Background(lipgloss.Color("#87CEEB")).
			Render(fmt.Sprintf("Name: %s", m.activeFilters.Name)))
		parts = append(parts, " ")
		hasFilters = true
	}

	if m.activeFilters.Label != "" {
		parts = append(parts, common.BadgeStyle.
			Foreground(lipgloss.Color("#000000")).
			Background(lipgloss.Color("#98FB98")).
			Render(fmt.Sprintf("Label: %s", m.activeFilters.Label)))
		parts = append(parts, " ")
		hasFilters = true
	}

	if hasFilters {
		parts = append(parts, common.KeyStyle.Render("ctrl+x")+common.DescStyle.Render(" clear")+"  ")
	}

	// Help
	help := fmt.Sprintf(
		"%s %s  %s %s  %s %s  %s %s  %s %s  %s %s  %s %s  %s %s  %s %s",
		common.KeyStyle.Render("↑↓"),
		common.DescStyle.Render("navigate"),
		common.KeyStyle.Render("enter"),
		common.DescStyle.Render("debug"),
		common.KeyStyle.Render("a"),
		common.DescStyle.Render("abort"),
		common.KeyStyle.Render("/"),
		common.DescStyle.Render("name"),
		common.KeyStyle.Render("l"),
		common.DescStyle.Render("label"),
		common.KeyStyle.Render("L"),
		common.DescStyle.Render("labels"),
		common.KeyStyle.Render("s"),
		common.DescStyle.Render("status"),
		common.KeyStyle.Render("r"),
		common.DescStyle.Render("refresh"),
		common.KeyStyle.Render("q"),
		common.DescStyle.Render("quit"),
	)
	parts = append(parts, help)

	return common.HelpBarStyle.
		Width(m.width - 2).
		Render(strings.Join(parts, ""))
}
