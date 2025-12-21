package debug

import (
	"fmt"
	"strings"
	"time"

	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

func (m Model) renderFooter() string {
	var parts []string

	// Status message (if any)
	if m.statusMessage != "" {
		// Calculate time remaining for visual feedback
		timeRemaining := time.Until(m.statusMessageExpires)
		progressBar := ""

		if timeRemaining > 0 {
			// Show a progress bar that fades out
			percentage := int(timeRemaining.Seconds() * 100 / 3) // 3 seconds total
			if percentage > 100 {
				percentage = 100
			}
			barLength := (percentage * 30) / 100 // 30 chars max
			progressBar = " [" + strings.Repeat("━", barLength) + strings.Repeat("╌", 30-barLength) + "]"
		}

		parts = append(parts, temporaryStatusStyle.Render("⏱ "+m.statusMessage+progressBar))
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
