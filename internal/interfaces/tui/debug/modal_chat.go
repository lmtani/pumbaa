package debug

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

// handleChatContextLoaded handles the completion of context collection.
func (m Model) handleChatContextLoaded(msg chatContextLoadedMsg) (tea.Model, tea.Cmd) {
	m.isLoading = false
	m.loadingMessage = ""

	navMsg := common.NavigateToChatMsg{
		SystemInstruction: fmt.Sprintf("%s\n\n---\n\n%s", taskDebugSystemInstruction, msg.context),
		ContextSummary:    m.buildChatContextSummary(msg.errors),
		ContextLabel:      m.buildChatContextLabel(),
	}
	return m, common.NavigateCmd(navMsg)
}

// buildChatContextLabel builds the breadcrumb-style badge shown in the chat
// header, e.g. "my-workflow ▸ align_reads".
func (m Model) buildChatContextLabel() string {
	var wfName string
	if m.metadata != nil {
		wfName = m.metadata.Name
	}

	var taskName string
	if m.chatContextNode != nil {
		taskName = m.chatContextNode.Name
		if taskName == "" && m.chatContextNode.CallData != nil {
			taskName = m.chatContextNode.CallData.Name
		}
	}

	switch {
	case wfName != "" && taskName != "":
		return wfName + " ▸ " + taskName
	case taskName != "":
		return taskName
	case wfName != "":
		return wfName
	default:
		return "Task Context"
	}
}

func (m Model) buildChatContextSummary(errors []string) string {
	var taskName string
	if m.chatContextNode != nil {
		taskName = m.chatContextNode.Name
		if taskName == "" && m.chatContextNode.CallData != nil {
			taskName = m.chatContextNode.CallData.Name
		}
	}

	var contexts []string
	if m.chatDataSelections.Metadata {
		contexts = append(contexts, "Metadata")
	}
	if m.chatDataSelections.Stderr {
		contexts = append(contexts, "Stderr")
	}
	if m.chatDataSelections.Stdout {
		contexts = append(contexts, "Stdout")
	}
	if m.chatDataSelections.MonitoringLog {
		contexts = append(contexts, "Monitoring")
	}
	if m.chatDataSelections.BatchLogs {
		contexts = append(contexts, "Batch Logs")
	}
	if len(contexts) == 0 {
		contexts = append(contexts, "none")
	}

	var sb strings.Builder
	sb.WriteString("Task context")
	if taskName != "" {
		sb.WriteString("\nTask: ")
		sb.WriteString(taskName)
	}
	if len(contexts) > 0 {
		sb.WriteString("\nAvailable: ")
		sb.WriteString(strings.Join(contexts, ", "))
	}
	if len(errors) > 0 {
		sb.WriteString("\nNotes: some items failed to load")
	}

	return strings.TrimSpace(sb.String())
}
