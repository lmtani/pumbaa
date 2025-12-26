package debug

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

func (m Model) renderDetails() string {
	style := detailsPanelStyle.Width(m.detailsWidth).Height(m.height - 8)
	if m.focus == FocusDetails {
		style = style.BorderForeground(lipgloss.Color("#7D56F4"))
	}

	title := m.getDetailsTitle()
	titleRendered := titleStyle.Render(title)

	content := m.detailViewport.View()

	return style.Render(lipgloss.JoinVertical(lipgloss.Left, titleRendered, "", content))
}

func (m Model) getDetailsTitle() string {
	switch m.viewMode {
	case ViewModeCommand:
		return "📜 Command"
	case ViewModeLogs:
		return "📋 Logs"
	case ViewModeInputs:
		return "📥 Inputs"
	case ViewModeOutputs:
		return "📤 Outputs"
	default:
		return "📊 Details"
	}
}

func (m Model) renderDetailsContent(node *TreeNode) string {
	var sb strings.Builder

	// Node type badge
	sb.WriteString(m.getNodeTypeBadge(node) + "\n")

	// Breadcrumb navigation at the very top
	sb.WriteString(m.renderBreadcrumb(node) + "\n")

	// Action bar is visible for all node types (except scatter)
	isScatter := node.Type == NodeTypeCall && len(node.Children) > 0
	if !isScatter {
		sb.WriteString(m.renderActionBar(node))
		sb.WriteString("\n\n")
	}

	// Content based on view mode
	switch m.viewMode {
	case ViewModeCommand:
		sb.WriteString(m.renderCommand(node))
	case ViewModeLogs:
		sb.WriteString(m.renderLogs(node))
	case ViewModeInputs:
		sb.WriteString(m.renderInputs(node))
	case ViewModeOutputs:
		sb.WriteString(m.renderOutputs(node))
	case ViewModeMonitor:
		sb.WriteString(m.renderMonitorContent())
	default:
		sb.WriteString(m.renderBasicDetailsBody(node))
	}

	return sb.String()
}

func (m Model) renderBasicDetailsBody(node *TreeNode) string {
	var sb strings.Builder

	// Status line (minimal, since badge already shows type+name)
	sb.WriteString(labelStyle.Render("Status: ") + statusStyle(node.Status) + " " + valueStyle.Render(node.Status) + "\n")
	if node.SubWorkflowID != "" {
		sb.WriteString(labelStyle.Render("SubWorkflow ID: ") + mutedStyle.Render(node.SubWorkflowID) + "\n")
	}
	sb.WriteString("\n")

	// Scatter summary for Call nodes with shards
	if node.Type == NodeTypeCall && len(node.Children) > 0 {
		sb.WriteString("\n")
		sb.WriteString(m.renderScatterSummary(node))
		sb.WriteString("\n" + sectionSeparator(35) + "\n")
	}

	// Call-specific details
	if node.CallData != nil {
		cd := node.CallData

		// Show task-level failures if present
		if len(cd.Failures) > 0 {
			sb.WriteString("\n")
			sb.WriteString(m.renderTaskFailures(cd.Failures))
		}

		// Timing
		if !cd.Start.IsZero() || !cd.End.IsZero() || !cd.VMStartTime.IsZero() || !cd.VMEndTime.IsZero() {
			sb.WriteString("\n")
			sb.WriteString(titleStyle.Render("⏱ Timing") + "\n")
			if !cd.Start.IsZero() {
				sb.WriteString(labelStyle.Render("Start: ") + valueStyle.Render(cd.Start.Format("15:04:05")) + "\n")
			}
			if !cd.End.IsZero() {
				sb.WriteString(labelStyle.Render("End: ") + valueStyle.Render(cd.End.Format("15:04:05")) + "\n")
				if !cd.Start.IsZero() {
					duration := cd.End.Sub(cd.Start)
					sb.WriteString(labelStyle.Render("Duration: ") + valueStyle.Render(formatDuration(duration)) + "\n")
				}
			}
			if !cd.VMStartTime.IsZero() {
				sb.WriteString(labelStyle.Render("VM Start: ") + valueStyle.Render(cd.VMStartTime.Format("15:04:05")) + "\n")
			}
			if !cd.VMEndTime.IsZero() {
				sb.WriteString(labelStyle.Render("VM End: ") + valueStyle.Render(cd.VMEndTime.Format("15:04:05")) + "\n")
			}
			sb.WriteString(sectionSeparator(35) + "\n")
		}

		// Resources - only show if has data
		if cd.CPU != "" || cd.Memory != "" || cd.Disk != "" || cd.Preemptible != "" {
			sb.WriteString("\n")
			sb.WriteString(titleStyle.Render("💻 Resources") + "\n")
			if cd.CPU != "" {
				sb.WriteString(labelStyle.Render("CPU: ") + valueStyle.Render(cd.CPU) + "\n")
			}
			if cd.Memory != "" {
				sb.WriteString(labelStyle.Render("Memory: ") + valueStyle.Render(cd.Memory) + "\n")
			}
			if cd.Disk != "" {
				sb.WriteString(labelStyle.Render("Disk: ") + valueStyle.Render(cd.Disk) + "\n")
			}
			if cd.Preemptible != "" {
				sb.WriteString(labelStyle.Render("Preemptible: ") + valueStyle.Render(cd.Preemptible) + "\n")
			}
		}

		// Docker - only show if has data
		if cd.DockerImage != "" || cd.DockerSize != "" {
			sb.WriteString("\n")
			sb.WriteString(titleStyle.Render("🐳 Docker") + "\n")
			if cd.DockerImage != "" {
				sb.WriteString(labelStyle.Render("Image:") + " " + mutedStyle.Render("(y to copy)") + "\n")
				sb.WriteString(formatDockerImage(cd.DockerImage))
			}
			if cd.DockerSize != "" {
				sb.WriteString(labelStyle.Render("Size: ") + valueStyle.Render(cd.DockerSize) + "\n")
			}
		}

		// Cache - only show if has meaningful data
		if cd.CacheHit || cd.CacheResult != "" {
			sb.WriteString("\n")
			sb.WriteString(titleStyle.Render("📦 Cache") + "\n")
			cacheStatus := "Miss"
			if cd.CacheHit {
				cacheStatus = "Hit"
			}
			sb.WriteString(labelStyle.Render("Status: ") + valueStyle.Render(cacheStatus) + "\n")
			if cd.CacheResult != "" {
				sb.WriteString(labelStyle.Render("Result: ") + valueStyle.Render(cd.CacheResult) + "\n")
			}
		}

		// Cost
		if cd.VMCostPerHour > 0 {
			sb.WriteString("\n")
			sb.WriteString(titleStyle.Render("💰 Cost") + "\n")
			sb.WriteString(labelStyle.Render("VM Cost/Hour: ") + valueStyle.Render(fmt.Sprintf("$%.4f", cd.VMCostPerHour)) + "\n")
		}
	} else {
		// For workflow/subworkflow nodes without CallData
		// Show workflow root and log paths
		if node.Type == NodeTypeWorkflow || node.Type == NodeTypeSubWorkflow {
			var metadata *WorkflowMetadata
			if node.Type == NodeTypeWorkflow {
				metadata = m.metadata
			}

			if metadata != nil {
				if metadata.WorkflowRoot != "" || metadata.WorkflowLog != "" {
					sb.WriteString("\n")
					sb.WriteString(titleStyle.Render("📁 Workflow Paths") + "\n")
					if metadata.WorkflowRoot != "" {
						sb.WriteString(labelStyle.Render("Root:") + "\n")
						sb.WriteString(pathStyle.Render(truncatePath(metadata.WorkflowRoot, m.detailsWidth-8)) + "\n")
					}
					if metadata.WorkflowLog != "" {
						sb.WriteString(labelStyle.Render("Log:") + " " + mutedStyle.Render("(w to view)") + "\n")
						sb.WriteString(pathStyle.Render(truncatePath(metadata.WorkflowLog, m.detailsWidth-8)) + "\n")
					}
				}
			}
		}

		// Show workflow-level failures if this is the root workflow node
		if node.Type == NodeTypeWorkflow && len(m.metadata.Failures) > 0 {
			sb.WriteString("\n")
			sb.WriteString(m.renderFailures())
		}

		// Show preemption summary for workflow nodes
		if node.Type == NodeTypeWorkflow || node.Type == NodeTypeSubWorkflow {
			sb.WriteString("\n")
			sb.WriteString(m.renderPreemptionSummary(node))
		}
	}

	return sb.String()
}

func (m Model) renderCommand(node *TreeNode) string {
	if node.CallData == nil || node.CallData.CommandLine == "" {
		return mutedStyle.Render("No command available")
	}
	// Aplicar syntax highlighting como Bash
	highlighted := common.Highlight(node.CallData.CommandLine, common.ProfileShell, m.detailsWidth-8)
	return highlighted
}

func (m Model) renderLogs(node *TreeNode) string {
	if node.CallData == nil {
		return mutedStyle.Render("No logs available")
	}

	var sb strings.Builder
	cd := node.CallData

	// Show selection indicator (always show when in log view mode)
	stdoutPrefix := "  "
	stderrPrefix := "  "
	monitoringPrefix := "  "
	switch m.logCursor {
	case 0:
		stdoutPrefix = "▶ "
	case 1:
		stderrPrefix = "▶ "
	case 2:
		monitoringPrefix = "▶ "
	}

	sb.WriteString(stdoutPrefix + labelStyle.Render("stdout: ") + "\n")
	if cd.Stdout != "" {
		sb.WriteString("  " + pathStyle.Render(truncatePath(cd.Stdout, m.detailsWidth-8)) + "\n\n")
	} else {
		sb.WriteString("  " + mutedStyle.Render("(not available)") + "\n\n")
	}

	sb.WriteString(stderrPrefix + labelStyle.Render("stderr: ") + "\n")
	if cd.Stderr != "" {
		sb.WriteString("  " + pathStyle.Render(truncatePath(cd.Stderr, m.detailsWidth-8)) + "\n\n")
	} else {
		sb.WriteString("  " + mutedStyle.Render("(not available)") + "\n\n")
	}

	sb.WriteString(monitoringPrefix + labelStyle.Render("monitoring: ") + "\n")
	if cd.MonitoringLog != "" {
		sb.WriteString("  " + pathStyle.Render(truncatePath(cd.MonitoringLog, m.detailsWidth-8)) + "\n\n")
	} else {
		sb.WriteString("  " + mutedStyle.Render("(not available)") + "\n\n")
	}

	if m.focus == FocusDetails {
		sb.WriteString(mutedStyle.Render("  Press Enter to view the selected log"))
	}

	return sb.String()
}

func (m Model) renderInputs(node *TreeNode) string {
	if node.CallData == nil || len(node.CallData.Inputs) == 0 {
		return mutedStyle.Render("No inputs available")
	}

	var sb strings.Builder
	for k, v := range node.CallData.Inputs {
		sb.WriteString(labelStyle.Render(k+": ") + "\n")
		sb.WriteString(valueStyle.Render(fmt.Sprintf("  %v", v)) + "\n\n")
	}
	return sb.String()
}

func (m Model) renderOutputs(node *TreeNode) string {
	if node.CallData == nil || len(node.CallData.Outputs) == 0 {
		return mutedStyle.Render("No outputs available")
	}

	var sb strings.Builder
	for k, v := range node.CallData.Outputs {
		sb.WriteString(labelStyle.Render(k+": ") + "\n")
		sb.WriteString(pathStyle.Render(fmt.Sprintf("  %v", v)) + "\n\n")
	}
	return sb.String()
}

// getNodeTypeBadge returns a clean header with node type and name
func (m Model) getNodeTypeBadge(node *TreeNode) string {
	var icon, label string
	var color lipgloss.Color

	// Determine badge based on node type
	switch node.Type {
	case NodeTypeWorkflow:
		icon = common.IconWorkflow
		label = "workflow"
		color = lipgloss.Color("#9C27B0") // Purple
	case NodeTypeSubWorkflow:
		icon = common.IconSubworkflow
		label = "subworkflow"
		color = lipgloss.Color("#2196F3") // Blue
	case NodeTypeCall:
		if len(node.Children) > 0 {
			icon = "↻"
			label = "scatter"
			color = lipgloss.Color("#FFA726") // Orange
		} else {
			icon = common.IconTask
			label = "task"
			color = lipgloss.Color("#4CAF50") // Green
		}
	case NodeTypeShard:
		icon = common.IconShard
		label = "shard"
		color = lipgloss.Color("#4CAF50") // Green
	default:
		icon = "·"
		label = "node"
		color = lipgloss.Color("#9E9E9E") // Gray
	}

	// Type style (colored, lowercase)
	typeStyle := lipgloss.NewStyle().
		Foreground(color).
		Bold(true)

	return icon + " " + typeStyle.Render(label)
}

// renderScatterSummary renders a summary for Call nodes that have shards
func (m Model) renderScatterSummary(node *TreeNode) string {
	var sb strings.Builder
	children := node.Children
	total := len(children)

	if total == 0 {
		return ""
	}

	sb.WriteString(titleStyle.Render("📊 Shards Summary") + "\n")
	sb.WriteString(labelStyle.Render("Total Shards: ") + valueStyle.Render(fmt.Sprintf("%d", total)) + "\n")

	// Count status breakdown
	statusCounts := make(map[string]int)
	var durations []time.Duration
	for _, child := range children {
		statusCounts[child.Status]++
		if child.Duration > 0 {
			durations = append(durations, child.Duration)
		}
	}

	// Status breakdown
	sb.WriteString(labelStyle.Render("Status: "))
	var parts []string
	if c := statusCounts["Done"]; c > 0 {
		parts = append(parts, lipgloss.NewStyle().Foreground(lipgloss.Color("#4CAF50")).Render(fmt.Sprintf("✓ %d Done", c)))
	}
	if c := statusCounts["Running"]; c > 0 {
		parts = append(parts, lipgloss.NewStyle().Foreground(lipgloss.Color("#2196F3")).Render(fmt.Sprintf("● %d Running", c)))
	}
	if c := statusCounts["Failed"]; c > 0 {
		parts = append(parts, lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6B6B")).Render(fmt.Sprintf("✗ %d Failed", c)))
	}
	if c := statusCounts["Preempted"]; c > 0 {
		parts = append(parts, lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA726")).Render(fmt.Sprintf("↺ %d Preempted", c)))
	}
	sb.WriteString(strings.Join(parts, "  ") + "\n")

	// Timing statistics
	if len(durations) > 0 {
		sb.WriteString("\n" + titleStyle.Render("⏱ Timing") + "\n")

		// Total duration (wall clock from first start to last end)
		if !node.Start.IsZero() && !node.End.IsZero() {
			sb.WriteString(labelStyle.Render("Wall Clock: ") + valueStyle.Render(formatDuration(node.End.Sub(node.Start))) + "\n")
		}

		// Calculate min/max/avg
		var minDur, maxDur, sumDur time.Duration
		minDur = durations[0]
		for _, d := range durations {
			sumDur += d
			if d < minDur {
				minDur = d
			}
			if d > maxDur {
				maxDur = d
			}
		}
		avgDur := sumDur / time.Duration(len(durations))

		sb.WriteString(labelStyle.Render("Per-shard: ") + "\n")
		sb.WriteString("  " + mutedStyle.Render("Min: ") + valueStyle.Render(formatDuration(minDur)) + "\n")
		sb.WriteString("  " + mutedStyle.Render("Max: ") + valueStyle.Render(formatDuration(maxDur)) + "\n")
		sb.WriteString("  " + mutedStyle.Render("Avg: ") + valueStyle.Render(formatDuration(avgDur)) + "\n")
	}

	// Hint
	sb.WriteString("\n" + mutedStyle.Render("Expand node to see individual shards"))

	return sb.String()
}

// renderActionBar renders compact action hints based on node type.
func (m Model) renderActionBar(node *TreeNode) string {
	var actions []string

	// Helper to format action - using footer-style pattern
	formatAction := func(key, desc string, enabled bool) string {
		if !enabled {
			return ""
		}
		return common.KeyStyle.Render(key) + common.DescStyle.Render(" "+desc)
	}

	// Render actions based on node type
	switch node.Type {
	case NodeTypeWorkflow, NodeTypeSubWorkflow:
		var meta *WorkflowMetadata
		if node.Type == NodeTypeWorkflow {
			meta = m.metadata
		} else if node.CallData != nil && node.CallData.SubWorkflowMetadata != nil {
			meta = node.CallData.SubWorkflowMetadata
		} else {
			meta = m.metadata
		}

		if a := formatAction("1", "inputs", len(meta.Inputs) > 0); a != "" {
			actions = append(actions, a)
		}
		if a := formatAction("2", "outputs", len(meta.Outputs) > 0); a != "" {
			actions = append(actions, a)
		}
		if a := formatAction("3", "options", meta.SubmittedOptions != ""); a != "" {
			actions = append(actions, a)
		}
		actions = append(actions, formatAction("4", "timeline", true))
		if a := formatAction("5", "log", meta.WorkflowLog != ""); a != "" {
			actions = append(actions, a)
		}

	case NodeTypeCall, NodeTypeShard:
		if node.CallData != nil {
			cd := node.CallData
			if a := formatAction("1", "inputs", len(cd.Inputs) > 0); a != "" {
				actions = append(actions, a)
			}
			if a := formatAction("2", "outputs", len(cd.Outputs) > 0); a != "" {
				actions = append(actions, a)
			}
			if a := formatAction("3", "command", cd.CommandLine != ""); a != "" {
				actions = append(actions, a)
			}
			if a := formatAction("4", "logs", cd.Stdout != "" || cd.Stderr != "" || cd.MonitoringLog != ""); a != "" {
				actions = append(actions, a)
			}
			if a := formatAction("5", "efficiency", cd.MonitoringLog != ""); a != "" {
				actions = append(actions, a)
			}
		}
	}

	if len(actions) == 0 {
		return ""
	}

	// Join actions with separator
	result := mutedStyle.Render("Actions: ") + strings.Join(actions, mutedStyle.Render("  "))

	return result
}

// renderMonitorContent renders the resource efficiency analysis inline
func (m Model) renderMonitorContent() string {
	if m.resourceError != "" {
		return errorStyle.Render("Error: " + m.resourceError)
	}

	if m.resourceReport == nil {
		return mutedStyle.Render("Loading resource analysis... Press 5 again if needed.")
	}

	var sb strings.Builder
	report := m.resourceReport

	// Header with duration and data points
	sb.WriteString(mutedStyle.Render(fmt.Sprintf("⏱ Duration: %s  📊 Data points: %d",
		formatDuration(report.Duration), report.DataPoints)) + "\n\n")

	// CPU Section
	sb.WriteString(titleStyle.Render("💻 CPU") + "\n")
	sb.WriteString(renderGaugeBar(report.CPU.Efficiency, 25) + "\n")
	sb.WriteString(fmt.Sprintf("Peak: %.0f%%  Avg: %.0f%%  Efficiency: %.0f%%\n\n",
		report.CPU.Peak, report.CPU.Avg, report.CPU.Efficiency*100))

	// Memory Section
	sb.WriteString(titleStyle.Render("🧠 Memory") + "\n")
	sb.WriteString(renderGaugeBar(report.Mem.Efficiency, 25) + "\n")
	sb.WriteString(fmt.Sprintf("Peak: %.0fMB / %.0fMB  Efficiency: %.0f%%\n\n",
		report.Mem.Peak, report.Mem.Total, report.Mem.Efficiency*100))

	// Disk Section
	sb.WriteString(titleStyle.Render("💾 Disk") + "\n")
	sb.WriteString(renderGaugeBar(report.Disk.Efficiency, 25) + "\n")
	sb.WriteString(fmt.Sprintf("Peak: %.1fGB / %.1fGB  Efficiency: %.0f%%\n\n",
		report.Disk.Peak, report.Disk.Total, report.Disk.Efficiency*100))

	// Efficiency explanation
	sb.WriteString(mutedStyle.Render("─── How this efficiency is calculated ───") + "\n")
	sb.WriteString(mutedStyle.Render("• CPU: Average usage / 100%") + "\n")
	sb.WriteString(mutedStyle.Render("• Memory & Disk: Peak usage / Total allocated") + "\n")
	sb.WriteString(mutedStyle.Render("Low efficiency = over-provisioned resources") + "\n")

	sb.WriteString(mutedStyle.Render("─── Note ───") + "\n")
	sb.WriteString(mutedStyle.Render("Resource usage depends on input size and") + "\n")
	sb.WriteString(mutedStyle.Render("analysis program efficiency.") + "\n")

	return sb.String()
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

// renderBreadcrumb builds a breadcrumb navigation showing the node hierarchy
func (m Model) renderBreadcrumb(node *TreeNode) string {
	// Build path from root to current node
	var path []struct {
		icon string
		name string
	}

	// Start with current node and walk up to root
	current := node
	for current != nil {
		var icon string
		name := current.Name

		switch current.Type {
		case NodeTypeWorkflow:
			icon = common.IconWorkflow
		case NodeTypeSubWorkflow:
			icon = common.IconSubworkflow
		case NodeTypeCall:
			if len(current.Children) > 0 {
				icon = "↻" // Scatter
			} else {
				icon = common.IconTask
			}
		case NodeTypeShard:
			icon = common.IconShard
			if current.CallData != nil {
				name = fmt.Sprintf("Shard %d", current.CallData.ShardIndex)
			}
		default:
			icon = "·"
		}

		path = append([]struct {
			icon string
			name string
		}{{icon, name}}, path...)

		current = current.Parent
	}

	// Build the breadcrumb string
	var parts []string
	for i, p := range path {
		// Truncate long names
		displayName := p.name
		if len(displayName) > 20 {
			displayName = displayName[:17] + "..."
		}

		if i == len(path)-1 {
			// Current node (active) - highlighted
			parts = append(parts, p.icon+" "+breadcrumbActiveStyle.Render(displayName))
		} else {
			// Parent nodes - muted
			parts = append(parts, p.icon+" "+breadcrumbStyle.Render(displayName))
		}
	}

	separator := breadcrumbSeparatorStyle.Render(" › ")
	return strings.Join(parts, separator)
}

// sectionSeparator returns a horizontal line separator for sections
func sectionSeparator(width int) string {
	if width <= 0 {
		width = 37
	}
	return sectionSeparatorStyle.Render(strings.Repeat("─", width))
}
