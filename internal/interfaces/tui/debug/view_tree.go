package debug

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

func (m Model) renderTree() string {
	var sb strings.Builder

	startIdx := 0
	maxVisible := m.height - 10 // Leave room for header and footer
	if maxVisible < 5 {
		maxVisible = 5
	}
	if m.cursor >= maxVisible {
		startIdx = m.cursor - maxVisible + 1
	}

	endIdx := startIdx + maxVisible
	if endIdx > len(m.nodes) {
		endIdx = len(m.nodes)
	}

	for i := startIdx; i < endIdx; i++ {
		node := m.nodes[i]
		sb.WriteString(m.renderTreeNode(node, i))
		sb.WriteString("\n")
	}

	style := treePanelStyle.Width(m.treeWidth).Height(m.height - 8)
	if m.focus == FocusTree {
		style = style.BorderForeground(lipgloss.Color("#7D56F4"))
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
	expandIndicator := " "
	if len(node.Children) > 0 || (node.Type == NodeTypeSubWorkflow && node.SubWorkflowID != "") {
		if node.Expanded {
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

	// Name with truncation (account for preempt badge)
	maxNameLen := m.treeWidth - node.Depth*2 - 12
	if preemptBadge != "" {
		maxNameLen -= 4 // Reserve space for badge
	}
	name := truncate(node.Name, maxNameLen)

	// Build the node string
	nodeStr := fmt.Sprintf("%s%s %s %s %s %s%s", prefix, indicator, expandIndicator, statusIcon, typeIcon, name, preemptBadge)

	// Style based on selection
	if index == m.cursor {
		return selectedStyle.Render(nodeStr)
	}
	return nodeStr
}
