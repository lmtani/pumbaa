// Package tui provides the terminal user interface for the application.
package tui

import (
	"context"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	adksession "google.golang.org/adk/session"

	"github.com/lmtani/pumbaa/internal/domain/workflow"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/chat"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/dashboard"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/debug"
)

// Screen represents the different screens available in the TUI.
type Screen int

const (
	ScreenDashboard Screen = iota
	ScreenDebug
	ScreenChat
)

const (
	debugChatAppName = "pumbaa-debug"
	debugChatUserID  = "default"
)

// Navigation messages - these are used to navigate between screens

// NavigateToDebugMsg requests navigation to the debug screen.
type NavigateToDebugMsg struct {
	Workflow *workflow.Workflow
}

// NavigateToChatMsg requests navigation to the chat screen.
type NavigateToChatMsg struct {
	SystemInstruction string
	ContextSummary    string
}

// NavigateBackMsg requests navigation back to the previous screen.
type NavigateBackMsg struct{}

// NavigateToDashboardMsg requests navigation back to the dashboard.
type NavigateToDashboardMsg struct{}

// AppModel is the root model that coordinates navigation between screens.
type AppModel struct {
	currentScreen  Screen
	previousScreen Screen
	startScreen    Screen // The screen we started on (for determining if ESC should quit or go back)
	dashboard      dashboard.Model
	debug          debug.Model
	chat           *chat.Model
	width          int
	height         int
	globalKeys     common.GlobalKeys
	deps           *Dependencies

	// State for preserving context during navigation
	lastWorkflow        *workflow.Workflow
	lastChatInstruction string
	lastChatSummary     string

	// Quit confirmation modal
	showQuitConfirm bool
}

// NewAppModel creates a new app model with the given dependencies.
func NewAppModel(deps *Dependencies, initialScreen Screen) AppModel {
	m := AppModel{
		currentScreen: initialScreen,
		startScreen:   initialScreen,
		globalKeys:    common.DefaultGlobalKeys(),
		deps:          deps,
	}

	// Initialize dashboard
	m.dashboard = dashboard.NewModelWithFetcher(deps.Repository)
	m.dashboard.SetMetadataFetcher(deps.Repository)
	m.dashboard.SetHealthChecker(deps.Repository)
	m.dashboard.SetLabelManager(deps.Repository)
	m.dashboard.SetMetadataParser(deps.MetadataParser)
	m.dashboard.SetCurrentVersion(deps.CurrentVersion)

	return m
}

// NewAppModelWithWorkflow creates a new app model starting at the debug screen.
func NewAppModelWithWorkflow(deps *Dependencies, wf *workflow.Workflow) AppModel {
	m := AppModel{
		currentScreen: ScreenDebug,
		startScreen:   ScreenDebug,
		globalKeys:    common.DefaultGlobalKeys(),
		deps:          deps,
		lastWorkflow:  wf,
	}

	// Initialize debug directly
	m.debug = debug.NewModelWithChat(
		wf,
		deps.Repository,
		deps.MetadataParser,
		deps.MonitoringUC,
		deps.FileProvider,
		deps.BatchLogsUC,
		convertChatDeps(deps.ChatDeps),
	)

	// Since we're starting directly on debug, ESC should quit, not go back
	m.debug.SetCanGoBack(false)

	return m
}

// Init implements tea.Model.
func (m AppModel) Init() tea.Cmd {
	switch m.currentScreen {
	case ScreenDashboard:
		return m.dashboard.Init()
	case ScreenDebug:
		return m.debug.Init()
	case ScreenChat:
		if m.chat != nil {
			return m.chat.Init()
		}
	}
	return nil
}

// Update implements tea.Model.
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Propagate to current screen
		return m.updateCurrentScreen(msg)

	case tea.KeyMsg:
		// Handle quit confirmation modal first
		if m.showQuitConfirm {
			return m.handleQuitConfirmKeys(msg)
		}

		// Ctrl+C shows quit confirmation
		if key.Matches(msg, m.globalKeys.Quit) {
			m.showQuitConfirm = true
			return m, nil
		}

		// ESC handling: delegate to screen first, then handle back navigation
		if msg.Type == tea.KeyEsc {
			return m.handleEscapeKey()
		}

	// Handle navigation messages
	case NavigateToDebugMsg:
		return m.navigateToDebug(msg.Workflow)

	// Handle navigation from dashboard (different package, same message type)
	case dashboard.NavigateToDebugMsg:
		return m.navigateToDebug(msg.Workflow)

	case NavigateToChatMsg:
		return m.navigateToChat(msg.SystemInstruction, msg.ContextSummary)

	// Handle navigation from debug (different package)
	case debug.NavigateToChatMsg:
		return m.navigateToChat(msg.SystemInstruction, msg.ContextSummary)

	case NavigateBackMsg:
		return m.navigateBack()

	case NavigateToDashboardMsg:
		return m.navigateToDashboard()
	}

	return m.updateCurrentScreen(msg)
}

// navigateToDebug switches to the debug screen with the given workflow.
func (m AppModel) navigateToDebug(wf *workflow.Workflow) (tea.Model, tea.Cmd) {
	m.previousScreen = m.currentScreen
	m.currentScreen = ScreenDebug
	m.lastWorkflow = wf

	m.debug = debug.NewModelWithChat(
		wf,
		m.deps.Repository,
		m.deps.MetadataParser,
		m.deps.MonitoringUC,
		m.deps.FileProvider,
		m.deps.BatchLogsUC,
		convertChatDeps(m.deps.ChatDeps),
	)

	// Send window size to new screen
	cmd := m.debug.Init()
	if m.width > 0 && m.height > 0 {
		sizeCmd := func() tea.Msg {
			return tea.WindowSizeMsg{Width: m.width, Height: m.height}
		}
		return m, tea.Batch(cmd, sizeCmd)
	}
	return m, cmd
}

// navigateToChat switches to the chat screen.
func (m AppModel) navigateToChat(systemInstruction, contextSummary string) (tea.Model, tea.Cmd) {
	if m.deps.ChatDeps == nil || m.deps.ChatDeps.LLM == nil {
		// Chat not available, stay on current screen
		return m, nil
	}

	m.previousScreen = m.currentScreen
	m.currentScreen = ScreenChat
	m.lastChatInstruction = systemInstruction
	m.lastChatSummary = contextSummary

	// Create chat session
	ctx := context.Background()
	var sess adksession.Session
	if m.deps.ChatDeps.SessionSvc != nil {
		resp, err := m.deps.ChatDeps.SessionSvc.Create(ctx, &adksession.CreateRequest{
			AppName: debugChatAppName,
			UserID:  debugChatUserID,
		})
		if err == nil {
			sess = resp.Session
		}
	}

	chatModel := chat.NewModel(
		m.deps.ChatDeps.LLM,
		m.deps.ChatDeps.Tools,
		systemInstruction,
		m.deps.ChatDeps.SessionSvc,
		sess,
	)
	m.chat = &chatModel
	m.chat.SetContextLabel("Task Context")
	if contextSummary != "" {
		m.chat.AddInfoMessage(contextSummary)
	}

	cmd := m.chat.Init()
	if m.width > 0 && m.height > 0 {
		sizeCmd := func() tea.Msg {
			return tea.WindowSizeMsg{Width: m.width, Height: m.height}
		}
		return m, tea.Batch(cmd, sizeCmd)
	}
	return m, cmd
}

// navigateBack returns to the previous screen.
func (m AppModel) navigateBack() (tea.Model, tea.Cmd) {
	switch m.previousScreen {
	case ScreenDashboard:
		return m.navigateToDashboard()
	case ScreenDebug:
		if m.lastWorkflow != nil {
			return m.navigateToDebug(m.lastWorkflow)
		}
		return m.navigateToDashboard()
	default:
		return m.navigateToDashboard()
	}
}

// navigateToDashboard returns to the dashboard screen (preserving state).
func (m AppModel) navigateToDashboard() (tea.Model, tea.Cmd) {
	m.previousScreen = m.currentScreen
	m.currentScreen = ScreenDashboard

	// Dashboard maintains state, just send window size
	var cmds []tea.Cmd
	if m.width > 0 && m.height > 0 {
		cmds = append(cmds, func() tea.Msg {
			return tea.WindowSizeMsg{Width: m.width, Height: m.height}
		})
	}

	if len(cmds) > 0 {
		return m, tea.Batch(cmds...)
	}
	return m, nil
}

// updateCurrentScreen delegates the update to the current screen.
func (m AppModel) updateCurrentScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.currentScreen {
	case ScreenDashboard:
		newModel, cmd := m.dashboard.Update(msg)
		m.dashboard = newModel.(dashboard.Model)

		// Check for navigation request from dashboard
		if navCmd := m.dashboard.GetNavigationCmd(); navCmd != nil {
			m.dashboard.ClearNavigation()
			// Execute the command and handle the resulting message
			return m, navCmd
		}

		// Check if dashboard wants to quit
		if m.dashboard.ShouldQuit {
			return m, tea.Quit
		}

		return m, cmd

	case ScreenDebug:
		newModel, cmd := m.debug.Update(msg)
		m.debug = newModel.(debug.Model)

		// Check for back navigation from debug
		if m.debug.ShouldGoBack() {
			m.debug.ClearNavigation()
			return m.navigateToDashboard()
		}

		// Check for navigation request from debug (to chat)
		if navCmd := m.debug.GetNavigationCmd(); navCmd != nil {
			m.debug.ClearNavigation()
			return m, navCmd
		}

		return m, cmd

	case ScreenChat:
		if m.chat == nil {
			return m, nil
		}
		newModel, cmd := m.chat.Update(msg)
		m.chat = newModel.(*chat.Model)

		// Check for quit from chat
		if m.chat.ShouldQuit() {
			return m, tea.Quit
		}

		// Check for back navigation from chat
		if m.chat.ShouldGoBack() {
			m.chat.ClearNavigation()
			return m.navigateBack()
		}

		return m, cmd
	}

	return m, nil
}

// handleQuitConfirmKeys handles key presses when quit confirmation modal is shown.
func (m AppModel) handleQuitConfirmKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		return m, tea.Quit
	case "n", "N", "esc":
		m.showQuitConfirm = false
	}
	return m, nil
}

// handleEscapeKey handles ESC key press with the new navigation flow.
func (m AppModel) handleEscapeKey() (tea.Model, tea.Cmd) {
	// Check if current screen has a modal open or can handle ESC internally
	switch m.currentScreen {
	case ScreenDebug:
		// Check if debug has a modal open
		if m.debug.HasActiveModal() {
			// Let debug handle it
			return m.updateCurrentScreen(tea.KeyMsg{Type: tea.KeyEsc})
		}
		// If debug is in search mode, let it handle to exit search
		if m.debug.IsSearchActive() {
			return m.updateCurrentScreen(tea.KeyMsg{Type: tea.KeyEsc})
		}
		// If debug is in a non-tree view mode, let it handle to return to tree view
		if m.debug.GetViewMode() != debug.ViewModeTree {
			return m.updateCurrentScreen(tea.KeyMsg{Type: tea.KeyEsc})
		}
		// If we started on debug, show quit confirmation (no dashboard to go back to)
		if m.startScreen == ScreenDebug {
			m.showQuitConfirm = true
			return m, nil
		}
		// Otherwise, go back to dashboard
		return m.navigateToDashboard()

	case ScreenChat:
		if m.chat != nil {
			// If chat has a modal, let it handle
			if m.chat.HasActiveModal() {
				return m.updateCurrentScreen(tea.KeyMsg{Type: tea.KeyEsc})
			}
			// If in FocusInput, switch to FocusMessages
			if m.chat.GetFocusMode() == chat.FocusInput {
				m.chat.SetFocusMode(chat.FocusMessages)
				return m, nil
			}
			// If we started on chat, show quit confirmation
			if m.startScreen == ScreenChat {
				m.showQuitConfirm = true
				return m, nil
			}
			// In FocusMessages, go back
			return m.navigateBack()
		}
		// If we started on chat, show quit confirmation
		if m.startScreen == ScreenChat {
			m.showQuitConfirm = true
			return m, nil
		}
		return m.navigateBack()

	case ScreenDashboard:
		// If dashboard has a modal open, let it handle ESC
		if m.dashboard.HasActiveModal() {
			return m.updateCurrentScreen(tea.KeyMsg{Type: tea.KeyEsc})
		}
		// Dashboard is root - show quit confirmation
		m.showQuitConfirm = true
		return m, nil
	}

	return m, nil
}

// View implements tea.Model.
func (m AppModel) View() string {
	// Show quit confirmation modal if active
	if m.showQuitConfirm {
		return m.renderWithQuitModal()
	}

	switch m.currentScreen {
	case ScreenDashboard:
		return m.dashboard.View()
	case ScreenDebug:
		return m.debug.View()
	case ScreenChat:
		if m.chat != nil {
			return m.chat.View()
		}
	}
	return ""
}

// renderWithQuitModal renders the current screen with a quit confirmation modal overlay.
func (m AppModel) renderWithQuitModal() string {
	// Get the background (current screen)
	var bg string
	switch m.currentScreen {
	case ScreenDashboard:
		bg = m.dashboard.View()
	case ScreenDebug:
		bg = m.debug.View()
	case ScreenChat:
		if m.chat != nil {
			bg = m.chat.View()
		}
	}

	// Create the modal
	modalContent := common.TitleStyle.Render("Quit Pumbaa?") + "\n\n" +
		"Are you sure you want to exit?\n\n" +
		common.KeyStyle.Render("[Y]") + " " + common.DescStyle.Render("Yes, quit") + "    " +
		common.KeyStyle.Render("[N]") + " " + common.DescStyle.Render("No, stay")

	modal := common.ModalStyle.
		Width(40).
		Render(modalContent)

	// Center the modal
	modalWidth := 44  // modal width + border
	modalHeight := 7  // approximate modal height

	// Calculate position
	x := (m.width - modalWidth) / 2
	y := (m.height - modalHeight) / 2

	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	// Place modal on top of background
	return common.PlaceOverlay(x, y, modal, bg)
}

// convertChatDeps converts tui.ChatDependencies to debug.ChatDependencies.
func convertChatDeps(deps *ChatDependencies) *debug.ChatDependencies {
	if deps == nil {
		return nil
	}
	return &debug.ChatDependencies{
		LLM:        deps.LLM,
		Tools:      deps.Tools,
		SessionSvc: deps.SessionSvc,
	}
}
