package debug

import (
	"fmt"
	"strings"

	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

func (m Model) renderTree() string {
	var sb strings.Builder
	panelHeight := common.ContentPanelHeight(m.height)
	if len(m.nodes) == 0 && m.searchQuery != "" {
		sb.WriteString(mutedStyle.Render(fmt.Sprintf("No matches for %q", m.searchQuery)))
		style := treePanelStyle.Width(m.treeWidth).Height(panelHeight)
		if m.focus == FocusTree {
			style = style.BorderForeground(common.FocusBorder)
		}
		return style.Render(sb.String())
	}

	startIdx := 0
	maxVisible := panelHeight
	if m.cursor >= maxVisible {
		startIdx = m.cursor - maxVisible + 1
	}

	endIdx := startIdx + maxVisible
	if endIdx > len(m.nodes) {
		endIdx = len(m.nodes)
	}

	lines := make([]string, 0, endIdx-startIdx)
	for i := startIdx; i < endIdx; i++ {
		lines = append(lines, m.renderTreeNode(m.nodes[i], i))
	}
	// Join without a trailing newline: an extra final line would grow the
	// panel past its height budget when the tree fills it completely.
	sb.WriteString(strings.Join(lines, "\n"))

	style := treePanelStyle.Width(m.treeWidth).Height(panelHeight)
	if m.focus == FocusTree {
		style = style.BorderForeground(common.FocusBorder)
	}

	return style.Render(sb.String())
}

func (m Model) renderTreeNode(node *TreeNode, index int) string {
	prefix := strings.Repeat("  ", node.Depth)

	// Node indicator
	indicator := "├─"
	// Check if this is the last child of its parent
	isLast := false
	if node.Parent != nil {
		children := node.Parent.Children
		if len(children) > 0 && children[len(children)-1] == node {
			isLast = true
		}
	}
	if isLast {
		indicator = "└─"
	}

	// Expand/collapse indicator
	expanded := node.Expanded
	if m.searchQuery != "" && m.searchForcedExpanded != nil {
		expanded = m.searchForcedExpanded[node]
	}
	expandIndicator := " "
	if len(node.Children) > 0 || (node.Type == NodeTypeSubWorkflow && node.SubWorkflowID != "") {
		if expanded {
			expandIndicator = common.IconExpanded
		} else {
			expandIndicator = common.IconCollapsed
		}
	}

	// Status icon
	statusIcon := statusStyle(node.Status)

	// Node type icon
	typeIcon := ""
	switch node.Type {
	case NodeTypeWorkflow:
		typeIcon = common.IconWorkflow
	case NodeTypeCall:
		typeIcon = common.IconTask
	case NodeTypeSubWorkflow:
		typeIcon = common.IconSubworkflow
	case NodeTypeShard:
		typeIcon = common.IconShard
	}

	// Preemption count indicator (only for nodes with children or successful retries)
	preemptBadge := ""
	if len(node.Children) > 0 || (node.Status == "Done" && node.CallData != nil) {
		preemptCount := countPreemptions(node)
		if preemptCount > 0 {
			preemptBadge = mutedStyle.Render(fmt.Sprintf(" ⟳%d", preemptCount))
		}
	}

	// Failed-descendant count on collapsed nodes, so failures buried in
	// collapsed scatters/subworkflows stay visible at a glance
	failedBadge := ""
	if len(node.Children) > 0 && !expanded {
		if failedCount := countFailedLeaves(node); failedCount > 0 {
			failedBadge = failedBadgeStyle.Render(fmt.Sprintf(" %s%d", common.IconFailed, failedCount))
		}
	}

	// Name with truncation (account for badges)
	maxNameLen := m.treeWidth - node.Depth*2 - 12
	if preemptBadge != "" {
		maxNameLen -= 4 // Reserve space for badge
	}
	if failedBadge != "" {
		maxNameLen -= 4 // Reserve space for badge
	}
	name := truncate(node.Name, maxNameLen)

	// Build the node string
	nodeStr := fmt.Sprintf("%s%s %s %s %s %s%s%s", prefix, indicator, expandIndicator, statusIcon, typeIcon, name, preemptBadge, failedBadge)

	// Style based on selection
	if index == m.cursor {
		return selectedStyle.Render(nodeStr)
	}
	return nodeStr
}
