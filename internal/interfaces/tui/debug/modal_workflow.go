package debug

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

// renderInputsModal renders the inputs modal.
func (m Model) renderInputsModal() string {
	title := titleStyle.Render(common.IconInputs + " Workflow Inputs: " + m.metadata.Name)

	content := renderModalViewportContent(
		m.inputsModalViewport.View(),
		m.inputsModalViewport.Width,
		false,
		"",
	)

	footer := m.modalFooter()

	return m.renderStandardModal(title, content, footer)
}

// renderOutputsModal renders the outputs modal.
func (m Model) renderOutputsModal() string {
	title := titleStyle.Render(common.IconOutputs + " Workflow Outputs: " + m.metadata.Name)

	content := renderModalViewportContent(
		m.outputsModalViewport.View(),
		m.outputsModalViewport.Width,
		false,
		"",
	)

	footer := m.modalFooter()

	return m.renderStandardModal(title, content, footer)
}

// renderOptionsModal renders the options modal.
func (m Model) renderOptionsModal() string {
	title := titleStyle.Render(common.IconOptions + " Workflow Options: " + m.metadata.Name)

	content := renderModalViewportContent(
		m.optionsModalViewport.View(),
		m.optionsModalViewport.Width,
		false,
		"",
	)

	footer := m.modalFooter()

	return m.renderStandardModal(title, content, footer)
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
	var options map[string]any
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
