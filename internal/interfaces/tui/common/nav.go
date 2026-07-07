package common

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/lmtani/pumbaa/internal/domain/workflow"
)

// Navigation messages shared by all TUI screens.
//
// Screens emit these messages through commands (see NavigateCmd); the root
// AppModel is the only handler. This keeps navigation unidirectional: screens
// never know about each other, and the app model never inspects screen
// internals to decide where to go.

// NavigateToDebugMsg requests navigation to the debug screen for a workflow.
type NavigateToDebugMsg struct {
	Workflow *workflow.Workflow
}

// NavigateToChatMsg requests navigation to the chat screen.
type NavigateToChatMsg struct {
	SystemInstruction string
	ContextSummary    string
	ContextLabel      string // Shown as a badge in the chat header (e.g. "wf ▸ task")
}

// NavigateBackMsg asks the root model to leave the current screen. A screen
// sends it only when it has nothing left to close itself (no modal, search,
// or alternate view mode); at the root screen it triggers the quit flow.
type NavigateBackMsg struct{}

// NavigateCmd wraps a navigation message in a command.
func NavigateCmd(msg tea.Msg) tea.Cmd {
	return func() tea.Msg { return msg }
}
