package debug

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// chatSelectionOption represents a selectable option in the chat data selection modal.
type chatSelectionOption struct {
	label       string
	description string
	field       *bool
	available   bool // Whether this option is available for the current node
}

// getChatSelectionOptions returns the list of selectable options for chat context.
func (m *Model) getChatSelectionOptions() []chatSelectionOption {
	node := m.chatContextNode
	hasStdout := node != nil && node.CallData != nil && node.CallData.Stdout != ""
	hasStderr := node != nil && node.CallData != nil && node.CallData.Stderr != ""
	hasMonitoring := node != nil && node.CallData != nil && node.CallData.MonitoringLog != ""
	hasBatchLogs := node != nil && node.CallData != nil && node.CallData.JobID != "" && m.batchLogsUC != nil

	return []chatSelectionOption{
		{
			label:       "Metadata",
			description: "Task info, status, timing, command",
			field:       &m.chatDataSelections.Metadata,
			available:   true,
		},
		{
			label:       "Stderr",
			description: "Standard error output (last 200 lines)",
			field:       &m.chatDataSelections.Stderr,
			available:   hasStderr,
		},
		{
			label:       "Stdout",
			description: "Standard output (last 200 lines)",
			field:       &m.chatDataSelections.Stdout,
			available:   hasStdout,
		},
		{
			label:       "Monitoring",
			description: "Resource efficiency analysis",
			field:       &m.chatDataSelections.MonitoringLog,
			available:   hasMonitoring,
		},
		{
			label:       "Batch Logs",
			description: "Google Batch logs (slow)",
			field:       &m.chatDataSelections.BatchLogs,
			available:   hasBatchLogs,
		},
	}
}

// renderChatSelectionModal renders the modal for selecting data to include in chat.
func (m Model) renderChatSelectionModal() string {
	modalWidth := 65
	modalHeight := 18

	// Title
	title := titleStyle.Render("Chat with AI")

	// Subtitle
	subtitle := mutedStyle.Render("Select data to include in context:")

	// Options
	options := m.getChatSelectionOptions()
	var optionLines []string

	for i, opt := range options {
		// Checkbox
		checkbox := "[ ]"
		if *opt.field {
			checkbox = "[x]"
		}

		// Style based on availability and selection
		var line string
		if !opt.available {
			// Unavailable option
			line = mutedStyle.Render(fmt.Sprintf("  %s %s (unavailable)", checkbox, opt.label))
		} else if i == m.chatSelectionCursor {
			// Selected/focused option
			checkboxStyled := lipgloss.NewStyle().
				Foreground(primaryColor).
				Bold(true).
				Render(checkbox)
			labelStyled := lipgloss.NewStyle().
				Foreground(textColor).
				Bold(true).
				Render(opt.label)
			descStyled := mutedStyle.Render(" - " + opt.description)
			line = fmt.Sprintf("▶ %s %s%s", checkboxStyled, labelStyled, descStyled)
		} else {
			// Normal option
			descStyled := mutedStyle.Render(" - " + opt.description)
			line = fmt.Sprintf("  %s %s%s", checkbox, opt.label, descStyled)
		}

		optionLines = append(optionLines, line)
	}

	optionsContent := strings.Join(optionLines, "\n")

	// Warning note
	note := mutedStyle.Render("\nNote: Data collection may take a few seconds.")

	// Footer
	footer := m.modalFooterWithHints("↑↓ navigate", "space toggle", "enter confirm", "esc cancel")

	content := strings.Join([]string{
		subtitle,
		"",
		optionsContent,
		note,
	}, "\n")

	return m.renderCenteredModal(modalWidth, modalHeight, title, content, footer)
}

// handleChatSelectionModalKeys handles keyboard input in the chat selection modal.
func (m Model) handleChatSelectionModalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	options := m.getChatSelectionOptions()
	maxCursor := len(options) - 1

	switch {
	case key.Matches(msg, m.keys.Escape):
		// Cancel selection
		m.activeModal = ModalNone
		m.chatContextNode = nil
		return m, nil

	case key.Matches(msg, m.keys.Up):
		// Move cursor up, skip unavailable options
		for {
			if m.chatSelectionCursor > 0 {
				m.chatSelectionCursor--
			} else {
				break
			}
			if options[m.chatSelectionCursor].available {
				break
			}
		}
		return m, nil

	case key.Matches(msg, m.keys.Down):
		// Move cursor down, skip unavailable options
		for {
			if m.chatSelectionCursor < maxCursor {
				m.chatSelectionCursor++
			} else {
				break
			}
			if options[m.chatSelectionCursor].available {
				break
			}
		}
		return m, nil

	case key.Matches(msg, m.keys.Space):
		// Toggle current option
		opt := options[m.chatSelectionCursor]
		if opt.available {
			*opt.field = !*opt.field
		}
		return m, nil

	case key.Matches(msg, m.keys.Enter):
		// Confirm selection and start collecting context
		m.activeModal = ModalNone
		m.isLoading = true
		m.loadingMessage = "Collecting task context..."
		return m, tea.Batch(m.loadingSpinner.Tick, m.collectChatContext())
	}

	return m, nil
}

// openChatSelectionModal opens the modal for selecting chat data.
func (m Model) openChatSelectionModal(node *TreeNode) (tea.Model, tea.Cmd) {
	// Check if LLM is configured
	if m.llm == nil {
		m.setStatusMessage("Chat not available. Configure LLM provider first.")
		return m, getClearStatusCmd()
	}

	// Check if node has call data
	if node == nil || node.CallData == nil {
		m.setStatusMessage("No task data available for chat")
		return m, getClearStatusCmd()
	}

	// Initialize selection modal
	m.activeModal = ModalChatSelection
	m.chatContextNode = node
	m.chatSelectionCursor = 0
	m.chatDataSelections = DefaultChatDataSelection()

	// Adjust default selections based on available data
	if node.CallData.Stderr == "" {
		m.chatDataSelections.Stderr = false
	}

	return m, nil
}
