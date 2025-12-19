package debug

import (
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

// renderCallInputsModal renders the call inputs modal.
func (m Model) renderCallInputsModal() string {
	modalWidth := m.width - 6
	modalHeight := m.height - 4

	// Get current node for title
	nodeName := "Unknown"
	if m.cursor < len(m.nodes) {
		nodeName = m.nodes[m.cursor].Name
	}

	title := titleStyle.Render("📥 Call Inputs: " + nodeName)

	content := m.callInputsViewport.View()

	footer := m.modalFooter()

	modalContent := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		content,
		"",
		footer,
	)

	modal := modalStyle.
		Width(modalWidth).
		Height(modalHeight).
		Render(modalContent)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		modal,
	)
}

// renderCallOutputsModal renders the call outputs modal.
func (m Model) renderCallOutputsModal() string {
	modalWidth := m.width - 6
	modalHeight := m.height - 4

	// Get current node for title
	nodeName := "Unknown"
	if m.cursor < len(m.nodes) {
		nodeName = m.nodes[m.cursor].Name
	}

	title := titleStyle.Render("📤 Call Outputs: " + nodeName)

	content := m.callOutputsViewport.View()

	footer := m.modalFooter()

	modalContent := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		content,
		"",
		footer,
	)

	modal := modalStyle.
		Width(modalWidth).
		Height(modalHeight).
		Render(modalContent)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		modal,
	)
}

// renderCallCommandModal renders the call command modal.
func (m Model) renderCallCommandModal() string {
	modalWidth := m.width - 6
	modalHeight := m.height - 4

	// Get current node for title
	nodeName := "Unknown"
	if m.cursor < len(m.nodes) {
		nodeName = m.nodes[m.cursor].Name
	}

	title := titleStyle.Render("📜 Call Command: " + nodeName)

	content := m.callCommandViewport.View()

	footer := m.modalFooter()

	modalContent := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		content,
		"",
		footer,
	)

	modal := modalStyle.
		Width(modalWidth).
		Height(modalHeight).
		Render(modalContent)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		modal,
	)
}

// formatCallInputsForModal formats call inputs for display in the modal.
func (m Model) formatCallInputsForModal(node *TreeNode) string {
	if node.CallData == nil || len(node.CallData.Inputs) == 0 {
		return mutedStyle.Render("No inputs available")
	}

	var sb strings.Builder

	// Sort keys for consistent display
	keys := make([]string, 0, len(node.CallData.Inputs))
	for k := range node.CallData.Inputs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := node.CallData.Inputs[k]
		// Skip null values
		if v == nil {
			continue
		}
		sb.WriteString(modalLabelStyle.Render(k) + "\n")
		sb.WriteString(formatValueForModal(v, m.width-16) + "\n\n")
	}

	return sb.String()
}

// formatCallOutputsForModal formats call outputs for display in the modal.
func (m Model) formatCallOutputsForModal(node *TreeNode) string {
	if node.CallData == nil || len(node.CallData.Outputs) == 0 {
		return mutedStyle.Render("No outputs available")
	}

	var sb strings.Builder

	// Sort keys for consistent display
	keys := make([]string, 0, len(node.CallData.Outputs))
	for k := range node.CallData.Outputs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := node.CallData.Outputs[k]
		// Skip null values
		if v == nil {
			continue
		}
		sb.WriteString(modalLabelStyle.Render(k) + "\n")
		sb.WriteString(formatValueForModal(v, m.width-16) + "\n\n")
	}

	return sb.String()
}

// formatCallCommandForModal formats call command for display in the modal.
func (m Model) formatCallCommandForModal(node *TreeNode) string {
	if node.CallData == nil || node.CallData.CommandLine == "" {
		return mutedStyle.Render("No command available")
	}

	// Aplicar syntax highlighting como Bash
	highlighted := common.Highlight(node.CallData.CommandLine, common.ProfileShell, m.width-20)
	return highlighted
}
