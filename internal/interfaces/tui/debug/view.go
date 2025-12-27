package debug

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	workflowDomain "github.com/lmtani/pumbaa/internal/domain/workflow"
)

// statusStyle returns a styled status icon
func statusStyle(status string) string {
	icon := StatusIcon(status)
	style := StatusStyle(status)
	return style.Render(icon)
}

// View renders the current model state.
func (m Model) View() string {
	if m.isLoading && m.metadata == nil {
		return m.renderLoading()
	}

	// Check for modal states first
	if m.showHelp {
		return m.renderHelpOverlay()
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
	if m.showGlobalTimelineModal {
		return m.renderGlobalTimelineModal()
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

	// Main layout: tree + details
	treePanel := m.renderTree()
	detailsPanel := m.renderDetails()

	layout := lipgloss.JoinHorizontal(lipgloss.Top, treePanel, detailsPanel)

	header := m.renderHeader()
	footer := m.renderFooter()

	return lipgloss.JoinVertical(lipgloss.Left, header, layout, footer)
}

func (m Model) renderLoading() string {
	loadingBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7D56F4")).
		Padding(2, 4).
		Render(m.loadingSpinner.View() + " " + m.loadingMessage)

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
	var wf *WorkflowMetadata
	var workflowID, workflowName string

	// Get the workflow for this node
	if node.Type == NodeTypeWorkflow {
		wf = m.metadata
		workflowID = m.metadata.ID
		workflowName = m.metadata.Name
	} else if node.Type == NodeTypeSubWorkflow && node.CallData != nil && node.CallData.SubWorkflowMetadata != nil {
		wf = node.CallData.SubWorkflowMetadata
		workflowID = node.CallData.SubWorkflowMetadata.ID
		workflowName = node.CallData.SubWorkflowMetadata.Name
	} else {
		return ""
	}

	var summary *workflowDomain.PreemptionSummary
	if m.preemption != nil && node.Type == NodeTypeWorkflow {
		// If model was constructed with DebugInfo, use its precomputed summary for the workflow root
		summary = m.preemption
	} else {
		// Analyze preemption for this specific workflow/subworkflow - DDD pattern
		// Create a temporary workflow to analyze
		tempWf := &workflowDomain.Workflow{
			ID:    workflowID,
			Name:  workflowName,
			Calls: wf.Calls,
		}
		summary = tempWf.CalculatePreemptionSummary()
	}
	subworkflowCount := countSubworkflows(wf.Calls)

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
	sb.WriteString(statsLine + "\n\n")

	// Problematic tasks list (if any)
	if len(summary.ProblematicTasks) > 0 {
		sb.WriteString(titleStyle.Render("⚠ Problematic Tasks") + "\n")
		sb.WriteString(mutedStyle.Render("  (highest cost impact first)") + "\n\n")

		for i, task := range summary.ProblematicTasks {
			if i >= 3 {
				sb.WriteString(mutedStyle.Render(fmt.Sprintf("  ... and %d more\n", len(summary.ProblematicTasks)-3)))
				break
			}
			// Format: TaskName: 5 shards, 12 attempts, 7 preemptions (58% efficiency)
			taskLine := fmt.Sprintf("  %s: %d shards, %d attempts, %d preemptions",
				valueStyle.Render(task.Name),
				task.ShardCount,
				task.TotalAttempts,
				task.TotalPreemptions,
			)
			effPercent := task.CostEfficiency * 100
			var effColor lipgloss.Color
			if effPercent >= 80 {
				effColor = lipgloss.Color("#00FF00")
			} else if effPercent >= 50 {
				effColor = lipgloss.Color("#FFFF00")
			} else {
				effColor = lipgloss.Color("#FF6B6B")
			}
			effStyle := lipgloss.NewStyle().Foreground(effColor)
			sb.WriteString(taskLine + " " + effStyle.Render(fmt.Sprintf("(%.0f%% eff)", effPercent)) + "\n")
		}
	}

	// Hint about subworkflows if present
	if subworkflowCount > 0 {
		sb.WriteString("\n")
		infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#5599FF")).Italic(true)
		sb.WriteString(infoStyle.Render(fmt.Sprintf("ℹ This workflow has %d subworkflow(s).\n", subworkflowCount)))
		sb.WriteString(infoStyle.Render("  Navigate to each subworkflow to see its preemption stats.\n"))
	}

	return sb.String()
}

// renderPreemptionGauge renders a visual gauge bar for efficiency
func renderPreemptionGauge(efficiency float64, width int) string {
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
	if efficiency >= 0.8 {
		barColor = lipgloss.Color("#00FF00") // Green for high efficiency
	} else if efficiency >= 0.5 {
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
