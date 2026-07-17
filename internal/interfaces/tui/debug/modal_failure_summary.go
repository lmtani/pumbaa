package debug

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/lmtani/pumbaa/internal/domain/workflow"
)

// collectFailureGroups groups every failed leaf in the tree by error
// signature, using the domain's failure grouping. When no failed leaves
// carry messages, it falls back to the workflow-level failures.
func collectFailureGroups(root *TreeNode, workflowFailures []Failure) []workflow.FailureGroup {
	grouper := workflow.NewFailureGrouper()

	for _, node := range flattenTree(root) {
		if len(node.Children) > 0 || !isFailedStatus(node.Status) || node.Type == NodeTypeWorkflow {
			continue
		}
		task := workflow.FailureTask{Name: node.Name, ShardIndex: -1}
		if node.CallData != nil {
			task.ShardIndex = node.CallData.ShardIndex
			task.Stderr = node.CallData.Stderr
		}
		if node.CallData != nil && len(node.CallData.Failures) > 0 {
			grouper.AddFailures(task, node.CallData.Failures)
		} else {
			grouper.Add(task, "(no failure message in metadata)")
		}
	}

	groups := grouper.Sorted()
	if len(groups) == 0 {
		grouper.AddFailures(workflow.FailureTask{Name: "(workflow)", ShardIndex: -1}, workflowFailures)
		groups = grouper.Sorted()
	}

	return groups
}

// openFailureSummary opens the aggregated failure summary modal.
func (m Model) openFailureSummary() (tea.Model, tea.Cmd) {
	groups := collectFailureGroups(m.tree, m.metadata.Failures)
	if len(groups) == 0 {
		m.setStatusMessage("No failures found")
		return m, getClearStatusCmd()
	}

	styled, raw := m.formatFailureSummary(groups)
	m.failureSummaryRaw = raw
	m.activeModal = ModalFailureSummary
	m.failureSummaryViewport = viewport.New(m.width-10, m.height-8)
	m.failureSummaryViewport.SetContent(styled)
	return m, nil
}

// formatFailureSummary renders the groups for the modal viewport and as raw
// text for the clipboard.
func (m Model) formatFailureSummary(groups []workflow.FailureGroup) (styled, raw string) {
	const maxTasksShown = 5
	width := m.width - 14

	var sb, rawSB strings.Builder
	for i, group := range groups {
		count := len(group.Tasks)
		names := make([]string, len(group.Tasks))
		for j, task := range group.Tasks {
			names[j] = task.Name
		}

		header := fmt.Sprintf("%d× %s", count, group.Sample)
		sb.WriteString(errorStyle.Render(fmt.Sprintf("✗ %d×", count)) + " " +
			errorMsgStyle.Render(wrapText(group.Sample, width-7)) + "\n")
		rawSB.WriteString(header + "\n")

		shown := names
		if len(shown) > maxTasksShown {
			shown = shown[:maxTasksShown]
		}
		taskLine := strings.Join(shown, ", ")
		if extra := count - len(shown); extra > 0 {
			taskLine += fmt.Sprintf(" (+%d more)", extra)
		}
		sb.WriteString(mutedStyle.Render(wrapText("  "+taskLine, width)) + "\n")
		rawSB.WriteString("  " + strings.Join(names, ", ") + "\n")

		if i < len(groups)-1 {
			sb.WriteString("\n")
			rawSB.WriteString("\n")
		}
	}

	return sb.String(), rawSB.String()
}

// renderFailureSummaryModal renders the failure summary modal.
func (m Model) renderFailureSummaryModal() string {
	title := errorStyle.Render("⚠  Failure Summary") + " " +
		mutedStyle.Render("(grouped by error signature)")
	content := renderModalViewportContent(m.failureSummaryViewport.View(), m.failureSummaryViewport.Width, false, "")
	return m.renderStandardModal(title, content, m.modalFooterWithHints("↑↓ scroll", "y copy", "esc close"))
}

// handleFailureSummaryModalKeys handles keyboard input in the failure
// summary modal.
func (m Model) handleFailureSummaryModalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	actions := viewportModalActions{
		onClose: func(m *Model) {
			m.activeModal = ModalNone
			m.failureSummaryRaw = ""
		},
		onCopy: func(m *Model) tea.Cmd {
			if m.failureSummaryRaw != "" {
				return copyToClipboard(m.failureSummaryRaw, "failure summary")
			}
			return nil
		},
	}
	cmd, handled := m.handleViewportModalKeys(msg, &m.failureSummaryViewport, actions)
	if handled {
		return m, cmd
	}
	return m, nil
}
