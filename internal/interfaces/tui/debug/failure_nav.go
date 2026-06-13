package debug

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// isFailedStatus reports whether a node status counts as a terminal failure.
// RetryableFailure/Preempted are excluded: those attempts were retried.
func isFailedStatus(status string) bool {
	return status == "Failed"
}

// wasRetried reports whether a node went through preemption or any retry:
// either its status is a retryable failure or it took more than one attempt.
func wasRetried(node *TreeNode) bool {
	if node.Status == "Preempted" || node.Status == "RetryableFailure" {
		return true
	}
	if node.CallData == nil {
		return false
	}
	if node.CallData.Attempt > 1 {
		return true
	}
	status := string(node.CallData.Status)
	return status == "Preempted" || status == "RetryableFailure"
}

// isProblemNode reports whether a node is worth surfacing when expanding
// with f: terminal failures and preempted/retried tasks.
func isProblemNode(node *TreeNode) bool {
	return isFailedStatus(node.Status) || wasRetried(node)
}

// countFailedLeaves counts failed leaf nodes (tasks, shards or unloaded
// subworkflows) in the subtree rooted at node.
func countFailedLeaves(node *TreeNode) int {
	if len(node.Children) == 0 {
		if isFailedStatus(node.Status) {
			return 1
		}
		return 0
	}
	count := 0
	for _, child := range node.Children {
		count += countFailedLeaves(child)
	}
	return count
}

// countRetriedLeaves counts leaf nodes that were preempted/retried but did
// not end up failed (those are already counted by countFailedLeaves).
func countRetriedLeaves(node *TreeNode) int {
	if len(node.Children) == 0 {
		if wasRetried(node) && !isFailedStatus(node.Status) {
			return 1
		}
		return 0
	}
	count := 0
	for _, child := range node.Children {
		count += countRetriedLeaves(child)
	}
	return count
}

// expandSummaryMessage builds the footer message after a failure expansion.
func expandSummaryMessage(failed, retried int) string {
	var parts []string
	if failed > 0 {
		parts = append(parts, fmt.Sprintf("%d failed", failed))
	}
	if retried > 0 {
		parts = append(parts, fmt.Sprintf("%d preempted/retried", retried))
	}
	return strings.Join(parts, ", ") + " task(s)"
}

// isUnloadedSubWorkflow reports whether node is a subworkflow whose metadata
// has not been fetched yet.
func isUnloadedSubWorkflow(node *TreeNode) bool {
	return node.Type == NodeTypeSubWorkflow &&
		len(node.Children) == 0 &&
		node.SubWorkflowID != "" &&
		(node.CallData == nil || node.CallData.SubWorkflowMetadata == nil)
}

// expandFailurePaths expands every node whose subtree contains a failure or
// a preempted/retried task and collapses every other expandable node. It
// returns the matching subworkflow nodes whose metadata still needs to be
// fetched.
func expandFailurePaths(node *TreeNode) (hasProblem bool, unloaded []*TreeNode) {
	selfProblem := isProblemNode(node)
	childHasProblem := false

	for _, child := range node.Children {
		childProblem, childUnloaded := expandFailurePaths(child)
		childHasProblem = childHasProblem || childProblem
		unloaded = append(unloaded, childUnloaded...)
	}

	if len(node.Children) > 0 {
		node.Expanded = childHasProblem
	}

	if selfProblem && isUnloadedSubWorkflow(node) {
		unloaded = append(unloaded, node)
	}

	return selfProblem || childHasProblem, unloaded
}

// expandToFailures collapses the tree down to only the paths that lead to
// failed or preempted/retried calls, fetching matching subworkflows that
// were not loaded yet.
func (m *Model) expandToFailures() tea.Cmd {
	if m.tree == nil {
		return nil
	}
	if countFailedLeaves(m.tree) == 0 && countRetriedLeaves(m.tree) == 0 {
		m.setStatusMessage("No failed or preempted tasks found")
		return getClearStatusCmd()
	}

	_, unloaded := expandFailurePaths(m.tree)
	m.updateSearchFilter()
	m.moveCursorToFirstFailure()

	cmds := m.fetchFailedSubWorkflows(unloaded)
	if len(cmds) > 0 {
		m.failureExpandActive = true
		m.setStatusMessage(fmt.Sprintf("Loading %d subworkflow(s)...", len(cmds)))
		return tea.Batch(cmds...)
	}

	m.setStatusMessage(expandSummaryMessage(countFailedLeaves(m.tree), countRetriedLeaves(m.tree)))
	return getClearStatusCmd()
}

// fetchFailedSubWorkflows dispatches metadata fetches for failed subworkflows
// that were not requested yet during the current expansion.
func (m *Model) fetchFailedSubWorkflows(nodes []*TreeNode) []tea.Cmd {
	if m.fetcher == nil {
		return nil
	}
	if m.failureFetchRequested == nil {
		m.failureFetchRequested = make(map[string]bool)
	}

	var cmds []tea.Cmd
	for _, node := range nodes {
		if m.failureFetchRequested[node.ID] {
			continue
		}
		m.failureFetchRequested[node.ID] = true
		if cmd := m.fetchSubWorkflowMetadata(node); cmd != nil {
			cmds = append(cmds, cmd)
			m.failureExpandPending++
		}
	}
	return cmds
}

// continueFailureExpansion re-runs failure-path expansion after a subworkflow
// fetch finishes, chaining fetches for failed subworkflows revealed by it.
func (m *Model) continueFailureExpansion() tea.Cmd {
	m.failureExpandPending--

	_, unloaded := expandFailurePaths(m.tree)
	m.updateSearchFilter()

	cmds := m.fetchFailedSubWorkflows(unloaded)
	if m.failureExpandPending > 0 || len(cmds) > 0 {
		return tea.Batch(cmds...)
	}

	m.failureExpandActive = false
	m.failureFetchRequested = nil
	m.moveCursorToFirstFailure()
	m.setStatusMessage(expandSummaryMessage(countFailedLeaves(m.tree), countRetriedLeaves(m.tree)))
	return getClearStatusCmd()
}

// moveCursorToFirstFailure selects the first visible failed leaf, falling
// back to the first preempted/retried leaf, then to any failed non-root node.
func (m *Model) moveCursorToFirstFailure() {
	for i, node := range m.nodes {
		if isFailedStatus(node.Status) && len(node.Children) == 0 {
			m.changeSelectedNode(i)
			return
		}
	}
	for i, node := range m.nodes {
		if wasRetried(node) && len(node.Children) == 0 {
			m.changeSelectedNode(i)
			return
		}
	}
	for i, node := range m.nodes {
		if i > 0 && isFailedStatus(node.Status) {
			m.changeSelectedNode(i)
			return
		}
	}
}

// jumpToFailure moves the cursor to the next/previous failed leaf in the
// full tree, expanding ancestors as needed to make it visible.
func (m *Model) jumpToFailure(next bool) tea.Cmd {
	if m.tree == nil {
		return nil
	}

	all := flattenTree(m.tree)
	var failedIdx []int
	for i, node := range all {
		if isFailedStatus(node.Status) && len(node.Children) == 0 {
			failedIdx = append(failedIdx, i)
		}
	}
	if len(failedIdx) == 0 {
		m.setStatusMessage("No failed tasks found")
		return getClearStatusCmd()
	}

	current := -1
	if m.cursor < len(m.nodes) {
		currentNode := m.nodes[m.cursor]
		for i, node := range all {
			if node == currentNode {
				current = i
				break
			}
		}
	}

	target := all[pickWrapped(failedIdx, current, next)]
	for p := target.Parent; p != nil; p = p.Parent {
		p.Expanded = true
	}
	m.updateSearchFilter()

	for i, node := range m.nodes {
		if node == target {
			m.changeSelectedNode(i)
			return nil
		}
	}

	m.setStatusMessage("Failed task hidden by active filter")
	return getClearStatusCmd()
}

// pickWrapped returns the first index after (or before, when next is false)
// current, wrapping around. indices must be sorted ascending.
func pickWrapped(indices []int, current int, next bool) int {
	if next {
		for _, idx := range indices {
			if idx > current {
				return idx
			}
		}
		return indices[0]
	}
	for i := len(indices) - 1; i >= 0; i-- {
		if indices[i] < current {
			return indices[i]
		}
	}
	return indices[len(indices)-1]
}

// flattenTree returns all nodes of the tree in depth-first (visual) order,
// regardless of expansion state.
func flattenTree(node *TreeNode) []*TreeNode {
	result := []*TreeNode{node}
	for _, child := range node.Children {
		result = append(result, flattenTree(child)...)
	}
	return result
}
