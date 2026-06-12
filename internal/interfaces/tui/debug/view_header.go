package debug

import (
	"fmt"

	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

func (m Model) renderHeader() string {
	brand := common.HeaderBrandStyle.Render("Pumbaa")
	breadcrumbs := common.RenderBreadcrumbs([]common.Screen{
		{Name: "Dashboard", Active: false},
		{Name: "Debug", Active: true},
	})

	// Status badge
	statusText := string(m.metadata.Status)
	statusIcon := common.StatusIcon(statusText)
	statusBadge := common.StatusBadgeStyle(statusText).Render(statusIcon + " " + statusText)

	// Workflow name (truncated to keep the bar on one line) and short ID
	workflowName := headerTitleStyle.Render(common.Truncate(m.metadata.Name, 40))
	shortID := m.metadata.ID
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}
	workflowID := mutedStyle.Render(shortID)

	left := brand + " " + breadcrumbs + "  " + statusBadge + " " + workflowName + " " + workflowID

	// Right side: duration, cost and search badges
	right := ""
	if !m.metadata.Start.IsZero() {
		duration := m.metadata.End.Sub(m.metadata.Start)
		if m.metadata.End.IsZero() {
			duration = 0
		}
		right += durationBadgeStyle.Render(formatDuration(duration))
	}
	if m.totalCost > 0 {
		right += costBadgeStyle.Render(fmt.Sprintf("$%.4f", m.totalCost))
	}
	if m.searchActive || m.searchQuery != "" {
		label := "SEARCH"
		if m.searchActive && m.searchQuery == "" {
			label = "SEARCH..."
		} else if m.searchQuery != "" {
			label = "SEARCH: " + truncate(m.searchQuery, 24)
		}
		right += searchBadgeStyle.Render(label)
	}

	return common.RenderHeaderBar(m.width, left, right)
}
