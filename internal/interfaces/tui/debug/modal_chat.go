package debug

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/lmtani/pumbaa/internal/interfaces/tui/chat"
)

// renderChatModal renders the embedded chat as a modal overlay using the same style as other modals.
func (m Model) renderChatModal() string {
	modalWidth := m.width - 6
	modalHeight := m.height - 4

	// Title
	title := titleStyle.Render("🤖 Debug Chat")

	var content string
	if m.chatModel == nil {
		// Show loading state
		content = m.loadingSpinner.View() + " Initializing chat..."
	} else {
		// Get the chat view - the chat model handles its own layout
		content = m.chatModel.View()
	}

	// Footer with instructions
	footer := mutedStyle.Render("ctrl+d send • ↑↓ scroll • tab navigate • esc close")

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

// handleChatModalUpdate delegates updates to the embedded chat model.
func (m Model) handleChatModalUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.chatModel == nil {
		return m, nil
	}

	// Check for ESC to close the chat
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.Type == tea.KeyEsc {
			m.showChatModal = false
			m.chatModel = nil
			m.chatContextNode = nil
			return m, nil
		}
	}

	// Delegate to chat model
	updatedModel, cmd := m.chatModel.Update(msg)
	if chatModel, ok := updatedModel.(*chat.Model); ok {
		m.chatModel = chatModel
	}

	return m, cmd
}

// handleChatContextLoaded handles the completion of context collection.
func (m Model) handleChatContextLoaded(msg chatContextLoadedMsg) (tea.Model, tea.Cmd) {
	m.chatContextLoading = false
	m.isLoading = false
	m.loadingMessage = ""

	systemInstruction := fmt.Sprintf("%s\n\n---\n\n%s", taskDebugSystemInstruction, msg.context)
	m.NavigateToChatSystemInstruction = systemInstruction
	return m, tea.Quit
}
