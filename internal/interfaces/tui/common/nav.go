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

	// Focus, when set, places the cursor on that call once the screen opens.
	Focus *CallFocus

	// Origin is set when the request comes from another debug screen (e.g.
	// following a cache hit to its source). The new screen is stacked on top
	// of the current one, and ESC returns to it with its state intact.
	Origin string

	// Notice is shown in the footer when the screen opens.
	Notice string
}

// CallFocus identifies a call within a workflow: its fully-qualified name and
// shard index (-1 when not scattered).
type CallFocus struct {
	CallName string
	Shard    int
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
