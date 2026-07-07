package dashboard

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

// statusDuration is the duration for temporary status messages.
const statusDuration = 3 * time.Second

// temporaryStatusStyle is the style for auto-expiring notification messages.
var temporaryStatusStyle = lipgloss.NewStyle().
	Foreground(common.StatusRunning).
	Bold(true)

// renderFooter renders the status bar and help footer.
func (m Model) renderFooter() string {
	var parts []string

	// Temporary status message with animated progress bar
	if m.statusMsg != "" && !m.statusMessageExpires.IsZero() {
		timeRemaining := time.Until(m.statusMessageExpires)
		progressBar := ""
		if timeRemaining > 0 {
			percentage := int(timeRemaining.Seconds() * 100 / statusDuration.Seconds())
			if percentage > 100 {
				percentage = 100
			}
			barLength := (percentage * 20) / 100
			progressBar = " [" + strings.Repeat("━", barLength) + strings.Repeat("╌", 20-barLength) + "]"
		}
		parts = append(parts, temporaryStatusStyle.Render(m.statusMsg+progressBar))
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
			Foreground(common.BadgeFg).
			Background(common.BadgeWarnBg).
			Render(fmt.Sprintf("Status: %s", strings.Join(statusNames, "/"))))
		parts = append(parts, " ")
		hasFilters = true
	}

	if m.activeFilters.Name != "" {
		parts = append(parts, common.BadgeStyle.
			Foreground(common.BadgeFg).
			Background(common.BadgeInfoBg).
			Render(fmt.Sprintf("Name: %s", m.activeFilters.Name)))
		parts = append(parts, " ")
		hasFilters = true
	}

	if m.activeFilters.Label != "" {
		parts = append(parts, common.BadgeStyle.
			Foreground(common.BadgeFg).
			Background(common.BadgeSuccessBg).
			Render(fmt.Sprintf("Label: %s", m.activeFilters.Label)))
		parts = append(parts, " ")
		hasFilters = true
	}

	if hasFilters {
		parts = append(parts, common.KeyStyle.Render("ctrl+x")+common.DescStyle.Render(" clear")+"  ")
	}

	// Help - only as many hints as fit on one line, so the footer never wraps.
	// The full reference lives in the ? overlay.
	hints := []string{
		renderHint("↑↓", "navigate"),
		renderHint("enter", "debug"),
		renderHint("/", "filter"),
		renderHint("a", "abort"),
		renderHint("c", "compare"),
	}
	if m.LastError != nil {
		hints = append(hints, renderHint("e", "error details"))
	}
	hints = append(hints,
		renderHint("?", "help"),
		renderHint("esc", "quit"),
		renderHint("l", "label filter"),
		renderHint("u", "go to UUID"),
		renderHint("L", "edit labels"),
		renderHint("s", "status"),
		renderHint("r", "refresh"),
		renderHint("w", "auto-refresh"),
	)
	prefix := strings.Join(parts, "")
	hintBudget := m.width - 2 - lipgloss.Width(prefix)
	help := common.FitParts(hintBudget, "  ", hints)

	return common.HelpBarStyle.
		Width(m.width).
		Render(prefix + help)
}

// renderHint formats a single key binding hint for the footer help bar.
func renderHint(key, desc string) string {
	return common.KeyStyle.Render(key) + " " + common.DescStyle.Render(desc)
}
