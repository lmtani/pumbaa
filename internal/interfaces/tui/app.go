// Package tui provides the terminal user interface for the application.
package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/dashboard"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/debug"
)

// Screen represents the different screens available in the TUI.
type Screen int

const (
	ScreenDashboard Screen = iota
	ScreenDebug
)

// AppModel is the root model that coordinates navigation between screens.
type AppModel struct {
	currentScreen Screen
	dashboard     dashboard.Model
	debug         debug.Model
	width         int
	height        int
	globalKeys    common.GlobalKeys
}

// Init implements tea.Model.
func (m AppModel) Init() tea.Cmd {
	switch m.currentScreen {
	case ScreenDashboard:
		return m.dashboard.Init()
	case ScreenDebug:
		return m.debug.Init()
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
	}

	return m.updateCurrentScreen(msg)
}

// updateCurrentScreen delegates the update to the current screen.
func (m AppModel) updateCurrentScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.currentScreen {
	case ScreenDashboard:
		newDash, cmd := m.dashboard.Update(msg)
		m.dashboard = newDash.(dashboard.Model)
		return m, cmd

	case ScreenDebug:
		newDebug, cmd := m.debug.Update(msg)
		m.debug = newDebug.(debug.Model)
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
	}
	return ""
}

// NavigateToDebugMsg is a message to navigate to the debug screen.
type NavigateToDebugMsg struct {
	Metadata *debug.WorkflowMetadata
	Fetcher  debug.MetadataFetcher
}

// NavigateToDebug creates a command to navigate to the debug screen.
func NavigateToDebug(metadata *debug.WorkflowMetadata, fetcher debug.MetadataFetcher) tea.Cmd {
	return func() tea.Msg {
		return NavigateToDebugMsg{Metadata: metadata, Fetcher: fetcher}
	}
}
