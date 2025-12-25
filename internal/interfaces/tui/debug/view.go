package debug

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/lmtani/pumbaa/internal/domain/workflow/preemption"
	"github.com/lmtani/pumbaa/internal/infrastructure/cromwell"
)

// statusStyle returns a styled status icon
func statusStyle(status string) string {
	icon := StatusIcon(status)
	style := StatusStyle(status)
	return style.Render(icon)
}

// Panel styles
var (
	treePanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#444444")).
			Padding(0, 1)

	detailsPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#444444")).
				Padding(0, 1)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#874BFD"))
)

// View renders the TUI.
func (m Model) View() string {
	if m.isLoading {
		return m.renderLoading()
	}

	if m.showLogModal {
		return m.renderLogModal()
	}

	if m.showInputsModal {
		return m.renderInputsModal()
	}

	if m.showOutputsModal {
		return m.renderOutputsModal()
	}

	if m.showOptionsModal {
		return m.renderOptionsModal()
	}

	if m.showCallInputsModal {
		return m.renderCallInputsModal()
	}

	if m.showCallOutputsModal {
		return m.renderCallOutputsModal()
	}

	if m.showCallCommandModal {
		return m.renderCallCommandModal()
	}

	if m.showGlobalTimelineModal {
		return m.renderGlobalTimelineModal()
	}

	if m.showHelp {
		return m.renderHelp()
	}

	header := m.renderHeader()
	tree := m.renderTree()
	details := m.renderDetails()
	footer := m.renderFooter()

	mainContent := lipgloss.JoinHorizontal(lipgloss.Top, tree, details)

	return lipgloss.JoinVertical(lipgloss.Left, header, mainContent, footer)
}

func (m Model) renderHelp() string {
	return m.help.View(m.keys)
}

func (m Model) renderLoading() string {
	loadingBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7D56F4")).
		Padding(2, 4).
		Render(m.loadingSpinner.View() + "  " + m.loadingMessage)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		loadingBox,
	)
}

// countSubworkflows counts the number of subworkflows in the calls
func countSubworkflows(calls map[string][]CallDetails) int {
	count := 0
	for _, callList := range calls {
		for _, cd := range callList {
			if cd.SubWorkflowMetadata != nil {
				count++
			}
		}
	}
	return count
}

// renderPreemptionSummary renders a summary of preemption efficiency for a workflow
func (m Model) renderPreemptionSummary(node *TreeNode) string {
	var calls map[string][]CallDetails
	var workflowID, workflowName string

	// Get the calls for this workflow/subworkflow
	if node.Type == NodeTypeWorkflow {
		calls = m.metadata.Calls
		workflowID = m.metadata.ID
		workflowName = m.metadata.Name
	} else if node.Type == NodeTypeSubWorkflow && node.CallData != nil && node.CallData.SubWorkflowMetadata != nil {
		calls = node.CallData.SubWorkflowMetadata.Calls
		workflowID = node.CallData.SubWorkflowMetadata.ID
		workflowName = node.CallData.SubWorkflowMetadata.Name
	} else {
		return ""
	}

	var summary *preemption.WorkflowSummary
	if m.preemption != nil && node.Type == NodeTypeWorkflow {
		// If model was constructed with DebugInfo, use its precomputed summary for the workflow root
		summary = m.preemption
	} else {
		callData := cromwell.ConvertToPreemptionCallData(calls)
		summary = preemption.NewAnalyzer().AnalyzeWorkflow(workflowID, workflowName, callData)
	}
	subworkflowCount := countSubworkflows(calls)

	// Check if we have any preemptible tasks at this level
	if summary.PreemptibleTasks == 0 {
		var sb strings.Builder
		sb.WriteString(mutedStyle.Render("No preemptible tasks at this level\n"))
		if subworkflowCount > 0 {
			sb.WriteString("\n")
			infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#5599FF")).Italic(true)
			sb.WriteString(infoStyle.Render(fmt.Sprintf("ℹ This workflow has %d subworkflow(s).\n", subworkflowCount)))
			sb.WriteString(infoStyle.Render("  Navigate to each subworkflow to see its preemption stats.\n"))
		}
		return sb.String()
	}

	var sb strings.Builder

	sb.WriteString(titleStyle.Render("🔄 Preemption Summary") + "\n")
	sb.WriteString(mutedStyle.Render("  (this level only, excluding subworkflows)") + "\n\n")

	// Cost-weighted efficiency with visual gauge bar
	costEff := summary.CostEfficiency
	sb.WriteString(labelStyle.Render("Cost Efficiency: ") + "\n")
	sb.WriteString(renderPreemptionGauge(costEff, 25) + "\n\n")

	// Compact stats line: Preemptible | Attempts | Preemptions
	statsLine := fmt.Sprintf("%s %d/%d  │  %s %d  │  %s %d",
		labelStyle.Render("Preemptible:"),
		summary.PreemptibleTasks, summary.TotalTasks,
		labelStyle.Render("Attempts:"),
		summary.TotalAttempts,
		labelStyle.Render("Preemptions:"),
		summary.TotalPreemptions,
	)
	sb.WriteString(statsLine + "\n")

	// Cost breakdown with explanation
	sb.WriteString("\n")
	sb.WriteString(mutedStyle.Render("  Cost = CPU × Memory(GB) × Duration(h)") + "\n")
	if summary.TotalCost > 0.01 {
		sb.WriteString(labelStyle.Render("  Total Cost: ") +
			valueStyle.Render(fmt.Sprintf("%.2f %s", summary.TotalCost, summary.CostUnit)) + "\n")
		if summary.WastedCost > 0 {
			sb.WriteString(labelStyle.Render("  Wasted Cost: ") +
				lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6B6B")).Render(
					fmt.Sprintf("%.2f %s (%.0f%%)", summary.WastedCost, summary.CostUnit, (summary.WastedCost/summary.TotalCost)*100)) + "\n")
		}
	} else {
		sb.WriteString(labelStyle.Render("  Total Cost: ") +
			mutedStyle.Render("< 0.01 resource-hours") + "\n")
	}

	// Note about subworkflows
	if subworkflowCount > 0 {
		sb.WriteString("\n")
		infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#5599FF")).Italic(true)
		sb.WriteString(infoStyle.Render(fmt.Sprintf("  ℹ %d subworkflow(s) not included above.\n", subworkflowCount)))
		sb.WriteString(infoStyle.Render("    Navigate to each to see their stats.\n"))
	}

	// Show problematic tasks if any (sorted by wasted cost / impact)
	if len(summary.ProblematicTasks) > 0 {
		sb.WriteString("\n")
		warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFAA00")).Bold(true)
		sb.WriteString(warningStyle.Render("⚠ Tasks with High Preemption Impact:") + "\n")

		// Show up to 5 worst tasks (sorted by wasted cost)
		maxToShow := 5
		if len(summary.ProblematicTasks) < maxToShow {
			maxToShow = len(summary.ProblematicTasks)
		}

		for i := 0; i < maxToShow; i++ {
			task := summary.ProblematicTasks[i]

			// Format cost efficiency
			costEffStr := fmt.Sprintf("%.0f%%", task.CostEfficiency*100)
			var taskEffStyle string
			if task.CostEfficiency < 0.5 {
				taskEffStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6B6B")).Render(costEffStr)
			} else {
				taskEffStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFAA00")).Render(costEffStr)
			}

			// Format impact
			impactStr := ""
			if task.ImpactPercent > 0 {
				impactStr = fmt.Sprintf(" [%.0f%% of total waste]", task.ImpactPercent)
			}

			sb.WriteString(fmt.Sprintf("  • %s: %d preemptions, eff: %s, wasted: %.2f%s\n",
				valueStyle.Render(task.Name),
				task.TotalPreemptions,
				taskEffStyle,
				task.WastedCost,
				mutedStyle.Render(impactStr),
			))
		}

		if len(summary.ProblematicTasks) > maxToShow {
			sb.WriteString(mutedStyle.Render(fmt.Sprintf("  ... and %d more\n", len(summary.ProblematicTasks)-maxToShow)))
		}
	}

	return sb.String()
}

// renderPreemptionGauge creates a visual gauge bar for preemption efficiency
func renderPreemptionGauge(efficiency float64, width int) string {
	if efficiency < 0 {
		efficiency = 0
	}
	if efficiency > 1 {
		efficiency = 1
	}

	filled := int(efficiency * float64(width))
	empty := width - filled

	// Choose color and indicator based on efficiency level
	var barColor lipgloss.Color
	var indicator string
	if efficiency >= 0.8 {
		barColor = lipgloss.Color("#00FF00") // Green
		indicator = " ✓"
	} else if efficiency >= 0.5 {
		barColor = lipgloss.Color("#FFFF00") // Yellow
		indicator = " ⚠"
	} else {
		barColor = lipgloss.Color("#FF6B6B") // Red
		indicator = " ✗"
	}

	filledStyle := lipgloss.NewStyle().Foreground(barColor)
	emptyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#333333"))

	bar := "[" + filledStyle.Render(strings.Repeat("█", filled)) +
		emptyStyle.Render(strings.Repeat("░", empty)) + "]"

	percentStr := fmt.Sprintf(" %.0f%%", efficiency*100)
	return bar + lipgloss.NewStyle().Foreground(barColor).Bold(true).Render(percentStr+indicator)
}
