package debug

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
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
	switch m.viewMode {
	case ViewModeCommand:
		return m.renderCommand(node)
	case ViewModeLogs:
		return m.renderLogs(node)
	case ViewModeInputs:
		return m.renderInputs(node)
	case ViewModeOutputs:
		return m.renderOutputs(node)
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

	// Call-specific details
	if node.CallData != nil {
		cd := node.CallData

		// Quick Actions section - FIRST!
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

		if cd.Stdout != "" || cd.Stderr != "" || cd.MonitoringLog != "" {
			sb.WriteString(buttonStyle.Render(" 4 ") + " Logs")
		} else {
			sb.WriteString(disabledButtonStyle.Render(" 4 ") + mutedStyle.Render(" Logs"))
		}

		sb.WriteString("\n")

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
						sb.WriteString(pathStyle.Render(metadata.WorkflowRoot) + "\n")
					}
					if metadata.WorkflowLog != "" {
						sb.WriteString(labelStyle.Render("Log:") + " " + mutedStyle.Render("(w to view)") + "\n")
						sb.WriteString(pathStyle.Render(metadata.WorkflowLog) + "\n")
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
	monitoringPrefix := "  "
	if m.focus == FocusDetails {
		switch m.logCursor {
		case 0:
			stdoutPrefix = "▶ "
		case 1:
			stderrPrefix = "▶ "
		case 2:
			monitoringPrefix = "▶ "
		}
	}

	sb.WriteString(stdoutPrefix + labelStyle.Render("stdout: ") + "\n")
	if cd.Stdout != "" {
		sb.WriteString("  " + pathStyle.Render(cd.Stdout) + "\n\n")
	} else {
		sb.WriteString("  " + mutedStyle.Render("(not available)") + "\n\n")
	}

	sb.WriteString(stderrPrefix + labelStyle.Render("stderr: ") + "\n")
	if cd.Stderr != "" {
		sb.WriteString("  " + pathStyle.Render(cd.Stderr) + "\n\n")
	} else {
		sb.WriteString("  " + mutedStyle.Render("(not available)") + "\n\n")
	}

	sb.WriteString(monitoringPrefix + labelStyle.Render("monitoring: ") + "\n")
	if cd.MonitoringLog != "" {
		sb.WriteString("  " + pathStyle.Render(cd.MonitoringLog) + "\n\n")
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
