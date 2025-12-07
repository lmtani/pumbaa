package debug

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderLogModal renders the log modal.
func (m Model) renderLogModal() string {
	modalWidth := m.width - 6
	modalHeight := m.height - 4

	// Modal title
	title := titleStyle.Render("📄 " + m.logModalTitle)

	// Modal content
	var content string
	if m.logModalError != "" {
		content = errorStyle.Render("Error: " + m.logModalError)
	} else if m.logModalLoading {
		content = mutedStyle.Render("Loading...")
	} else {
		content = m.logModalViewport.View()
	}

	// Footer with instructions
	footer := mutedStyle.Render("↑↓/PgUp/PgDn scroll • esc close")

	// Build modal box
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

	// Center the modal
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		modal,
	)
}

// renderInputsModal renders the inputs modal.
func (m Model) renderInputsModal() string {
	modalWidth := m.width - 6
	modalHeight := m.height - 4

	title := titleStyle.Render("📥 Workflow Inputs: " + m.metadata.Name)

	content := m.inputsModalViewport.View()

	footer := mutedStyle.Render("↑↓/PgUp/PgDn scroll • esc close")

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

// renderOutputsModal renders the outputs modal.
func (m Model) renderOutputsModal() string {
	modalWidth := m.width - 6
	modalHeight := m.height - 4

	title := titleStyle.Render("📤 Workflow Outputs: " + m.metadata.Name)

	content := m.outputsModalViewport.View()

	footer := mutedStyle.Render("↑↓/PgUp/PgDn scroll • esc close")

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

// renderOptionsModal renders the options modal.
func (m Model) renderOptionsModal() string {
	modalWidth := m.width - 6
	modalHeight := m.height - 4

	title := titleStyle.Render("⚙️  Workflow Options: " + m.metadata.Name)

	content := m.optionsModalViewport.View()

	footer := mutedStyle.Render("↑↓/PgUp/PgDn scroll • esc close")

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

	footer := mutedStyle.Render("↑↓/PgUp/PgDn scroll • esc close")

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

	footer := mutedStyle.Render("↑↓/PgUp/PgDn scroll • esc close")

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

	footer := mutedStyle.Render("↑↓/PgUp/PgDn scroll • esc close")

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

// formatInputsForModal formats inputs for display in the modal.
func (m Model) formatInputsForModal(node *TreeNode) string {
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
		sb.WriteString(labelStyle.Render(k) + "\n")
		sb.WriteString(formatValue(v, m.width-16) + "\n\n")
	}

	return sb.String()
}

// formatOutputsForModal formats outputs for display in the modal.
func (m Model) formatOutputsForModal(node *TreeNode) string {
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
		sb.WriteString(labelStyle.Render(k) + "\n")
		sb.WriteString(formatValue(v, m.width-16) + "\n\n")
	}

	return sb.String()
}

// formatWorkflowInputsForModal formats workflow inputs for display in the modal.
func (m Model) formatWorkflowInputsForModal() string {
	if len(m.metadata.Inputs) == 0 {
		return mutedStyle.Render("No inputs available")
	}

	var sb strings.Builder

	// Sort keys for consistent display
	keys := make([]string, 0, len(m.metadata.Inputs))
	for k := range m.metadata.Inputs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := m.metadata.Inputs[k]
		// Skip null values
		if v == nil {
			continue
		}
		sb.WriteString(labelStyle.Render(k) + "\n")
		sb.WriteString(formatValue(v, m.width-16) + "\n\n")
	}

	return sb.String()
}

// formatWorkflowOutputsForModal formats workflow outputs for display in the modal.
func (m Model) formatWorkflowOutputsForModal() string {
	if len(m.metadata.Outputs) == 0 {
		return mutedStyle.Render("No outputs available")
	}

	var sb strings.Builder

	// Sort keys for consistent display
	keys := make([]string, 0, len(m.metadata.Outputs))
	for k := range m.metadata.Outputs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := m.metadata.Outputs[k]
		// Skip null values
		if v == nil {
			continue
		}
		sb.WriteString(labelStyle.Render(k) + "\n")
		sb.WriteString(formatValue(v, m.width-16) + "\n\n")
	}

	return sb.String()
}

// formatOptionsForModal formats workflow options for display in the modal.
func (m Model) formatOptionsForModal() string {
	if m.metadata.SubmittedOptions == "" {
		return mutedStyle.Render("No options available")
	}

	// Parse the JSON options
	var options map[string]interface{}
	if err := json.Unmarshal([]byte(m.metadata.SubmittedOptions), &options); err != nil {
		// If it's not valid JSON, just return the raw string formatted
		return valueStyle.Render(m.metadata.SubmittedOptions)
	}

	var sb strings.Builder

	// Sort keys for consistent display
	keys := make([]string, 0, len(options))
	for k := range options {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := options[k]
		// Skip null values
		if v == nil {
			continue
		}
		sb.WriteString(labelStyle.Render(k) + "\n")
		sb.WriteString(formatValue(v, m.width-16) + "\n\n")
	}

	return sb.String()
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
		sb.WriteString(labelStyle.Render(k) + "\n")
		sb.WriteString(formatValue(v, m.width-16) + "\n\n")
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
		sb.WriteString(labelStyle.Render(k) + "\n")
		sb.WriteString(formatValue(v, m.width-16) + "\n\n")
	}

	return sb.String()
}

// formatCallCommandForModal formats call command for display in the modal.
func (m Model) formatCallCommandForModal(node *TreeNode) string {
	if node.CallData == nil || node.CallData.CommandLine == "" {
		return mutedStyle.Render("No command available")
	}

	// Wrap text to fit the modal width
	wrapped := wrapText(node.CallData.CommandLine, m.width-20)
	return commandStyle.Render(wrapped)
}

// formatValue formats a value for human-readable display.
func formatValue(v interface{}, maxWidth int) string {
	switch val := v.(type) {
	case nil:
		return mutedStyle.Render("  null")
	case bool:
		return valueStyle.Render(fmt.Sprintf("  %v", val))
	case float64:
		// Check if it's an integer
		if val == float64(int64(val)) {
			return valueStyle.Render(fmt.Sprintf("  %d", int64(val)))
		}
		return valueStyle.Render(fmt.Sprintf("  %g", val))
	case string:
		// Handle GCS paths with special styling
		if strings.HasPrefix(val, "gs://") {
			return pathStyle.Render("  " + val)
		}
		// Handle local paths
		if strings.HasPrefix(val, "/") {
			return pathStyle.Render("  " + val)
		}
		// Wrap long strings
		if len(val) > maxWidth-2 {
			return valueStyle.Render("  " + wrapText(val, maxWidth-2))
		}
		return valueStyle.Render("  " + val)
	case []interface{}:
		if len(val) == 0 {
			return mutedStyle.Render("  []")
		}
		var sb strings.Builder
		for i, item := range val {
			prefix := "  - "
			itemStr := formatValue(item, maxWidth-4)
			// Remove leading spaces from nested formatValue
			itemStr = strings.TrimPrefix(itemStr, "  ")
			sb.WriteString(prefix + itemStr)
			if i < len(val)-1 {
				sb.WriteString("\n")
			}
		}
		return sb.String()
	case map[string]interface{}:
		// Pretty print maps with indentation
		jsonBytes, err := json.MarshalIndent(val, "  ", "  ")
		if err != nil {
			return mutedStyle.Render("  [complex object]")
		}
		return valueStyle.Render("  " + string(jsonBytes))
	default:
		// Fallback to JSON for unknown types
		jsonBytes, err := json.MarshalIndent(val, "  ", "  ")
		if err != nil {
			return valueStyle.Render(fmt.Sprintf("  %v", val))
		}
		return valueStyle.Render("  " + string(jsonBytes))
	}
}
