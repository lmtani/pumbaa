package dashboard

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

// handleLabelsModalKeys processes keyboard input in the labels modal.
func (m Model) handleLabelsModalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle editing mode (text input)
	if m.labelsEditing {
		return m.handleLabelsEditingKeys(msg)
	}

	switch msg.String() {
	case "esc", "q":
		m.showLabelsModal = false
		m.labelsData = nil
		m.labelsEditing = false
		m.labelsMessage = ""
		return m, nil

	case "up", "k":
		if m.labelsCursor > 0 {
			m.labelsCursor--
			m.labelsMessage = "" // Clear message on navigation
		}

	case "down", "j":
		if m.labelsData != nil && m.labelsCursor < len(m.labelsData)-1 {
			m.labelsCursor++
			m.labelsMessage = "" // Clear message on navigation
		}

	case "a": // Add new label
		m.labelsEditing = true
		m.labelsEditKey = ""
		m.labelsEditValue = ""
		m.labelsInput = textinput.New()
		m.labelsInput.Placeholder = "key:value"
		m.labelsInput.Focus()
		m.labelsInput.CharLimit = 100
		m.labelsInput.Width = 40
		return m, textinput.Blink

	case "e": // Edit selected label
		if m.labelsData != nil && len(m.labelsData) > 0 {
			keys := getSortedLabelKeys(m.labelsData)
			if m.labelsCursor < len(keys) {
				key := keys[m.labelsCursor]
				m.labelsEditing = true
				m.labelsEditKey = key
				m.labelsEditValue = m.labelsData[key]
				m.labelsInput = textinput.New()
				m.labelsInput.SetValue(m.labelsData[key])
				m.labelsInput.Focus()
				m.labelsInput.CharLimit = 100
				m.labelsInput.Width = 40
				return m, textinput.Blink
			}
		}

	case "d": // Delete - not supported by Cromwell API
		m.labelsMessage = "⚠ Cromwell API does not support label deletion"
	}

	return m, nil
}

// handleLabelsEditingKeys handles keyboard input when editing a label.
func (m Model) handleLabelsEditingKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.Type {
	case tea.KeyEsc:
		m.labelsEditing = false
		m.labelsInput.Blur()
		return m, nil

	case tea.KeyEnter:
		value := m.labelsInput.Value()

		var updateLabels map[string]string
		if m.labelsEditKey == "" {
			// Adding new label - parse key:value
			if value == "" {
				m.labelsEditing = false
				return m, nil
			}
			parts := strings.SplitN(value, ":", 2)
			if len(parts) != 2 {
				m.labelsMessage = "Invalid format. Use key:value"
				m.labelsEditing = false
				return m, nil
			}
			updateLabels = map[string]string{strings.TrimSpace(parts[0]): strings.TrimSpace(parts[1])}
		} else {
			// Editing existing label - allow empty value
			updateLabels = map[string]string{m.labelsEditKey: value}
		}

		// Optimistic local update - update labelsData immediately
		if m.labelsData == nil {
			m.labelsData = make(map[string]string)
		}
		for k, v := range updateLabels {
			m.labelsData[k] = v
		}

		m.labelsEditing = false
		m.labelsUpdating = true
		m.labelsInput.Blur()
		return m, tea.Batch(m.spinner.Tick, m.updateLabels(m.labelsWorkflowID, updateLabels))
	}

	m.labelsInput, cmd = m.labelsInput.Update(msg)
	return m, cmd
}

// renderLabelsModal renders the labels modal.
func (m Model) renderLabelsModal() string {
	width := minInt(80, m.width-10)
	height := minInt(20, m.height-10)

	var content strings.Builder

	// Title left, spinner right
	titleText := fmt.Sprintf("%s Labels: %s", common.IconLabels, truncateName(m.labelsWorkflowName, 35))
	title := common.TitleStyle.Render(titleText)

	// Build header row with activity indicator on the right
	headerRow := title
	if m.labelsLoading || m.labelsUpdating {
		// Just show spinner - no extra icons
		spinnerText := m.spinner.View()
		spinnerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))

		// Calculate padding to push spinner to the right
		titleLen := lipgloss.Width(title)
		spinnerLen := lipgloss.Width(spinnerText)
		availableWidth := width - 6 // account for modal padding
		padding := availableWidth - titleLen - spinnerLen
		if padding < 2 {
			padding = 2
		}
		headerRow = title + strings.Repeat(" ", padding) + spinnerStyle.Render(spinnerText)
	}
	content.WriteString(headerRow + "\n\n")

	if m.labelsEditing {
		// Show editing input
		if m.labelsEditKey == "" {
			content.WriteString(common.LabelStyle.Render("Add new label (key:value):") + "\n")
		} else {
			content.WriteString(common.LabelStyle.Render(fmt.Sprintf("Edit value for '%s':", m.labelsEditKey)) + "\n")
		}
		content.WriteString(m.labelsInput.View() + "\n")
	} else if m.labelsData == nil || len(m.labelsData) == 0 {
		if m.labelsLoading {
			content.WriteString(common.MutedStyle.Render("Fetching labels...") + "\n")
		} else {
			content.WriteString(common.MutedStyle.Render("No labels found") + "\n")
			content.WriteString(common.MutedStyle.Render("Press 'a' to add a label") + "\n")
		}
	} else {
		// Sort keys for consistent display
		keys := getSortedLabelKeys(m.labelsData)

		// Render each label (dimmed when updating)
		for i, key := range keys {
			value := m.labelsData[key]

			prefix := "  "
			if i == m.labelsCursor {
				prefix = "▶ "
			}

			keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#87CEEB")).Bold(true)
			valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))

			if m.labelsUpdating || m.labelsLoading {
				keyStyle = keyStyle.Faint(true)
				valueStyle = valueStyle.Faint(true)
			}

			line := fmt.Sprintf("%s%s: %s", prefix, keyStyle.Render(key), valueStyle.Render(value))
			content.WriteString(line + "\n")
		}
	}

	// Show in-modal message if any
	if m.labelsMessage != "" {
		content.WriteString("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#FFAA00")).Render(m.labelsMessage) + "\n")
	}

	// Footer
	content.WriteString("\n" + common.MutedStyle.Render("───────────────────────────────────────") + "\n")
	if m.labelsEditing {
		content.WriteString(common.KeyStyle.Render("enter") + common.DescStyle.Render(" save") + "  ")
		content.WriteString(common.KeyStyle.Render("esc") + common.DescStyle.Render(" cancel"))
	} else {
		content.WriteString(common.KeyStyle.Render("↑↓") + common.DescStyle.Render(" nav") + "  ")
		content.WriteString(common.KeyStyle.Render("a") + common.DescStyle.Render(" add") + "  ")
		content.WriteString(common.KeyStyle.Render("e") + common.DescStyle.Render(" edit") + "  ")
		content.WriteString(common.KeyStyle.Render("esc") + common.DescStyle.Render(" close"))
	}

	// Create modal box
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(common.PrimaryColor).
		Padding(1, 2).
		Width(width).
		Height(height)

	modal := modalStyle.Render(content.String())

	// Center the modal
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		modal,
	)
}

// getSortedLabelKeys returns sorted keys from a labels map.
func getSortedLabelKeys(labels map[string]string) []string {
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// truncateName truncates a name to maxLen characters.
func truncateName(name string, maxLen int) string {
	if len(name) <= maxLen {
		return name
	}
	return name[:maxLen-3] + "..."
}
