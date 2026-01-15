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
}

// NewAppModel creates a new app model with the given dependencies.
func NewAppModel(deps *Dependencies, startScreen Screen) AppModel {
	m := AppModel{
		currentScreen: startScreen,
		globalKeys:    common.DefaultGlobalKeys(),
		deps:          deps,
	}

	// Initialize dashboard
	m.dashboard = dashboard.NewModelWithFetcher(deps.Repository)
	m.dashboard.SetMetadataFetcher(deps.Repository)
	m.dashboard.SetHealthChecker(deps.Repository)
	m.dashboard.SetLabelManager(deps.Repository)
	m.dashboard.SetMetadataParser(deps.MetadataParser)

	return m
}

// NewAppModelWithWorkflow creates a new app model starting at the debug screen.
func NewAppModelWithWorkflow(deps *Dependencies, wf *workflow.Workflow) AppModel {
	m := AppModel{
		currentScreen: ScreenDebug,
		globalKeys:    common.DefaultGlobalKeys(),
		deps:          deps,
		lastWorkflow:  wf,
	}

	// Initialize debug directly
	m.debug = debug.NewModelWithChat(
		wf,
		deps.Repository,
		deps.MonitoringUC,
		deps.FileProvider,
		deps.BatchLogsUC,
		convertChatDeps(deps.ChatDeps),
	)

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
		// Global quit
		if key.Matches(msg, m.globalKeys.Quit) {
			return m, tea.Quit
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

// View implements tea.Model.
func (m AppModel) View() string {
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
