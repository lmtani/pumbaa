package debug

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/lmtani/pumbaa/internal/domain/workflow"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

// Cache source navigation (o key): a cache-hit call points at the call that
// actually produced its results. Following it opens that workflow as a new
// debug screen, with the cursor on the producing call; ESC comes back here.

type cacheSourceLoadedMsg struct {
	workflow *WorkflowMetadata
	source   workflow.CacheSource
	from     string // label of the call the jump started from
}

type cacheSourceErrorMsg struct {
	source workflow.CacheSource
	err    error
}

// cacheSourceOf returns where the node's results were copied from, when the
// node is a cache hit with a resolvable pointer.
func cacheSourceOf(node *TreeNode) (workflow.CacheSource, bool) {
	if node == nil || node.CallData == nil || !node.CallData.CacheHit {
		return workflow.CacheSource{}, false
	}
	return workflow.ParseCacheResult(node.CallData.CacheResult)
}

// openCacheSource starts the jump to the call that produced the selected
// node's cached results.
func (m Model) openCacheSource(node *TreeNode) (tea.Model, tea.Cmd) {
	src, ok := cacheSourceOf(node)
	if !ok {
		m.setStatusMessage("Not a cache hit: this task ran here")
		return m, getClearStatusCmd()
	}

	// A source inside this same workflow needs no fetch, only a cursor move.
	if src.WorkflowID == m.metadata.ID {
		if !m.FocusCall(src.CallName, src.ShardIndex) {
			m.setStatusMessage("Cache source not found in this workflow")
			return m, getClearStatusCmd()
		}
		m.setStatusMessage("Jumped to the cache source")
		return m, getClearStatusCmd()
	}

	if m.fetcher == nil {
		m.setStatusMessage("Opening the cache source needs a server connection (open with --id)")
		return m, getClearStatusCmd()
	}

	m.isLoading = true
	m.loadingMessage = "Opening cache source " + shortID(src.WorkflowID) + "..."
	m.loadingStartTime = time.Now()
	return m, tea.Batch(m.loadingSpinner.Tick, m.fetchCacheSource(src, node.Name))
}

func (m Model) fetchCacheSource(src workflow.CacheSource, from string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		data, err := m.fetcher.GetRawMetadataWithOptions(ctx, src.WorkflowID, false)
		if err != nil {
			return cacheSourceErrorMsg{source: src, err: err}
		}
		wf, err := m.fetcher.ParseMetadata(data)
		if err != nil {
			return cacheSourceErrorMsg{source: src, err: err}
		}
		return cacheSourceLoadedMsg{workflow: wf, source: src, from: from}
	}
}

// handleCacheSourceLoaded hands the source workflow to the app model, which
// stacks a new debug screen on top of this one.
func (m Model) handleCacheSourceLoaded(msg cacheSourceLoadedMsg) (tea.Model, tea.Cmd) {
	m.isLoading = false
	m.loadingMessage = ""
	return m, common.NavigateCmd(common.NavigateToDebugMsg{
		Workflow: msg.workflow,
		Focus: &common.CallFocus{
			CallName: msg.source.CallName,
			Shard:    msg.source.ShardIndex,
		},
		Origin: fmt.Sprintf("%s (%s)", msg.from, m.metadata.Name),
	})
}

func (m Model) handleCacheSourceError(msg cacheSourceErrorMsg) (tea.Model, tea.Cmd) {
	m.isLoading = false
	m.loadingMessage = ""
	m.lastError = msg.err.Error()
	m.setStatusMessage(fmt.Sprintf("Cannot open cache source %s: %s", shortID(msg.source.WorkflowID), common.Truncate(msg.err.Error(), 50)))
	return m, getClearStatusCmd()
}

// FocusCall selects the call with the given fully-qualified name and shard,
// expanding its ancestors. It reports whether the call was found.
func (m *Model) FocusCall(callName string, shard int) bool {
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
	if m.searchQuery != "" {
		m.searchQuery = ""
	}
	m.updateSearchFilter()
	for i, node := range m.nodes {
		if node == target {
			m.changeSelectedNode(i)
			return true
		}
	}
	return false
}

// SetOrigin marks this screen as opened by following a cache hit, so the
// header can say where ESC goes back to.
func (m *Model) SetOrigin(origin string) {
	m.origin = origin
}

// Suspend pauses watch mode while the screen sits under another debug
// screen; hidden debug screens stop receiving async messages.
func (m *Model) Suspend() {
	if m.watchActive {
		m.watchActive = false
		m.watchRefreshing = false
		m.watchSuspended = true
	}
}

// Reactivate resumes what Suspend paused, refreshing right away since the
// snapshot may be stale.
func (m *Model) Reactivate() tea.Cmd {
	resume := m.ResumeCmd()
	if !m.watchSuspended {
		return resume
	}
	m.watchSuspended = false
	m.watchActive = true
	m.watchRefreshing = true
	return tea.Batch(resume, m.refreshWorkflowMetadata())
}

func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

// cacheSourceCallLabel renders a cache source call as "Task" or
// "Task · shard N", dropping the workflow prefix of the FQN.
func cacheSourceCallLabel(src workflow.CacheSource) string {
	name := src.CallName
	if i := strings.LastIndex(name, "."); i >= 0 {
		name = name[i+1:]
	}
	if src.ShardIndex >= 0 {
		return fmt.Sprintf("%s · shard %d", name, src.ShardIndex)
	}
	return name
}

// SetStatus shows a temporary message in the footer.
func (m *Model) SetStatus(message string) tea.Cmd {
	m.setStatusMessage(message)
	return getClearStatusCmd()
}
