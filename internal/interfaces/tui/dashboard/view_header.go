package dashboard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

// renderHeader renders the dashboard header with status badges and information.
func (m Model) renderHeader() string {
	// Title
	title := common.HeaderTitleStyle.Render("Cromwell Dashboard")

	// Status badges
	var badges []string

	// Connection status
	if m.loading {
		badges = append(badges, m.spinner.View()+" Loading...")
	} else if m.error != "" {
		badges = append(badges, common.ErrorStyle.Render(common.IconFailed+" Error"))
	} else {
		badges = append(badges, common.SuccessStyle.Render(common.IconRunning+" Connected"))
	}

	// Server health status badge
	if m.healthStatus != nil {
		if m.healthStatus.OK {
			badges = append(badges, lipgloss.NewStyle().
				Foreground(lipgloss.Color("#00FF00")).
				Render(common.IconRunning+" Healthy"))
		} else if m.healthStatus.Degraded {
			// Show which systems are unhealthy
			systemsStr := ""
			if len(m.healthStatus.UnhealthySystems) > 0 {
				systemsStr = " (" + strings.Join(m.healthStatus.UnhealthySystems, ", ") + ")"
			}
			badges = append(badges, lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFAA00")).
				Render(common.IconWarning+" Degraded"+systemsStr))
		}
	}

	// Workflow count
	countBadge := common.BadgeStyle.
		Foreground(lipgloss.Color("#000000")).
		Background(lipgloss.Color("#87CEEB")).
		Render(fmt.Sprintf("%d workflows", m.totalCount))
	badges = append(badges, countBadge)

	// Active filter indicator
	if len(m.activeFilters.Status) > 0 || m.activeFilters.Name != "" {
		filterBadge := common.BadgeStyle.
			Foreground(lipgloss.Color("#000000")).
			Background(lipgloss.Color("#FFD700")).
			Render("Filtered")
		badges = append(badges, filterBadge)
	}

	// Last refresh
	if !m.lastRefresh.IsZero() {
		refreshBadge := common.MutedStyle.Render(
			fmt.Sprintf("Updated %s", m.lastRefresh.Format("15:04:05")),
		)
		badges = append(badges, refreshBadge)
	}

	headerContent := lipgloss.JoinHorizontal(lipgloss.Center, title, "  ", strings.Join(badges, " "))

	return common.HeaderStyle.
		Width(m.width - 2).
		Render(headerContent)
}

// renderDebugLoadingScreen renders the loading screen when fetching debug metadata.
func (m Model) renderDebugLoadingScreen() string {
	// Find the workflow name
	workflowName := "workflow"
	for _, wf := range m.workflows {
		if wf.ID == m.loadingDebugID {
			workflowName = wf.Name
			break
		}
	}

	loadingContent := lipgloss.JoinVertical(lipgloss.Center,
		"",
		common.TitleStyle.Render("🔍 Loading Debug View"),
		"",
		m.spinner.View()+"  Fetching metadata...",
		"",
		common.MutedStyle.Render(workflowName),
		common.MutedStyle.Render(truncateID(m.loadingDebugID)),
		"",
	)

	loadingBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(common.PrimaryColor).
		Padding(2, 4).
		Render(loadingContent)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		loadingBox,
	)
}
