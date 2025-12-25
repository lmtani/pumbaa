package debug

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

// renderInputsModal renders the inputs modal.
func (m Model) renderInputsModal() string {
	modalWidth := m.width - 6
	modalHeight := m.height - 4

	title := titleStyle.Render(common.IconInputs + " Workflow Inputs: " + m.metadata.Name)

	content := m.inputsModalViewport.View()

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

// renderOutputsModal renders the outputs modal.
func (m Model) renderOutputsModal() string {
	modalWidth := m.width - 6
	modalHeight := m.height - 4

	title := titleStyle.Render(common.IconOutputs + " Workflow Outputs: " + m.metadata.Name)

	content := m.outputsModalViewport.View()

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

// renderOptionsModal renders the options modal.
func (m Model) renderOptionsModal() string {
	modalWidth := m.width - 6
	modalHeight := m.height - 4

	title := titleStyle.Render(common.IconOptions + " Workflow Options: " + m.metadata.Name)

	content := m.optionsModalViewport.View()

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
		sb.WriteString(modalLabelStyle.Render(k) + "\n")
		sb.WriteString(formatValueForModal(v, m.width-16) + "\n\n")
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
		sb.WriteString(modalLabelStyle.Render(k) + "\n")
		sb.WriteString(formatValueForModal(v, m.width-16) + "\n\n")
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
		return modalValueStyle.Render(m.metadata.SubmittedOptions)
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
		sb.WriteString(modalLabelStyle.Render(k) + "\n")
		sb.WriteString(formatValueForModal(v, m.width-16) + "\n\n")
	}

	return sb.String()
}
