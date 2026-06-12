package debug

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

// copyMenuItem is one copyable value in the copy menu modal.
type copyMenuItem struct {
	label string
	value string
}

// buildCopyMenuItems collects the copyable values for the given node,
// skipping anything empty.
func (m Model) buildCopyMenuItems(node *TreeNode) []copyMenuItem {
	var items []copyMenuItem
	add := func(label, value string) {
		if value != "" {
			items = append(items, copyMenuItem{label: label, value: value})
		}
	}

	switch node.Type {
	case NodeTypeWorkflow, NodeTypeSubWorkflow:
		meta := m.metadata
		if node.Type == NodeTypeSubWorkflow && node.CallData != nil && node.CallData.SubWorkflowMetadata != nil {
			meta = node.CallData.SubWorkflowMetadata
		}
		add("Workflow ID", meta.ID)
		if node.Type == NodeTypeSubWorkflow && meta.ID == "" {
			add("Workflow ID", node.SubWorkflowID)
		}
		add("Workflow root", meta.WorkflowRoot)
		add("Workflow log", meta.WorkflowLog)
		add("Inputs JSON", meta.SubmittedInputs)
		add("Options JSON", meta.SubmittedOptions)
	default:
		if call := node.CallData; call != nil {
			add("Stderr path", call.Stderr)
			add("Stdout path", call.Stdout)
			add("Call root", call.CallRoot)
			add("Command line", call.CommandLine)
			add("Docker image", call.DockerImage)
			add("Monitoring log", call.MonitoringLog)
			add("Job ID", call.JobID)
		}
		add("Workflow ID", m.metadata.ID)
	}

	return items
}

// openCopyMenu opens the copy menu for the currently selected node.
func (m Model) openCopyMenu() (tea.Model, tea.Cmd) {
	if m.cursor >= len(m.nodes) {
		return m, nil
	}

	items := m.buildCopyMenuItems(m.nodes[m.cursor])
	if len(items) == 0 {
		m.setStatusMessage("Nothing to copy for this node")
		return m, getClearStatusCmd()
	}

	m.copyMenuItems = items
	m.copyMenuCursor = 0
	m.activeModal = ModalCopyMenu
	return m, nil
}

// renderCopyMenuModal renders the copy menu modal.
func (m Model) renderCopyMenuModal() string {
	modalWidth := minInt(70, m.width-8)
	modalHeight := minInt(len(m.copyMenuItems)+8, m.height-4)

	title := titleStyle.Render("Copy to Clipboard")

	valueWidth := modalWidth - 24
	var lines []string
	for i, item := range m.copyMenuItems {
		shortcut := " "
		if i < 9 {
			shortcut = fmt.Sprintf("%d", i+1)
		}
		value := truncatePath(item.value, valueWidth)
		if i == m.copyMenuCursor {
			lines = append(lines, fmt.Sprintf("▶ %s %s %s",
				selectedStyle.Render(shortcut),
				labelStyle.Render(common.PadRight(item.label, 14)),
				valueStyle.Render(value)))
		} else {
			lines = append(lines, fmt.Sprintf("  %s %s %s",
				mutedStyle.Render(shortcut),
				labelStyle.Render(common.PadRight(item.label, 14)),
				mutedStyle.Render(value)))
		}
	}

	footer := m.modalFooterWithHints("↑↓ navigate", "enter/y copy", "1-9 quick copy", "esc close")
	return m.renderCenteredModal(modalWidth, modalHeight, title, strings.Join(lines, "\n"), footer)
}

// handleCopyMenuModalKeys handles keyboard input in the copy menu modal.
func (m Model) handleCopyMenuModalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Escape):
		m.activeModal = ModalNone
		return m, nil

	case key.Matches(msg, m.keys.Up):
		if m.copyMenuCursor > 0 {
			m.copyMenuCursor--
		}
		return m, nil

	case key.Matches(msg, m.keys.Down):
		if m.copyMenuCursor < len(m.copyMenuItems)-1 {
			m.copyMenuCursor++
		}
		return m, nil

	case key.Matches(msg, m.keys.Enter), key.Matches(msg, m.keys.Copy):
		return m.copyMenuSelect(m.copyMenuCursor)
	}

	// Number keys 1-9 copy directly
	if s := msg.String(); len(s) == 1 && s[0] >= '1' && s[0] <= '9' {
		idx := int(s[0] - '1')
		if idx < len(m.copyMenuItems) {
			return m.copyMenuSelect(idx)
		}
	}

	return m, nil
}

// copyMenuSelect copies the item at idx and closes the modal.
func (m Model) copyMenuSelect(idx int) (tea.Model, tea.Cmd) {
	item := m.copyMenuItems[idx]
	m.activeModal = ModalNone
	return m, copyToClipboard(item.value, item.label)
}
