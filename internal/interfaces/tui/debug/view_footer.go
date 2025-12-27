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
		loadingMsg := common.RenderLoadingMessage(m.loadingMessage, m.loadingStartTime, loadingDuration)
		parts = append(parts, loadingMsg)
		parts = append(parts, " • ")
	} else if m.statusMessage != "" {
		// Status message with decreasing progress bar
		statusMsg := common.RenderStatusMessage(m.statusMessage, m.statusMessageExpires, statusDuration)
		parts = append(parts, statusMsg)
		parts = append(parts, " • ")
	}

	// Help shortcuts with consistent styling
	help := fmt.Sprintf(
		"%s %s  %s %s  %s %s  %s %s  %s %s  %s %s  %s %s",
		common.KeyStyle.Render("↑↓"),
		common.DescStyle.Render("navigate"),
		common.KeyStyle.Render("tab"),
		common.DescStyle.Render("switch"),
		common.KeyStyle.Render("d"),
		common.DescStyle.Render("details"),
		common.KeyStyle.Render("1-5"),
		common.DescStyle.Render("actions"),
		common.KeyStyle.Render("E/C"),
		common.DescStyle.Render("expand/collapse"),
		common.KeyStyle.Render("?"),
		common.DescStyle.Render("help"),
		common.KeyStyle.Render("q"),
		common.DescStyle.Render("quit"),
	)
	parts = append(parts, help)

	return common.HelpBarStyle.
		Width(m.width - 2).
		Render(strings.Join(parts, ""))
}
