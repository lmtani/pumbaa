package debug

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	workflowapp "github.com/lmtani/pumbaa/internal/application/workflow"
	"github.com/lmtani/pumbaa/internal/domain/ports"
	"github.com/lmtani/pumbaa/internal/domain/workflow"
)

// CollectedChatData holds the data collected for chat context.
type CollectedChatData struct {
	Metadata         *workflow.Call
	StdoutContent    string
	StderrContent    string
	MonitoringReport *workflow.EfficiencyReport
	BatchLogs        []ports.BatchLogEntry
	Errors           []string // Data sources that failed to load
}

// chatContextLoadedMsg is sent when context collection completes.
type chatContextLoadedMsg struct {
	context string
	errors  []string
}

// chatContextErrorMsg is sent when context collection fails completely.
type chatContextErrorMsg struct {
	err error
}

// collectChatContext returns a command that collects the selected data for chat context.
func (m Model) collectChatContext() tea.Cmd {
	node := m.chatContextNode
	sel := m.chatDataSelections

	if node == nil || node.CallData == nil {
		return func() tea.Msg {
			return chatContextErrorMsg{err: fmt.Errorf("no task data available")}
		}
	}

	return func() tea.Msg {
		ctx := context.Background()
		var data CollectedChatData
		var errors []string

		// Metadata is always available (synchronous)
		if sel.Metadata {
			data.Metadata = node.CallData
		}

		// Collect stderr
		if sel.Stderr && node.CallData.Stderr != "" {
			content, err := m.fileProvider.Read(ctx, node.CallData.Stderr)
			if err != nil {
				errors = append(errors, fmt.Sprintf("Stderr: %v", err))
			} else {
				data.StderrContent = content
			}
		}

		// Collect stdout
		if sel.Stdout && node.CallData.Stdout != "" {
			content, err := m.fileProvider.Read(ctx, node.CallData.Stdout)
			if err != nil {
				errors = append(errors, fmt.Sprintf("Stdout: %v", err))
			} else {
				data.StdoutContent = content
			}
		}

		// Collect monitoring/efficiency analysis
		if sel.MonitoringLog && node.CallData.MonitoringLog != "" && m.monitoringUC != nil {
			input := workflowapp.MonitoringInput{LogPath: node.CallData.MonitoringLog}
			output, err := m.monitoringUC.Execute(ctx, input)
			if err != nil {
				errors = append(errors, fmt.Sprintf("Monitoring: %v", err))
			} else if output != nil {
				data.MonitoringReport = output.Report
			}
		}

		// Collect batch logs
		if sel.BatchLogs && node.CallData.JobID != "" && m.batchLogsUC != nil {
			// Add time window filter for batch logs
			var startTime, endTime time.Time
			if !node.CallData.VMStartTime.IsZero() {
				startTime = node.CallData.VMStartTime.Add(-2 * time.Hour)
			}
			if !node.CallData.VMEndTime.IsZero() {
				endTime = node.CallData.VMEndTime.Add(2 * time.Hour)
			}

			input := workflowapp.GetBatchLogsInput{
				JobName:   node.CallData.JobID,
				Limit:     100, // Limit for chat context
				StartTime: startTime,
				EndTime:   endTime,
			}

			output, err := m.batchLogsUC.Execute(ctx, input)
			if err != nil {
				errors = append(errors, fmt.Sprintf("Batch logs: %v", err))
			} else {
				data.BatchLogs = output.Entries
			}
		}

		data.Errors = errors

		// Format the collected data into context string
		contextStr := formatTaskContext(node.CallData, data)

		return chatContextLoadedMsg{
			context: contextStr,
			errors:  errors,
		}
	}
}

// formatTaskContext formats the collected data into a string for the LLM.
func formatTaskContext(call *workflow.Call, data CollectedChatData) string {
	var sb strings.Builder

	sb.WriteString("# Task Execution Context\n\n")

	// Task Information
	if data.Metadata != nil {
		sb.WriteString("## Task Information\n\n")
		sb.WriteString(fmt.Sprintf("- **Name**: %s\n", call.Name))
		sb.WriteString(fmt.Sprintf("- **Status**: %s\n", call.Status))
		sb.WriteString(fmt.Sprintf("- **Backend**: %s\n", call.Backend))

		if call.ReturnCode != nil && *call.ReturnCode != 0 {
			sb.WriteString(fmt.Sprintf("- **Return Code**: %d\n", *call.ReturnCode))
		}

		// Timing
		if !call.Start.IsZero() {
			sb.WriteString(fmt.Sprintf("- **Started**: %s\n", call.Start.Format(time.RFC3339)))
		}
		if !call.End.IsZero() {
			sb.WriteString(fmt.Sprintf("- **Ended**: %s\n", call.End.Format(time.RFC3339)))
			if !call.Start.IsZero() {
				duration := call.End.Sub(call.Start)
				sb.WriteString(fmt.Sprintf("- **Duration**: %s\n", formatChatDuration(duration)))
			}
		}

		// Resources
		if call.CPU != "" || call.Memory != "" || call.Disk != "" {
			sb.WriteString("\n### Resources\n")
			if call.CPU != "" {
				sb.WriteString(fmt.Sprintf("- **CPU**: %s\n", call.CPU))
			}
			if call.Memory != "" {
				sb.WriteString(fmt.Sprintf("- **Memory**: %s\n", call.Memory))
			}
			if call.Disk != "" {
				sb.WriteString(fmt.Sprintf("- **Disk**: %s\n", call.Disk))
			}
		}

		// Docker
		if call.DockerImage != "" {
			sb.WriteString(fmt.Sprintf("\n### Docker Image\n%s\n", call.DockerImage))
		}

		// Failures
		if len(call.Failures) > 0 {
			sb.WriteString("\n### Failures\n")
			for _, f := range call.Failures {
				sb.WriteString(fmt.Sprintf("- **Message**: %s\n", f.Message))
				if len(f.CausedBy) > 0 {
					for _, cause := range f.CausedBy {
						sb.WriteString(fmt.Sprintf("  - **Caused by**: %s\n", cause.Message))
					}
				}
			}
		}
	}

	// Command
	if call.CommandLine != "" {
		sb.WriteString("\n## Command Executed\n\n```bash\n")
		sb.WriteString(call.CommandLine)
		sb.WriteString("\n```\n")
	}

	// Stderr
	if data.StderrContent != "" {
		sb.WriteString("\n## Stderr (last 200 lines)\n\n```\n")
		sb.WriteString(truncateToLastNLines(data.StderrContent, 200))
		sb.WriteString("\n```\n")
	}

	// Stdout
	if data.StdoutContent != "" {
		sb.WriteString("\n## Stdout (last 100 lines)\n\n```\n")
		sb.WriteString(truncateToLastNLines(data.StdoutContent, 100))
		sb.WriteString("\n```\n")
	}

	// Monitoring/Efficiency Report
	if data.MonitoringReport != nil {
		sb.WriteString("\n## Resource Efficiency Analysis\n\n")
		report := data.MonitoringReport
		sb.WriteString(fmt.Sprintf("- **Duration**: %s\n", formatChatDuration(report.Duration)))
		sb.WriteString(fmt.Sprintf("- **Data Points**: %d\n", report.DataPoints))

		if report.CPU.Peak > 0 {
			sb.WriteString("\n### CPU Usage\n")
			sb.WriteString(fmt.Sprintf("- Peak: %.1f%%\n", report.CPU.Peak))
			sb.WriteString(fmt.Sprintf("- Avg: %.1f%%\n", report.CPU.Avg))
			sb.WriteString(fmt.Sprintf("- Efficiency: %.1f%%\n", report.CPU.Efficiency*100))
		}

		if report.Mem.Peak > 0 {
			sb.WriteString("\n### Memory Usage\n")
			sb.WriteString(fmt.Sprintf("- Peak: %.1f MB\n", report.Mem.Peak))
			sb.WriteString(fmt.Sprintf("- Avg: %.1f MB\n", report.Mem.Avg))
			sb.WriteString(fmt.Sprintf("- Efficiency: %.1f%%\n", report.Mem.Efficiency*100))
		}

		if len(report.Recommendations) > 0 {
			sb.WriteString("\n### Recommendations\n")
			for _, rec := range report.Recommendations {
				sb.WriteString(fmt.Sprintf("- %s\n", rec))
			}
		}
	}

	// Batch Logs
	if len(data.BatchLogs) > 0 {
		sb.WriteString("\n## Google Batch Logs (last 50 entries)\n\n```\n")
		entries := data.BatchLogs
		if len(entries) > 50 {
			entries = entries[len(entries)-50:]
		}
		for _, entry := range entries {
			sb.WriteString(fmt.Sprintf("[%s] [%s] %s\n",
				entry.Timestamp.Format("15:04:05"),
				entry.Severity,
				entry.Message))
		}
		sb.WriteString("```\n")
	}

	// Errors during collection
	if len(data.Errors) > 0 {
		sb.WriteString("\n## Data Collection Notes\n\n")
		sb.WriteString("Some data could not be collected:\n")
		for _, err := range data.Errors {
			sb.WriteString(fmt.Sprintf("- %s\n", err))
		}
	}

	return sb.String()
}

// truncateToLastNLines returns the last n lines of a string.
func truncateToLastNLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) <= n {
		return s
	}
	return strings.Join(lines[len(lines)-n:], "\n")
}

// formatChatDuration formats a duration in a human-readable format for chat context.
func formatChatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		secs := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm %ds", mins, secs)
	}
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh %dm", hours, mins)
}

// taskDebugSystemInstruction is the system instruction for task debugging chat.
const taskDebugSystemInstruction = `You are Pumbaa, a helpful assistant specialized in debugging Cromwell/WDL workflow tasks.

The user has provided context about a specific task execution that may have failed or has issues. Your job is to:

1. **Analyze the failure**: Look at the stderr, return code, and failure messages to identify the root cause
2. **Check resource usage**: If monitoring data is provided, identify potential resource issues (OOM, disk full, etc.)
3. **Provide actionable recommendations**: Suggest specific fixes or next steps
4. **Be concise**: Focus on the most likely cause and solution

Guidelines:
- Be technical and direct
- Use markdown formatting for clarity
- If you see common error patterns (OOM killer, disk space, permission denied), identify them immediately
- Suggest concrete changes to WDL runtime attributes if resource issues are detected
- Respond in the user's language (English or Portuguese)

You have access to tools for querying Cromwell and reading files if the user needs additional information.
`
