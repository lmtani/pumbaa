package debug

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	adksession "google.golang.org/adk/session"

	"github.com/lmtani/pumbaa/internal/interfaces/tui/chat"
)

// initializeChatWithContext creates and initializes the chat model with the collected context.
func (m *Model) initializeChatWithContext(taskContext string) tea.Cmd {
	return func() tea.Msg {
		// Create a new session for this chat
		ctx := context.Background()

		var sess adksession.Session
		if m.sessionSvc != nil {
			resp, err := m.sessionSvc.Create(ctx, &adksession.CreateRequest{
				AppName: "pumbaa-debug",
				UserID:  "default",
			})
			if err != nil {
				return chatInitErrorMsg{err: fmt.Errorf("failed to create session: %w", err)}
			}
			sess = resp.Session
		}

		// Build dynamic system instruction with task context
		systemInstruction := fmt.Sprintf("%s\n\n---\n\n%s", taskDebugSystemInstruction, taskContext)

		// Create the chat model
		chatModel := chat.NewModel(m.llm, m.chatTools, systemInstruction, m.sessionSvc, sess)

		return chatInitializedMsg{
			chatModel: &chatModel,
		}
	}
}

// chatInitializedMsg is sent when the chat model is ready.
type chatInitializedMsg struct {
	chatModel *chat.Model
}

// chatInitErrorMsg is sent when chat initialization fails.
type chatInitErrorMsg struct {
	err error
}

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

// handleChatInitialized handles successful chat initialization.
func (m Model) handleChatInitialized(msg chatInitializedMsg) (tea.Model, tea.Cmd) {
	m.chatModel = msg.chatModel

	// Calculate chat dimensions (modal minus padding)
	chatWidth := m.width - 10
	chatHeight := m.height - 12

	// Send window size to chat model so it becomes ready
	windowSizeMsg := tea.WindowSizeMsg{
		Width:  chatWidth,
		Height: chatHeight,
	}

	// Update chat model with window size
	updatedModel, cmd := m.chatModel.Update(windowSizeMsg)
	if chatModel, ok := updatedModel.(*chat.Model); ok {
		m.chatModel = chatModel
	}

	// Initialize the chat model and combine with any command from window size update
	initCmd := m.chatModel.Init()
	return m, tea.Batch(initCmd, cmd)
}

// handleChatInitError handles chat initialization errors.
func (m Model) handleChatInitError(msg chatInitErrorMsg) (tea.Model, tea.Cmd) {
	m.showChatModal = false
	m.chatModel = nil
	m.setStatusMessage(fmt.Sprintf("Chat error: %v", msg.err))
	return m, getClearStatusCmd()
}
