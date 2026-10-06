package debug

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	workflowapp "github.com/lmtani/pumbaa/internal/application/workflow"
	"github.com/lmtani/pumbaa/internal/domain/workflow"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

// Cache lineage: a cache-hit call points at the run it copied its results
// from, which may itself be a cache hit pointing further back. The whole
// chain is resolved in the background as soon as a cache hit is selected, so
// the details panel can show it (view_cache.go), o can jump straight to the
// run that actually executed the call, and O can list every run of the chain
// (modal_lineage.go). Each jump stacks a debug screen; ESC comes back.

// lineageEntry is the resolution state of one cache source.
type lineageEntry struct {
	pending bool
	lineage workflow.CacheLineage
}

type lineageResolvedMsg struct {
	source  workflow.CacheSource
	lineage workflow.CacheLineage
}

// lineageTracker owns the lineages known to a debug screen: which sources
// were requested, their answers, and a jump waiting for its answer.
type lineageTracker struct {
	resolver *workflowapp.CacheLineageUseCase
	entries  map[workflow.CacheSource]*lineageEntry

	// awaiting is the source whose jump (o) waits for its lineage.
	awaiting    workflow.CacheSource
	hasAwaiting bool
}

func (t *lineageTracker) enabled() bool {
	return t.resolver != nil
}

// get returns the resolution state of src, nil when never requested.
func (t *lineageTracker) get(src workflow.CacheSource) *lineageEntry {
	return t.entries[src]
}

// request starts resolving src unless it is known or already in flight.
func (t *lineageTracker) request(src workflow.CacheSource) tea.Cmd {
	if !t.enabled() || t.entries[src] != nil {
		return nil
	}
	if t.entries == nil {
		t.entries = make(map[workflow.CacheSource]*lineageEntry)
	}
	t.entries[src] = &lineageEntry{pending: true}

	resolver := t.resolver
	return func() tea.Msg {
		return lineageResolvedMsg{source: src, lineage: resolver.Resolve(context.Background(), src)}
	}
}

// store records an answer and reports whether a jump was waiting for it.
func (t *lineageTracker) store(msg lineageResolvedMsg) (jump bool) {
	if t.entries == nil {
		t.entries = make(map[workflow.CacheSource]*lineageEntry)
	}
	t.entries[msg.source] = &lineageEntry{lineage: msg.lineage}
	if t.hasAwaiting && t.awaiting == msg.source {
		t.hasAwaiting = false
		return true
	}
	return false
}

func (t *lineageTracker) await(src workflow.CacheSource) {
	t.awaiting, t.hasAwaiting = src, true
}

// forgetPending drops requests whose answers went to another screen (the one
// stacked on top got the broadcast), so they can be requested again.
func (t *lineageTracker) forgetPending() {
	for src, entry := range t.entries {
		if entry.pending {
			delete(t.entries, src)
		}
	}
	t.hasAwaiting = false
}

// SetCacheLineage attaches the resolver that follows cache hits. Optional:
// without it cache hits are described but cannot be followed.
func (m *Model) SetCacheLineage(uc *workflowapp.CacheLineageUseCase) {
	m.lineage.resolver = uc
}

// cacheSourceOf returns where the node's results were copied from, when the
// node is a cache hit with a resolvable pointer.
func cacheSourceOf(node *TreeNode) (workflow.CacheSource, bool) {
	if node == nil || node.CallData == nil || !node.CallData.CacheHit {
		return workflow.CacheSource{}, false
	}
	return workflow.ParseCacheResult(node.CallData.CacheResult)
}

// ensureSelectedLineage starts resolving the lineage of the selected node
// when it is a cache hit not requested yet.
func (m *Model) ensureSelectedLineage() tea.Cmd {
	src, ok := cacheSourceOf(m.selectedNode())
	if !ok {
		return nil
	}
	return m.lineage.request(src)
}

func (m Model) handleLineageResolved(msg lineageResolvedMsg) (tea.Model, tea.Cmd) {
	jump := m.lineage.store(msg)
	if msg.lineage.Err != nil {
		m.lastError = msg.lineage.Err.Error()
	}
	m.updateDetailsContent()

	if !jump {
		return m, nil
	}
	m.isLoading = false
	m.loadingMessage = ""
	return m.openFurthestHop(msg.source)
}

// openOriginal (o) jumps to the run that actually executed the selected
// cache hit, however many runs back it is.
func (m Model) openOriginal(node *TreeNode) (tea.Model, tea.Cmd) {
	src, ok := cacheSourceOf(node)
	if !ok {
		m.setStatusMessage("Not a cache hit: this task ran here")
		return m, getClearStatusCmd()
	}
	if !m.lineage.enabled() {
		m.setStatusMessage("Following cache hits needs a server connection")
		return m, getClearStatusCmd()
	}

	if entry := m.lineage.get(src); entry != nil && !entry.pending {
		return m.openFurthestHop(src)
	}
	cmd := m.lineage.request(src)
	m.lineage.await(src)
	m.startLoading("Following cache hits of " + node.Name + "...")
	return m, tea.Batch(m.loadingSpinner.Tick, cmd)
}

// openFurthestHop opens the producing run, or the furthest run reached when
// the chain is broken.
func (m Model) openFurthestHop(src workflow.CacheSource) (tea.Model, tea.Cmd) {
	lineage := m.lineage.get(src).lineage
	if len(lineage.Hops) == 0 {
		m.setStatusMessage("Cannot follow cache hit: " + common.Truncate(errText(lineage.Err), 60))
		return m, getClearStatusCmd()
	}
	return m.openHop(src, len(lineage.Hops)-1)
}

// openHop opens hop idx of src's lineage on a stacked debug screen, with the
// cursor on the call. A hop in this same workflow only moves the cursor.
func (m Model) openHop(src workflow.CacheSource, idx int) (tea.Model, tea.Cmd) {
	lineage := m.lineage.get(src).lineage
	hop := lineage.Hops[idx]
	notice := common.IconCached + " " + hopNotice(lineage, idx)

	if hop.Workflow.ID == m.metadata.ID {
		if !m.focusCall(hop.Source.CallName, hop.Source.ShardIndex) {
			notice = "Cache source not found in this workflow"
		}
		m.setStatusMessage(notice)
		return m, getClearStatusCmd()
	}

	return m, common.NavigateCmd(common.NavigateToDebugMsg{
		Workflow: hop.Workflow,
		Focus:    &common.CallFocus{CallName: hop.Source.CallName, Shard: hop.Source.ShardIndex},
		Origin:   fmt.Sprintf("%s in %s", callLabel(src.CallName, src.ShardIndex), shortID(m.metadata.ID)),
		Notice:   notice + " — esc goes back",
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
