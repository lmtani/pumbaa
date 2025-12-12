package debug

import (
	"strings"
	"time"
)

func (m Model) renderFooter() string {
	var footer string
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

		footer = temporaryStatusStyle.Render("⏱ " + m.statusMessage + progressBar)
	} else {
		footer = " ↑↓ navigate • tab switch • d details • c cmd • i inputs • o outputs • O options • t durations • ? help • q quit"
	}
	return helpBarStyle.Width(m.width - 2).Render(footer)
}
