package debug

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lmtani/pumbaa/internal/application/workflow/debuginfo"
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

func (m Model) renderTree() string {
	var sb strings.Builder

	startIdx := 0
	maxVisible := m.height - 10 // Leave room for header and footer
	if maxVisible < 5 {
		maxVisible = 5
	}
	if m.cursor >= maxVisible {
		startIdx = m.cursor - maxVisible + 1
	}

	endIdx := startIdx + maxVisible
	if endIdx > len(m.nodes) {
		endIdx = len(m.nodes)
	}

	for i := startIdx; i < endIdx; i++ {
		node := m.nodes[i]
		sb.WriteString(m.renderTreeNode(node, i))
		sb.WriteString("\n")
	}

	style := treePanelStyle.Width(m.treeWidth).Height(m.height - 8)
	if m.focus == FocusTree {
		style = style.BorderForeground(lipgloss.Color("#7D56F4"))
	}

	return style.Render(sb.String())
}

func (m Model) renderTreeNode(node *TreeNode, index int) string {
	prefix := strings.Repeat("  ", node.Depth)

	// Node indicator
	indicator := "├─"
	// Check if this is the last child of its parent
	isLast := false
	if node.Parent != nil {
		children := node.Parent.Children
		if len(children) > 0 && children[len(children)-1] == node {
			isLast = true
		}
	}
	if isLast {
		indicator = "└─"
	}

	// Expand/collapse indicator
	expandIndicator := " "
	if len(node.Children) > 0 || (node.Type == NodeTypeSubWorkflow && node.SubWorkflowID != "") {
		if node.Expanded {
			expandIndicator = "▼"
		} else {
			expandIndicator = "▶"
		}
	}

	// Status icon
	statusIcon := statusStyle(node.Status)

	// Node type icon
	typeIcon := ""
	switch node.Type {
	case NodeTypeWorkflow:
		typeIcon = "📋"
	case NodeTypeCall:
		typeIcon = "📦"
	case NodeTypeSubWorkflow:
		typeIcon = "📂"
	case NodeTypeShard:
		typeIcon = "📄"
	}

	// Name with truncation
	name := truncate(node.Name, m.treeWidth-node.Depth*2-12)

	// Build the node string
	nodeStr := fmt.Sprintf("%s%s %s %s %s %s", prefix, indicator, expandIndicator, statusIcon, typeIcon, name)

	// Style based on selection
	if index == m.cursor {
		return selectedStyle.Render(nodeStr)
	}
	return nodeStr
}

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
	case ViewModeTimeline:
		return "⏱ Timeline"
	default:
		return "📊 Details"
	}
}

func (m Model) renderDetailsContent(node *TreeNode) string {
	switch m.viewMode {
	case ViewModeCommand:
		return m.renderCommand(node)
	case ViewModeLogs:
		return m.renderLogs(node)
	case ViewModeInputs:
		return m.renderInputs(node)
	case ViewModeOutputs:
		return m.renderOutputs(node)
	case ViewModeTimeline:
		return m.renderTimeline(node)
	default:
		return m.renderBasicDetails(node)
	}
}

func (m Model) renderBasicDetails(node *TreeNode) string {
	var sb strings.Builder

	// Node info
	sb.WriteString(titleStyle.Render("📌 Node Info") + "\n")
	sb.WriteString(labelStyle.Render("Name: ") + valueStyle.Render(node.Name) + "\n")
	sb.WriteString(labelStyle.Render("Type: ") + valueStyle.Render(nodeTypeName(node.Type)) + "\n")
	sb.WriteString(labelStyle.Render("Status: ") + statusStyle(node.Status) + " " + valueStyle.Render(node.Status) + "\n")
	if node.SubWorkflowID != "" {
		sb.WriteString(labelStyle.Render("SubWorkflow ID: ") + valueStyle.Render(node.SubWorkflowID) + "\n")
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

	// Call-specific details
	if node.CallData == nil {
		return sb.String()
	}

	cd := node.CallData

	sb.WriteString("\n")

	// Timing
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

	sb.WriteString("\n")

	// Resources
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

	sb.WriteString("\n")

	// Docker
	sb.WriteString(titleStyle.Render("🐳 Docker") + "\n")
	if cd.DockerImage != "" {
		sb.WriteString(labelStyle.Render("Image: ") + valueStyle.Render(truncate(cd.DockerImage, 50)) + "\n")
	}
	if cd.DockerSize != "" {
		sb.WriteString(labelStyle.Render("Size: ") + valueStyle.Render(cd.DockerSize) + "\n")
	}

	sb.WriteString("\n")

	// Cache
	sb.WriteString(titleStyle.Render("📦 Cache") + "\n")
	cacheStatus := "Miss"
	if cd.CacheHit {
		cacheStatus = "Hit"
	}
	sb.WriteString(labelStyle.Render("Status: ") + valueStyle.Render(cacheStatus) + "\n")
	if cd.CacheResult != "" {
		sb.WriteString(labelStyle.Render("Result: ") + valueStyle.Render(cd.CacheResult) + "\n")
	}

	// Cost
	if cd.VMCostPerHour > 0 {
		sb.WriteString("\n")
		sb.WriteString(titleStyle.Render("💰 Cost") + "\n")
		sb.WriteString(labelStyle.Render("VM Cost/Hour: ") + valueStyle.Render(fmt.Sprintf("$%.4f", cd.VMCostPerHour)) + "\n")
	}

	// Quick Actions section
	sb.WriteString("\n")
	sb.WriteString(titleStyle.Render("⚡ Quick Actions") + "\n")

	// Show available actions based on data
	if len(cd.Inputs) > 0 {
		sb.WriteString(buttonStyle.Render(" 1 ") + " Inputs  ")
	} else {
		sb.WriteString(disabledButtonStyle.Render(" 1 ") + mutedStyle.Render(" Inputs  "))
	}

	if len(cd.Outputs) > 0 {
		sb.WriteString(buttonStyle.Render(" 2 ") + " Outputs  ")
	} else {
		sb.WriteString(disabledButtonStyle.Render(" 2 ") + mutedStyle.Render(" Outputs  "))
	}

	if cd.CommandLine != "" {
		sb.WriteString(buttonStyle.Render(" 3 ") + " Command  ")
	} else {
		sb.WriteString(disabledButtonStyle.Render(" 3 ") + mutedStyle.Render(" Command  "))
	}

	if cd.Stdout != "" || cd.Stderr != "" {
		sb.WriteString(buttonStyle.Render(" 4 ") + " Logs")
	} else {
		sb.WriteString(disabledButtonStyle.Render(" 4 ") + mutedStyle.Render(" Logs"))
	}

	sb.WriteString("\n")

	return sb.String()
}

func (m Model) renderCommand(node *TreeNode) string {
	if node.CallData == nil || node.CallData.CommandLine == "" {
		return mutedStyle.Render("No command available")
	}
	// Wrap text to fit the viewport width
	wrapped := wrapText(node.CallData.CommandLine, m.detailsWidth-8)
	return commandStyle.Render(wrapped)
}

func (m Model) renderLogs(node *TreeNode) string {
	if node.CallData == nil {
		return mutedStyle.Render("No logs available")
	}

	var sb strings.Builder
	cd := node.CallData

	// Show selection indicator when details panel is focused
	stdoutPrefix := "  "
	stderrPrefix := "  "
	if m.focus == FocusDetails {
		if m.logCursor == 0 {
			stdoutPrefix = "▶ "
		} else {
			stderrPrefix = "▶ "
		}
	}

	sb.WriteString(stdoutPrefix + labelStyle.Render("stdout: ") + "\n")
	sb.WriteString("  " + pathStyle.Render(cd.Stdout) + "\n\n")

	sb.WriteString(stderrPrefix + labelStyle.Render("stderr: ") + "\n")
	sb.WriteString("  " + pathStyle.Render(cd.Stderr) + "\n\n")

	sb.WriteString("  " + labelStyle.Render("Call Root: ") + "\n")
	sb.WriteString("  " + pathStyle.Render(cd.CallRoot) + "\n\n")

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

func (m Model) renderTimeline(node *TreeNode) string {
	if node.CallData == nil || len(node.CallData.ExecutionEvents) == 0 {
		return mutedStyle.Render("No timeline available")
	}

	var sb strings.Builder
	for _, event := range node.CallData.ExecutionEvents {
		time := event.Start.Format("15:04:05")
		sb.WriteString(labelStyle.Render(time) + " " + valueStyle.Render(event.Description) + "\n")
	}
	return sb.String()
}

func (m Model) renderFooter() string {
	var footer string
	if m.statusMessage != "" {
		footer = warningStyle.Render(m.statusMessage)
	} else {
		footer = " ↑↓ navigate • tab switch • d details • c cmd • i inputs • o outputs • T durations • ? help • q quit"
	}
	return helpBarStyle.Width(m.width - 2).Render(footer)
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

// renderFailures renders workflow-level failures
func (m Model) renderFailures() string {
	var sb strings.Builder

	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF6B6B")).
		Bold(true)

	errorMsgStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF8E8E"))

	sb.WriteString(errorStyle.Render("⚠️  Workflow Failures") + "\n\n")

	for i, failure := range m.metadata.Failures {
		sb.WriteString(renderFailure(failure, 0, i+1, errorMsgStyle))
	}

	return sb.String()
}

// renderFailure recursively renders a failure and its causes
func renderFailure(f Failure, depth int, index int, style lipgloss.Style) string {
	var sb strings.Builder
	indent := strings.Repeat("  ", depth)

	// Main failure message
	if depth == 0 {
		sb.WriteString(fmt.Sprintf("%s%d. %s\n", indent, index, style.Render(f.Message)))
	} else {
		sb.WriteString(fmt.Sprintf("%s└─ %s\n", indent, style.Render(f.Message)))
	}

	// Render causes
	for _, cause := range f.CausedBy {
		sb.WriteString(renderFailure(cause, depth+1, 0, style))
	}

	return sb.String()
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

	var summary *debuginfo.WorkflowPreemptionSummary
	if m.preemption != nil && node.Type == NodeTypeWorkflow {
		// If model was constructed with DebugInfo, use its precomputed summary for the workflow root
		summary = m.preemption
	} else {
		summary = debuginfo.CalculateWorkflowPreemptionSummary(workflowID, workflowName, calls)
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

	// Overall stats
	sb.WriteString(labelStyle.Render("Preemptible Tasks: ") +
		valueStyle.Render(fmt.Sprintf("%d / %d", summary.PreemptibleTasks, summary.TotalTasks)) + "\n")

	sb.WriteString(labelStyle.Render("Total Attempts: ") +
		valueStyle.Render(fmt.Sprintf("%d", summary.TotalAttempts)) + "\n")

	if summary.TotalPreemptions > 0 {
		sb.WriteString(labelStyle.Render("Total Preemptions: ") +
			lipgloss.NewStyle().Foreground(lipgloss.Color("#FFAA00")).Render(fmt.Sprintf("%d", summary.TotalPreemptions)) + "\n")
	} else {
		sb.WriteString(labelStyle.Render("Total Preemptions: ") +
			valueStyle.Render("0") + "\n")
	}

	// Cost-weighted efficiency (main metric)
	costEff := summary.CostEfficiency
	costEffStr := fmt.Sprintf("%.0f%%", costEff*100)
	var costEffStyled string
	if costEff >= 0.8 {
		costEffStyled = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Bold(true).Render(costEffStr + " ✓")
	} else if costEff >= 0.5 {
		costEffStyled = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFF00")).Bold(true).Render(costEffStr + " ⚠")
	} else {
		costEffStyled = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Bold(true).Render(costEffStr + " ✗")
	}
	sb.WriteString(labelStyle.Render("Cost Efficiency: ") + costEffStyled + "\n")

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
				task.Preemptions,
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
