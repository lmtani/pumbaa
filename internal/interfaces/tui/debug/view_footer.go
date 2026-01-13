package debug

import (
	"fmt"
	"strings"
	"time"

	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

// loadingDuration is the expected duration for loading operations (for progress bar)
const loadingDuration = 5 * time.Second

// statusDuration is the duration for temporary status messages
const statusDuration = 3 * time.Second

func (m Model) renderFooter() string {
	var parts []string

	// Loading indicator with increasing progress bar (takes priority)
	if m.isLoading && m.loadingMessage != "" {
		elapsed := time.Since(m.loadingStartTime)
		percentage := int(elapsed.Seconds() * 100 / loadingDuration.Seconds())
		if percentage > 100 {
			percentage = 100
		}

		barLength := (percentage * 20) / 100 // 20 chars max
		progressBar := " [" + strings.Repeat("━", barLength) + strings.Repeat("╌", 20-barLength) + "]"

		loadingStyle := temporaryStatusStyle
		parts = append(parts, loadingStyle.Render("⏳ "+m.loadingMessage+progressBar))
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

	// Help shortcuts with consistent styling
	parts = append(parts, m.renderFooterHints())

	return common.HelpBarStyle.
		Width(m.width - 2).
		Render(strings.Join(parts, ""))
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

func (m Model) renderFooterHints() string {
	hints := []string{
		renderFooterHint("↑↓", "navigate"),
		renderFooterHint("tab", "switch"),
		renderFooterHint("d", "details"),
		renderFooterHint("1-6", "actions"),
		renderFooterHint("E/C", "expand/collapse"),
		renderFooterHint("/", "search"),
	}

	if m.searchQuery != "" || m.searchActive {
		hints = append(hints, renderFooterHint("ctrl+x", "clear"))
	}

	hints = append(hints, renderFooterHint("?", "help"), renderFooterHint("q", "quit"))
	return strings.Join(hints, "  ")
}

func renderFooterHint(key, desc string) string {
	return fmt.Sprintf("%s %s", common.KeyStyle.Render(key), common.DescStyle.Render(desc))
}
