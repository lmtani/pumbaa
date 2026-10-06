package debug

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	workflowapp "github.com/lmtani/pumbaa/internal/application/workflow"
	"github.com/lmtani/pumbaa/internal/domain/workflow"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

// Cache source navigation: a cache-hit call points at the run it copied its
// results from, which may itself be a cache hit pointing further back. The
// whole chain (the lineage) is resolved in the background as soon as a cache
// hit is selected, so the details panel can show it and o can jump straight
// to the run that actually executed the call. O lists every run of the chain
// to open any of them. Each jump stacks a debug screen; ESC comes back.

// lineageEntry is the resolution state of one cache source.
type lineageEntry struct {
	pending bool
	lineage workflow.CacheLineage
}

type lineageResolvedMsg struct {
	source  workflow.CacheSource
	lineage workflow.CacheLineage
}

// SetCacheLineage attaches the resolver that follows cache hits. Optional:
// without it cache hits are marked but cannot be followed.
func (m *Model) SetCacheLineage(uc *workflowapp.CacheLineageUseCase) {
	m.lineageUC = uc
}

// cacheSourceOf returns where the node's results were copied from, when the
// node is a cache hit with a resolvable pointer.
func cacheSourceOf(node *TreeNode) (workflow.CacheSource, bool) {
	if node == nil || node.CallData == nil || !node.CallData.CacheHit {
		return workflow.CacheSource{}, false
	}
	return workflow.ParseCacheResult(node.CallData.CacheResult)
}

func (m Model) selectedNode() *TreeNode {
	if m.cursor < len(m.nodes) {
		return m.nodes[m.cursor]
	}
	return nil
}

// lineageFor returns the resolution state of src, nil when never requested.
func (m Model) lineageFor(src workflow.CacheSource) *lineageEntry {
	return m.lineages[src]
}

// ensureSelectedLineage starts resolving the lineage of the selected node
// when it is a cache hit not resolved yet.
func (m *Model) ensureSelectedLineage() tea.Cmd {
	src, ok := cacheSourceOf(m.selectedNode())
	if !ok {
		return nil
	}
	return m.resolveLineage(src)
}

func (m *Model) resolveLineage(src workflow.CacheSource) tea.Cmd {
	if m.lineageUC == nil || m.lineages[src] != nil {
		return nil
	}
	if m.lineages == nil {
		m.lineages = make(map[workflow.CacheSource]*lineageEntry)
	}
	m.lineages[src] = &lineageEntry{pending: true}

	uc := m.lineageUC
	return func() tea.Msg {
		return lineageResolvedMsg{source: src, lineage: uc.Execute(context.Background(), src)}
	}
}

func (m Model) handleLineageResolved(msg lineageResolvedMsg) (tea.Model, tea.Cmd) {
	if m.lineages == nil {
		m.lineages = make(map[workflow.CacheSource]*lineageEntry)
	}
	m.lineages[msg.source] = &lineageEntry{lineage: msg.lineage}
	if msg.lineage.Err != nil {
		m.lastError = msg.lineage.Err.Error()
	}
	m.updateDetailsContent()

	// o was pressed before the chain was known: finish the jump now
	if m.lineageOpenPending != nil && *m.lineageOpenPending == msg.source {
		m.lineageOpenPending = nil
		m.isLoading = false
		m.loadingMessage = ""
		return m.openFurthestHop(msg.source)
	}
	return m, nil
}

// openOriginal (o) jumps to the run that actually executed the selected
// cache hit, however many runs back it is.
func (m Model) openOriginal(node *TreeNode) (tea.Model, tea.Cmd) {
	src, ok := cacheSourceOf(node)
	if !ok {
		m.setStatusMessage("Not a cache hit: this task ran here")
		return m, getClearStatusCmd()
	}
	if m.lineageUC == nil {
		m.setStatusMessage("Following cache hits needs a server connection")
		return m, getClearStatusCmd()
	}

	entry := m.lineageFor(src)
	if entry == nil || entry.pending {
		cmd := m.resolveLineage(src)
		m.lineageOpenPending = &src
		m.startLoading("Following cache hits of " + node.Name + "...")
		return m, tea.Batch(m.loadingSpinner.Tick, cmd)
	}
	return m.openFurthestHop(src)
}

// openFurthestHop opens the producing run, or the furthest run reached when
// the chain is broken (saying why).
func (m Model) openFurthestHop(src workflow.CacheSource) (tea.Model, tea.Cmd) {
	lineage := m.lineageFor(src).lineage
	if len(lineage.Hops) == 0 {
		m.setStatusMessage("Cannot follow cache hit: " + common.Truncate(errText(lineage.Err), 60))
		return m, getClearStatusCmd()
	}
	return m.openHop(src, len(lineage.Hops)-1)
}

// openHop opens hop idx of src's lineage on a stacked debug screen, with the
// cursor on the call. A hop in this same workflow only moves the cursor.
func (m Model) openHop(src workflow.CacheSource, idx int) (tea.Model, tea.Cmd) {
	lineage := m.lineageFor(src).lineage
	hop := lineage.Hops[idx]
	notice := common.IconCached + " " + hopNotice(lineage, idx) + " — esc goes back"

	if hop.Workflow.ID == m.metadata.ID {
		if !m.FocusCall(hop.Source.CallName, hop.Source.ShardIndex) {
			m.setStatusMessage("Cache source not found in this workflow")
		} else {
			m.setStatusMessage(common.IconCached + " " + hopNotice(lineage, idx))
		}
		return m, getClearStatusCmd()
	}

	return m, common.NavigateCmd(common.NavigateToDebugMsg{
		Workflow: hop.Workflow,
		Focus:    &common.CallFocus{CallName: hop.Source.CallName, Shard: hop.Source.ShardIndex},
		Origin:   fmt.Sprintf("%s in %s", callLabel(src.CallName, src.ShardIndex), shortID(m.metadata.ID)),
		Notice:   notice,
	})
}

// hopNotice describes where a hop sits in the chain, e.g. "Producing run,
// 2 runs back" or "1 run back, also a cache hit".
func hopNotice(lineage workflow.CacheLineage, idx int) string {
	hop := lineage.Hops[idx]
	back := runsBack(idx + 1)
	switch {
	case hop.Ran():
		return "Producing run, " + back
	case idx == len(lineage.Hops)-1 && lineage.Err != nil:
		return "Furthest run reached, " + back + " (chain broken: " + common.Truncate(lineage.Err.Error(), 40) + ")"
	default:
		return back + ", also a cache hit"
	}
}

func runsBack(n int) string {
	if n == 1 {
		return "1 run back"
	}
	return fmt.Sprintf("%d runs back", n)
}

func errText(err error) string {
	if err == nil {
		return "unknown reason"
	}
	return err.Error()
}

func (m *Model) startLoading(message string) {
	m.isLoading = true
	m.loadingMessage = message
	m.loadingStartTime = time.Now()
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
// snapshot may be stale. Lineages still pending were answered while another
// screen was on top, so they are requested again (the resolver memoizes, so
// this is cheap).
func (m *Model) Reactivate() tea.Cmd {
	cmds := []tea.Cmd{m.ResumeCmd()}
	for src, entry := range m.lineages {
		if entry.pending {
			delete(m.lineages, src)
		}
	}
	m.lineageOpenPending = nil
	cmds = append(cmds, m.ensureSelectedLineage())

	if m.watchSuspended {
		m.watchSuspended = false
		m.watchActive = true
		m.watchRefreshing = true
		cmds = append(cmds, m.refreshWorkflowMetadata())
	}
	return tea.Batch(cmds...)
}

// InitialCmd is what a freshly opened screen needs beyond Init once the
// cursor has been placed (resolving the selected cache hit's lineage).
func (m *Model) InitialCmd() tea.Cmd {
	return m.ensureSelectedLineage()
}

func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

// callLabel renders a call as "Task" or "Task · shard N", dropping the
// workflow prefix of the FQN.
func callLabel(fqn string, shard int) string {
	name := fqn
	if i := strings.LastIndex(name, "."); i >= 0 {
		name = name[i+1:]
	}
	if shard >= 0 {
		return fmt.Sprintf("%s · shard %d", name, shard)
	}
	return name
}

// SetStatus shows a temporary message in the footer.
func (m *Model) SetStatus(message string) tea.Cmd {
	m.setStatusMessage(message)
	return getClearStatusCmd()
}
