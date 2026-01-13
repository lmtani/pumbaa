package debug

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderHeader() string {
	// Get just the icon without styling
	statusIcon := StatusIcon(string(m.metadata.Status))

	// Use cached total cost
	totalCost := m.totalCost

	// Build status badge based on workflow status
	statusText := string(m.metadata.Status)
	var statusBadge string
	switch statusText {
	case "Succeeded", "Done":
		statusBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000000")).
			Background(lipgloss.Color("#00FF00")).
			Padding(0, 1).
			Render(statusIcon + " " + statusText)
	case "Failed":
		statusBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#FF0000")).
			Padding(0, 1).
			Render(statusIcon + " " + statusText)
	case "Running":
		statusBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000000")).
			Background(lipgloss.Color("#FFFF00")).
			Padding(0, 1).
			Render(statusIcon + " " + statusText)
	default:
		statusBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#888888")).
			Padding(0, 1).
			Render(statusIcon + " " + statusText)
	}

	// Duration badge
	durationBadge := ""
	if !m.metadata.Start.IsZero() {
		duration := m.metadata.End.Sub(m.metadata.Start)
		if m.metadata.End.IsZero() {
			duration = 0
		}
		durationBadge = durationBadgeStyle.Render("⏱ " + formatDuration(duration))
	}

	// Cost badge
	costBadge := ""
	if totalCost > 0 {
		costBadge = costBadgeStyle.Render(fmt.Sprintf("💰 $%.4f", totalCost))
	}

	// Search badge (active or filtered)
	searchBadge := ""
	if m.searchActive || m.searchQuery != "" {
		label := "SEARCH"
		if m.searchActive && m.searchQuery == "" {
			label = "SEARCH..."
		} else if m.searchQuery != "" {
			label = "SEARCH: " + truncate(m.searchQuery, 24)
		}
		searchBadge = searchBadgeStyle.Render(label)
	}

	// Workflow name and ID
	workflowName := headerTitleStyle.Render(m.metadata.Name)
	workflowID := mutedStyle.Render(" " + m.metadata.ID)

	// Combine all parts
	header := lipgloss.JoinHorizontal(
		lipgloss.Center,
		statusBadge,
		"  ",
		workflowName,
		workflowID,
		durationBadge,
		costBadge,
		searchBadge,
	)

	return headerStyle.Width(m.width - 2).Render(header)
}
