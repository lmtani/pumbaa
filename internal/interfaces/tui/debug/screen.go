package debug

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

// Screen lifecycle as seen by the app model: a debug screen is opened by a
// navigation request (Arrive), may be parked under another debug screen
// (Suspend) and brought back with ESC (Reactivate).

// Arrive prepares a freshly built screen for the navigation that opened it:
// places the cursor on the requested call, remembers where ESC returns to,
// shows the notice and starts resolving the selected cache hit's lineage.
func (m *Model) Arrive(msg common.NavigateToDebugMsg) tea.Cmd {
	m.origin = msg.Origin
	notice := msg.Notice
	if msg.Focus != nil && !m.focusCall(msg.Focus.CallName, msg.Focus.Shard) {
		notice = "Task not found in this workflow"
	}

	cmds := []tea.Cmd{m.ensureSelectedLineage()}
	if notice != "" {
		m.setStatusMessage(notice)
		cmds = append(cmds, getClearStatusCmd())
	}
	return tea.Batch(cmds...)
}

// Suspend pauses watch mode while the screen sits under another debug
// screen: parked screens receive no async messages, so the refresh chain
// would die anyway.
func (m *Model) Suspend() {
	if m.watchActive {
		m.watchActive = false
		m.watchRefreshing = false
		m.watchSuspended = true
	}
}

// Reactivate resumes what Suspend paused, refreshing right away since the
// snapshot may be stale, and re-requests lineages whose answers went to the
// screen on top (the resolver memoizes, so this is cheap).
func (m *Model) Reactivate() tea.Cmd {
	m.lineage.forgetPending()
	cmds := []tea.Cmd{m.ResumeCmd(), m.ensureSelectedLineage()}

	if m.watchSuspended {
		m.watchSuspended = false
		m.watchActive = true
		m.watchRefreshing = true
		cmds = append(cmds, m.refreshWorkflowMetadata())
	}
	return tea.Batch(cmds...)
}

// focusCall selects the call with the given fully-qualified name and shard,
// expanding its ancestors. It reports whether the call was found.
func (m *Model) focusCall(callName string, shard int) bool {
	if m.tree == nil {
		return false
	}
	var target *TreeNode
	for _, node := range flattenTree(m.tree) {
		cd := node.CallData
		if cd == nil || cd.Name != callName || cd.ShardIndex != shard {
			continue
		}
		// Prefer the leaf over a scatter parent with the same call name
		if target == nil || len(node.Children) == 0 {
			target = node
		}
	}
	if target == nil {
		return false
	}

	for p := target.Parent; p != nil; p = p.Parent {
		p.Expanded = true
	}
	// The target must be visible: a search filter could hide it
	m.searchQuery = ""
	m.updateSearchFilter()
	for i, node := range m.nodes {
		if node == target {
			m.changeSelectedNode(i)
			return true
		}
	}
	return false
}

func (m Model) selectedNode() *TreeNode {
	if m.cursor < len(m.nodes) {
		return m.nodes[m.cursor]
	}
	return nil
}

func (m *Model) startLoading(message string) {
	m.isLoading = true
	m.loadingMessage = message
	m.loadingStartTime = time.Now()
}
