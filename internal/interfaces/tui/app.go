// Package tui provides the terminal user interface for the application.
package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

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

// AppModel is the root model that coordinates navigation between screens.
//
// Screens are self-contained: they handle their own keys (including ESC) and
// request navigation by emitting the common.Navigate* messages. AppModel only
// switches screens, keeps the navigation stack, and owns the quit flow.
type AppModel struct {
	currentScreen Screen
	stack         []Screen // Screens to return to on NavigateBackMsg

	dashboard    dashboard.Model
	hasDashboard bool // False when started directly on the debug screen
	debug        debug.Model
	chat         *chat.Model

	// chatContextLabel/chatInstruction identify the conversation currently
	// loaded in the chat screen, so re-entering the chat for the same task
	// resumes it instead of starting over.
	chatContextLabel string
	chatInstruction  string

	// debugWorkflow is the workflow currently loaded in the debug screen; it
	// lets navigateToDebug reuse the model (preserving cursor, expansion,
	// search and watch state) when the target workflow hasn't changed.
	debugWorkflow *workflow.Workflow

	width      int
	height     int
	globalKeys common.GlobalKeys
	deps       *Dependencies

	// Quit confirmation modal
	showQuitConfirm bool
}

// NewAppModel creates a new app model with the given dependencies.
func NewAppModel(deps *Dependencies, initialScreen Screen) AppModel {
	m := AppModel{
		currentScreen: initialScreen,
		globalKeys:    common.DefaultGlobalKeys(),
		deps:          deps,
	}

	// Initialize dashboard
	m.dashboard = dashboard.NewModelWithRepository(deps.Repository, deps.CompareUC, deps.CurrentVersion)
	m.hasDashboard = true

	return m
}

// NewAppModelWithWorkflow creates a new app model starting at the debug screen.
func NewAppModelWithWorkflow(deps *Dependencies, wf *workflow.Workflow) AppModel {
	m := AppModel{
		currentScreen: ScreenDebug,
		globalKeys:    common.DefaultGlobalKeys(),
		deps:          deps,
		debugWorkflow: wf,
	}

	m.debug = newDebugModel(deps, wf)
	// Started directly on debug: there is no dashboard to go back to
	m.debug.SetCanGoBack(false)

	return m
}

// newDebugModel builds a debug screen model for the given workflow.
func newDebugModel(deps *Dependencies, wf *workflow.Workflow) debug.Model {
	return debug.NewModelWithChat(
		wf,
		deps.Repository,
		deps.MonitoringUC,
		deps.FileProvider,
		deps.BatchLogsUC,
		convertChatDeps(deps.ChatDeps),
	)
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
		// Only the visible screen needs to re-layout; hidden screens get the
		// size replayed (sizeCmd) when they become current again.
		m.width = msg.Width
		m.height = msg.Height
		return m.updateCurrentScreen(msg)

	case tea.KeyMsg:
		// Handle quit confirmation modal first
		if m.showQuitConfirm {
			return m.handleQuitConfirmKeys(msg)
		}

		// Ctrl+C quits immediately; confirmation is only worth the friction
		// when background work would be interrupted.
		if key.Matches(msg, m.globalKeys.Quit) {
			if m.hasOngoingWork() {
				m.showQuitConfirm = true
				return m, nil
			}
			return m, tea.Quit
		}

	case common.NavigateToDebugMsg:
		return m.navigateToDebug(msg.Workflow)

	case common.NavigateToChatMsg:
		return m.navigateToChat(msg)

	case common.NavigateBackMsg:
		return m.navigateBack()
	}

	// Keys and spinner frames concern only the focused screen. Everything
	// else (async results, timers) is broadcast so hidden screens keep
	// working — e.g. watch mode keeps refreshing while the user is in chat.
	switch msg.(type) {
	case tea.KeyMsg, spinner.TickMsg:
		return m.updateCurrentScreen(msg)
	}
	return m.broadcast(msg)
}

// broadcast forwards a message to every initialized screen. Screen message
// types are package-private, so there is no cross-talk between screens.
func (m AppModel) broadcast(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if m.hasDashboard {
		newModel, cmd := m.dashboard.Update(msg)
		m.dashboard = newModel.(dashboard.Model)
		cmds = append(cmds, cmd)
	}
	if m.debugWorkflow != nil {
		newModel, cmd := m.debug.Update(msg)
		m.debug = newModel.(debug.Model)
		cmds = append(cmds, cmd)
	}
	if m.chat != nil {
		newModel, cmd := m.chat.Update(msg)
		m.chat = newModel.(*chat.Model)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// navigateToDebug switches to the debug screen with the given workflow.
func (m AppModel) navigateToDebug(wf *workflow.Workflow) (tea.Model, tea.Cmd) {
	m.stack = append(m.stack, m.currentScreen)
	m.currentScreen = ScreenDebug

	// Reuse the existing model when the workflow is unchanged so tree
	// expansion, cursor, search and watch state survive the round trip.
	if wf == m.debugWorkflow {
		return m, tea.Batch(m.sizeCmd(), m.debug.ResumeCmd())
	}

	m.debugWorkflow = wf
	m.debug = newDebugModel(m.deps, wf)

	return m, tea.Batch(m.debug.Init(), m.sizeCmd())
}

// navigateToChat switches to the chat screen. The chat session is created
// asynchronously so the UI never blocks on network or disk.
//
// Re-entering the chat for the same task (same context label) resumes the
// live conversation instead of starting a new one; freshly collected context
// only replaces the system instruction.
func (m AppModel) navigateToChat(msg common.NavigateToChatMsg) (tea.Model, tea.Cmd) {
	if !m.deps.HasChat() {
		// Screens guard against this before emitting the message; ignore.
		return m, nil
	}

	m.stack = append(m.stack, m.currentScreen)
	m.currentScreen = ScreenChat

	label := msg.ContextLabel
	if label == "" {
		label = "Task Context"
	}

	// Same task: resume the existing conversation
	if m.chat != nil && msg.ContextLabel != "" && msg.ContextLabel == m.chatContextLabel {
		if msg.SystemInstruction != m.chatInstruction {
			m.chatInstruction = msg.SystemInstruction
			m.chat.SetSystemInstruction(msg.SystemInstruction)
			m.chat.AppendNotice("Context refreshed for " + label)
		}
		return m, tea.Batch(m.sizeCmd(), m.chat.ResumeCmd())
	}

	chatModel := chat.NewModel(
		m.deps.ChatDeps.LLM,
		m.deps.ChatDeps.Tools,
		msg.SystemInstruction,
		m.deps.ChatDeps.SessionSvc,
		nil, // the chat creates its session lazily on the first message
	)
	m.chat = &chatModel
	m.chatContextLabel = msg.ContextLabel
	m.chatInstruction = msg.SystemInstruction

	if m.deps.Program != nil {
		// Enables streaming and tool records pushed from the generation goroutine
		m.chat.SetProgram(m.deps.Program)
	}
	m.chat.SetContextLabel(label)
	if msg.ContextSummary != "" {
		m.chat.AddInfoMessage(msg.ContextSummary)
	}

	return m, tea.Batch(m.chat.Init(), m.sizeCmd())
}

// navigateBack pops the navigation stack; at the root it starts the quit flow.
func (m AppModel) navigateBack() (tea.Model, tea.Cmd) {
	if len(m.stack) == 0 {
		m.showQuitConfirm = true
		return m, nil
	}

	m.currentScreen = m.stack[len(m.stack)-1]
	m.stack = m.stack[:len(m.stack)-1]

	// Returning screens kept their state; they only need the current size
	// and, if they were mid-load, a fresh spinner tick.
	var resume tea.Cmd
	switch m.currentScreen {
	case ScreenDashboard:
		resume = m.dashboard.ResumeCmd()
	case ScreenDebug:
		resume = m.debug.ResumeCmd()
	}
	return m, tea.Batch(m.sizeCmd(), resume)
}

// sizeCmd replays the last known window size to the (new) current screen.
func (m AppModel) sizeCmd() tea.Cmd {
	if m.width == 0 || m.height == 0 {
		return nil
	}
	width, height := m.width, m.height
	return func() tea.Msg {
		return tea.WindowSizeMsg{Width: width, Height: height}
	}
}

// hasOngoingWork reports whether quitting now would interrupt background
// work on any screen — hidden screens keep working after navigation.
func (m AppModel) hasOngoingWork() bool {
	if m.debugWorkflow != nil && m.debug.HasOngoingWork() {
		return true
	}
	return m.chat != nil && m.chat.IsBusy()
}

// updateCurrentScreen delegates the update to the current screen.
func (m AppModel) updateCurrentScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.currentScreen {
	case ScreenDashboard:
		newModel, cmd := m.dashboard.Update(msg)
		m.dashboard = newModel.(dashboard.Model)
		return m, cmd

	case ScreenDebug:
		newModel, cmd := m.debug.Update(msg)
		m.debug = newModel.(debug.Model)
		return m, cmd

	case ScreenChat:
		if m.chat == nil {
			return m, nil
		}
		newModel, cmd := m.chat.Update(msg)
		m.chat = newModel.(*chat.Model)
		return m, cmd
	}

	return m, nil
}

// handleQuitConfirmKeys handles key presses when quit confirmation modal is shown.
func (m AppModel) handleQuitConfirmKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y", "ctrl+c":
		return m, tea.Quit
	case "n", "N", "esc":
		m.showQuitConfirm = false
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

	body := "Are you sure you want to exit?"
	if m.hasOngoingWork() {
		body = "Background work is still running.\nQuit anyway?"
	}

	// Create the modal
	modalContent := common.TitleStyle.Render("Quit Pumbaa?") + "\n\n" +
		body + "\n\n" +
		common.KeyStyle.Render("[Y]") + " " + common.DescStyle.Render("Yes, quit") + "    " +
		common.KeyStyle.Render("[N]") + " " + common.DescStyle.Render("No, stay")

	modal := common.ModalStyle.
		Width(40).
		Render(modalContent)

	// Center the modal
	modalWidth := 44 // modal width + border
	modalHeight := 7 // approximate modal height

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
