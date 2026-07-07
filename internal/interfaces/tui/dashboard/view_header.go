package dashboard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

// renderHeader renders the single-line dashboard header bar.
func (m Model) renderHeader() string {
	brand := common.HeaderBrandStyle.Render("Pumbaa")
	breadcrumbs := common.RenderBreadcrumbs([]common.Screen{
		{Name: "Dashboard", Active: true},
	})

	// Connection / health status
	var status string
	switch {
	case m.loading:
		status = m.spinner.View() + " Loading..."
	case m.error != "":
		status = common.ErrorStyle.Render(common.IconFailed + " Error")
	default:
		status = common.SuccessStyle.Render(common.IconRunning + " Connected")
	}
	if m.healthStatus != nil {
		if m.healthStatus.OK {
			status += common.SuccessStyle.Render(" · Healthy")
		} else if m.healthStatus.Degraded {
			systemsStr := ""
			if len(m.healthStatus.UnhealthySystems) > 0 {
				systemsStr = " (" + strings.Join(m.healthStatus.UnhealthySystems, ", ") + ")"
			}
			status += lipgloss.NewStyle().
				Foreground(common.WarningColor).
				Render(" · " + common.IconWarning + " Degraded" + systemsStr)
		}
	}

	left := brand + " " + breadcrumbs + "  " + status

	// Right side: compare-base badge, update notice, workflow count, last refresh
	var right []string
	if m.compareBaseID != "" {
		base := m.compareBaseName
		if base == "" {
			base = truncateID(m.compareBaseID)
		}
		right = append(right, common.BadgeStyle.
			Foreground(common.BadgeFg).
			Background(common.BadgeWarnBg).
			Render("⇄ base: "+base))
	}
	if m.updateInfo != nil && m.updateInfo.UpdateAvailable {
		right = append(right, common.BadgeStyle.
			Foreground(common.BadgeFg).
			Background(common.BadgeDangerBg).
			Render(fmt.Sprintf("↑ Update: %s", m.updateInfo.Latest)))
	}
	if m.autoRefresh {
		right = append(right, common.BadgeStyle.
			Foreground(common.BadgeFg).
			Background(common.BadgeWarnBg).
			Render("⟳ AUTO 30s"))
	}
	right = append(right, common.BadgeStyle.
		Foreground(common.BadgeFg).
		Background(common.BadgeInfoBg).
		Render(fmt.Sprintf("%d workflows", m.totalCount)))
	if !m.lastRefresh.IsZero() {
		right = append(right, common.MutedStyle.Render(" "+m.lastRefresh.Format("15:04:05")))
	}

	return common.RenderHeaderBar(m.width, left, strings.Join(right, ""))
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
		common.TitleStyle.Render("Loading Debug View"),
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
