package debug

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

// statusDuration is the duration for temporary status messages
const statusDuration = 3 * time.Second

func (m Model) renderFooter() string {
	var parts []string

	// Loading indicator with elapsed time (takes priority). Elapsed time is
	// honest feedback; a synthetic progress bar would just guess.
	if m.isLoading && m.loadingMessage != "" {
		elapsed := time.Since(m.loadingStartTime)
		parts = append(parts, temporaryStatusStyle.Render(fmt.Sprintf("⏳ %s (%.1fs)", m.loadingMessage, elapsed.Seconds())))
		parts = append(parts, " • ")
	} else if m.statusMessage != "" {
		// Status message with decreasing progress bar
		timeRemaining := time.Until(m.statusMessageExpires)
		progressBar := ""

		if timeRemaining > 0 {
			percentage := int(timeRemaining.Seconds() * 100 / statusDuration.Seconds())
			if percentage > 100 {
				percentage = 100
			}
			barLength := (percentage * 20) / 100 // 20 chars max
			progressBar = " [" + strings.Repeat("━", barLength) + strings.Repeat("╌", 20-barLength) + "]"
		}

		parts = append(parts, temporaryStatusStyle.Render(m.statusMessage+progressBar))
		parts = append(parts, " • ")
	}

	if searchStatus := m.renderSearchStatus(); searchStatus != "" {
		parts = append(parts, searchStatus)
		parts = append(parts, " • ")
	}

	// Help shortcuts - only as many as fit, so the footer never wraps
	prefix := strings.Join(parts, "")
	hintBudget := m.width - 2 - lipgloss.Width(prefix)
	help := common.FitParts(hintBudget, "  ", m.footerHints())

	return common.HelpBarStyle.
		Width(m.width).
		Render(prefix + help)
}

func (m Model) renderSearchStatus() string {
	if m.searchQuery == "" && !m.searchActive {
		return ""
	}

	query := m.searchQuery
	if query == "" {
		query = "..."
	}

	matchInfo := ""
	if m.searchQuery != "" {
		matchInfo = fmt.Sprintf(" (%d matches)", len(m.searchMatches))
	}

	status := fmt.Sprintf("search %q%s", query, matchInfo)
	return fmt.Sprintf("%s %s", common.KeyStyle.Render("/"), common.DescStyle.Render(status))
}

func (m Model) footerHints() []string {
	hints := []string{
		renderFooterHint("↑↓", "navigate"),
	}

	// Quick actions reflect what 1-5/a actually do for the selected node,
	// instead of an opaque "1-6 actions".
	hints = append(hints, m.quickActionHints()...)

	hints = append(hints,
		renderFooterHint("tab", "switch"),
		renderFooterHint("E/C", "expand/collapse"),
		renderFooterHint("f", "failures"),
		renderFooterHint("w", "watch"),
		renderFooterHint("$", "cost"),
		renderFooterHint("/", "search"),
	)

	if m.searchQuery != "" || m.searchActive {
		hints = append(hints, renderFooterHint("n/N", "matches"))
		hints = append(hints, renderFooterHint("ctrl+x", "clear"))
	}

	if m.lastError != "" {
		hints = append(hints, renderFooterHint("e", "error details"))
	}

	hints = append(hints, renderFooterHint("?", "help"))
	hints = append(hints, renderFooterHint("esc", m.escHint()))

	return hints
}

// escHint describes what ESC does right now, so the footer never lies.
func (m Model) escHint() string {
	switch {
	case m.searchActive:
		return "exit search"
	case m.viewMode != ViewModeTree:
		return "tree view"
	case m.canGoBack:
		return "back"
	default:
		return "quit"
	}
}

// quickActionHints renders the quick actions available for the selected
// node, straight from the same table that dispatches the keys.
func (m Model) quickActionHints() []string {
	if m.cursor >= len(m.nodes) {
		return nil
	}

	actions := m.quickActionsFor(m.nodes[m.cursor])
	hints := make([]string, 0, len(actions))
	for _, action := range actions {
		if action.visible != nil && !action.visible(m) {
			continue
		}
		hints = append(hints, renderFooterHint(action.key, action.label))
	}
	return hints
}

func renderFooterHint(key, desc string) string {
	return fmt.Sprintf("%s %s", common.KeyStyle.Render(key), common.DescStyle.Render(desc))
}
