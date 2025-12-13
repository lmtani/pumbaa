package debug

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderHeader() string {
	// Get just the icon without styling
	statusIcon := StatusIcon(m.metadata.Status)

	// Calculate total cost
	totalCost := m.calculateTotalCost()

	// Build status badge based on workflow status
	statusText := m.metadata.Status
	var statusBadge string
	switch m.metadata.Status {
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
	)

	return headerStyle.Width(m.width - 2).Render(header)
}

func (m Model) calculateTotalCost() float64 {
	var total float64
	m.calculateNodeCost(m.tree, &total)
	return total
}

func (m Model) calculateNodeCost(node *TreeNode, total *float64) {
	if node.CallData != nil && node.CallData.VMCostPerHour > 0 {
		// Calculate duration
		var duration float64
		if !node.CallData.VMStartTime.IsZero() && !node.CallData.VMEndTime.IsZero() {
			duration = node.CallData.VMEndTime.Sub(node.CallData.VMStartTime).Hours()
		}
		*total += node.CallData.VMCostPerHour * duration
	}
	for _, child := range node.Children {
		m.calculateNodeCost(child, total)
	}
}
