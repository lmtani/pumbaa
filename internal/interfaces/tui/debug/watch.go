package debug

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/lmtani/pumbaa/internal/interfaces/tui/debug/tree"
)

// watchInterval is how often watch mode re-fetches the workflow metadata.
const watchInterval = 30 * time.Second

type watchTickMsg struct{}

type watchMetadataLoadedMsg struct {
	metadata *WorkflowMetadata
}

type watchErrorMsg struct {
	err error
}

func watchTick() tea.Cmd {
	return tea.Tick(watchInterval, func(time.Time) tea.Msg {
		return watchTickMsg{}
	})
}

// refreshWorkflowMetadata re-fetches the root workflow metadata from the server.
func (m Model) refreshWorkflowMetadata() tea.Cmd {
	workflowID := m.metadata.ID
	return func() tea.Msg {
		ctx := context.Background()
		data, err := m.fetcher.GetRawMetadataWithOptions(ctx, workflowID, false)
		if err != nil {
			return watchErrorMsg{err: err}
		}
		wf, err := m.fetcher.ParseMetadata(data)
		if err != nil {
			return watchErrorMsg{err: err}
		}
		return watchMetadataLoadedMsg{metadata: wf}
	}
}

// toggleWatch turns watch mode on or off.
func (m Model) toggleWatch() (tea.Model, tea.Cmd) {
	if m.watchActive {
		m.watchActive = false
		m.setStatusMessage("Watch stopped")
		return m, getClearStatusCmd()
	}

	if m.fetcher == nil {
		m.setStatusMessage("Watch requires a server connection (open with --id)")
		return m, getClearStatusCmd()
	}

	m.watchActive = true
	m.watchRefreshing = true
	m.setStatusMessage(fmt.Sprintf("Watching workflow (refresh every %ds)", int(watchInterval.Seconds())))
	return m, tea.Batch(getClearStatusCmd(), m.refreshWorkflowMetadata())
}

// applyRefreshedMetadata swaps in freshly fetched metadata, rebuilding the
// tree while preserving expansion state and cursor position. It returns the
// status transitions since the previous snapshot and the commands needed to
// re-fetch subworkflows that were loaded before the refresh.
func (m *Model) applyRefreshedMetadata(wf *WorkflowMetadata) ([]string, []tea.Cmd) {
	oldStatuses := make(map[string]string)
	expanded := make(map[string]bool)
	loadedSubs := make(map[string]bool)
	for _, node := range flattenTree(m.tree) {
		oldStatuses[node.ID] = node.Status
		if node.Expanded {
			expanded[node.ID] = true
		}
		if node.Type == NodeTypeSubWorkflow && node.CallData != nil && node.CallData.SubWorkflowMetadata != nil {
			loadedSubs[node.ID] = true
		}
	}

	var currentID string
	if m.cursor < len(m.nodes) {
		currentID = m.nodes[m.cursor].ID
	}

	m.metadata = wf
	m.preemption = wf.CalculatePreemptionSummary()
	m.tree = tree.BuildTree(wf)
	// The cached cost breakdown reflected the previous snapshot; drop it so
	// the next open recomputes (and re-fetches subworkflows if needed).
	m.costBreakdown = nil
	m.costError = ""

	var changes []string
	var cmds []tea.Cmd
	for _, node := range flattenTree(m.tree) {
		if expanded[node.ID] {
			node.Expanded = true
		}
		if old, ok := oldStatuses[node.ID]; ok && old != node.Status {
			changes = append(changes, fmt.Sprintf("%s: %s → %s", node.Name, old, node.Status))
		}
		// Re-fetch subworkflows the user had loaded before the refresh
		if loadedSubs[node.ID] && isUnloadedSubWorkflow(node) {
			delete(loadedSubs, node.ID)
			if cmd := m.fetchSubWorkflowMetadata(node); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}
	m.watchRestoreExpanded = expanded
	m.watchReloadSubs = loadedSubs
	m.updateSearchFilter()

	// Keep the cursor on the same node when possible
	if currentID != "" {
		for i, node := range m.nodes {
			if node.ID == currentID {
				m.changeSelectedNode(i)
				break
			}
		}
	}
	m.updateDetailsContent()

	return changes, cmds
}

// restoreWatchStateInSubtree reapplies the pre-refresh expansion snapshot to
// a freshly loaded subworkflow subtree and re-fetches nested subworkflows
// that were loaded before the refresh.
func (m *Model) restoreWatchStateInSubtree(node *TreeNode) []tea.Cmd {
	if m.watchRestoreExpanded == nil && m.watchReloadSubs == nil {
		return nil
	}

	var cmds []tea.Cmd
	for _, child := range flattenTree(node) {
		if m.watchRestoreExpanded[child.ID] {
			child.Expanded = true
		}
		if child != node && m.watchReloadSubs[child.ID] && isUnloadedSubWorkflow(child) {
			delete(m.watchReloadSubs, child.ID)
			if cmd := m.fetchSubWorkflowMetadata(child); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}
	return cmds
}

// watchStatusMessage builds the footer message for a completed refresh.
func watchStatusMessage(changes []string) string {
	if len(changes) == 0 {
		return "↻ Refreshed — no status changes"
	}
	shown := changes
	if len(shown) > 2 {
		shown = shown[:2]
	}
	msg := "↻ " + strings.Join(shown, "; ")
	if extra := len(changes) - len(shown); extra > 0 {
		msg += fmt.Sprintf(" (+%d more)", extra)
	}
	return msg
}
