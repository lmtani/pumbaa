package debug

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// handleChatContextLoaded handles the completion of context collection.
func (m Model) handleChatContextLoaded(msg chatContextLoadedMsg) (tea.Model, tea.Cmd) {
	m.isLoading = false
	m.loadingMessage = ""

	systemInstruction := fmt.Sprintf("%s\n\n---\n\n%s", taskDebugSystemInstruction, msg.context)
	m.NavigateToChatSystemInstruction = systemInstruction
	return m, tea.Quit
}
