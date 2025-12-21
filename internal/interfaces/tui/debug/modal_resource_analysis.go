package debug

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderResourceAnalysisModal renders the resource efficiency analysis modal
func (m Model) renderResourceAnalysisModal() string {
	modalWidth := m.width - 10
	modalHeight := m.height - 8

	var content strings.Builder

	if m.resourceError != "" {
		content.WriteString(errorStyle.Render("Error: "+m.resourceError) + "\n")
	} else if m.resourceReport != nil {
		report := m.resourceReport

		// Header with duration and data points
		content.WriteString(mutedStyle.Render(fmt.Sprintf("⏱ Duration: %s  📊 Data points: %d",
			formatDuration(report.Duration), report.DataPoints)) + "\n\n")

		// CPU Section
		content.WriteString(titleStyle.Render("💻 CPU") + "\n")
		content.WriteString(renderGaugeBar(report.CPU.Efficiency, 25) + "\n")
		content.WriteString(fmt.Sprintf("Peak: %.0f%%  Avg: %.0f%%  Efficiency: %.0f%%\n\n",
			report.CPU.Peak, report.CPU.Avg, report.CPU.Efficiency*100))

		// Memory Section
		content.WriteString(titleStyle.Render("🧠 Memory") + "\n")
		content.WriteString(renderGaugeBar(report.Mem.Efficiency, 25) + "\n")
		content.WriteString(fmt.Sprintf("Peak: %.0fMB / %.0fMB  Efficiency: %.0f%%\n\n",
			report.Mem.Peak, report.Mem.Total, report.Mem.Efficiency*100))

		// Disk Section
		content.WriteString(titleStyle.Render("💾 Disk") + "\n")
		content.WriteString(renderGaugeBar(report.Disk.Efficiency, 25) + "\n")
		content.WriteString(fmt.Sprintf("Peak: %.1fGB / %.1fGB  Efficiency: %.0f%%\n\n",
			report.Disk.Peak, report.Disk.Total, report.Disk.Efficiency*100))

		// Recommendations
		if len(report.Recommendations) > 0 {
			content.WriteString(titleStyle.Render("💡 Recommendations") + "\n")
			for _, rec := range report.Recommendations {
				content.WriteString("• " + rec + "\n")
			}
			content.WriteString("\n")
		}

		// Efficiency calculation explanation
		content.WriteString(mutedStyle.Render("─── How efficiency is calculated ───") + "\n")
		content.WriteString(mutedStyle.Render("• CPU: Average usage / 100%") + "\n")
		content.WriteString(mutedStyle.Render("• Memory & Disk: Peak usage / Total allocated") + "\n")
		content.WriteString(mutedStyle.Render("Low efficiency = over-provisioned resources") + "\n")
	} else {
		content.WriteString(mutedStyle.Render("Loading..."))
	}

	// Set content in viewport
	m.resourceViewport.SetContent(content.String())

	// Footer
	footer := mutedStyle.Render("↑↓ scroll • esc close")

	// Build modal with proper structure
	viewportView := m.resourceViewport.View()

	// Title
	title := titleStyle.Render("📊 Resource Analysis")

	// Join content vertically
	modalContent := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		viewportView,
		"",
		footer,
	)

	// Apply modal style
	styledModal := modalStyle.
		Width(modalWidth).
		Height(modalHeight).
		Render(modalContent)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		styledModal,
	)
}

// renderGaugeBar creates a visual gauge bar
func renderGaugeBar(efficiency float64, width int) string {
	if efficiency < 0 {
		efficiency = 0
	}
	if efficiency > 1 {
		efficiency = 1
	}

	filled := int(efficiency * float64(width))
	empty := width - filled

	// Choose color based on efficiency level
	var barColor lipgloss.Color
	if efficiency >= 0.7 {
		barColor = lipgloss.Color("#00FF00") // Green for high efficiency
	} else if efficiency >= 0.4 {
		barColor = lipgloss.Color("#FFFF00") // Yellow for medium
	} else {
		barColor = lipgloss.Color("#FF6B6B") // Red for low efficiency
	}

	filledStyle := lipgloss.NewStyle().Foreground(barColor)
	emptyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#333333"))

	bar := filledStyle.Render(strings.Repeat("█", filled)) +
		emptyStyle.Render(strings.Repeat("░", empty))

	percentStr := fmt.Sprintf(" %.0f%%", efficiency*100)
	return bar + lipgloss.NewStyle().Foreground(barColor).Bold(true).Render(percentStr)
}
