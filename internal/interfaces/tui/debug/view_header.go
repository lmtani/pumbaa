package debug

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"

	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

func (m Model) renderHeader() string {
	// Breadcrumbs - Debug is after Dashboard
	breadcrumbs := common.RenderBreadcrumbs([]common.Screen{
		{Name: "Dashboard", Active: false},
		{Name: "Debug", Active: true},
	})

	// Navigation hints
	navHints := common.RenderNavHints(true) // Can go back to dashboard
	// Get just the icon without styling
	statusIcon := common.StatusIcon(string(m.metadata.Status))

	// Use cached total cost
	totalCost := m.totalCost

	// Build status badge based on workflow status
	statusText := string(m.metadata.Status)
	statusBadge := common.StatusBadgeStyle(statusText).Render(statusIcon + " " + statusText)

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

	// First line: breadcrumbs and nav hints
	headerLine1 := lipgloss.JoinHorizontal(
		lipgloss.Top,
		breadcrumbs,
		"  ",
		navHints,
	)

	// Second line: status, workflow name, badges
	headerLine2 := lipgloss.JoinHorizontal(
		lipgloss.Center,
		statusBadge,
		"  ",
		workflowName,
		workflowID,
		durationBadge,
		costBadge,
		searchBadge,
	)

	headerContent := lipgloss.JoinVertical(lipgloss.Left, headerLine1, headerLine2)

	return headerStyle.Width(m.width - 2).Render(headerContent)
}
