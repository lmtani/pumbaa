package debug

// Helper methods for tree manipulation

func (m *Model) updateDetailsContent() {
	if m.cursor >= len(m.nodes) {
		return
	}

	node := m.nodes[m.cursor]
	content := m.renderDetailsContent(node)
	m.detailViewport.SetContent(content)
	m.detailViewport.GotoTop()
}

func (m *Model) expandAll(node *TreeNode) {
	node.Expanded = true
	for _, child := range node.Children {
		m.expandAll(child)
	}
}

func (m *Model) collapseAll(node *TreeNode) {
	if node.Depth > 0 {
		node.Expanded = false
	}
	for _, child := range node.Children {
		m.collapseAll(child)
	}
}

// findNodeByID finds a node by its ID in the tree
func (m Model) findNodeByID(node *TreeNode, id string) *TreeNode {
	if node.ID == id {
		return node
	}
	for _, child := range node.Children {
		if found := m.findNodeByID(child, id); found != nil {
			return found
		}
	}
	return nil
}
