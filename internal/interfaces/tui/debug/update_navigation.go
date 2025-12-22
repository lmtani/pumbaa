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

// saveNodeState saves the view state for the current node
func (m *Model) saveNodeState(nodeID string) {
	if m.nodeStates == nil {
		m.nodeStates = make(map[string]NodeViewState)
	}
	m.nodeStates[nodeID] = NodeViewState{
		ViewMode:   m.viewMode,
		LogCursor:  m.logCursor,
		PanelFocus: m.focus,
	}
}

// restoreNodeState restores the view state for the current node
func (m *Model) restoreNodeState(nodeID string) {
	if state, ok := m.nodeStates[nodeID]; ok {
		m.viewMode = state.ViewMode
		m.logCursor = state.LogCursor
		// Only restore focus if it makes sense (e.g. don't trap focus in details if we just navigated)
		// but for now let's keep it simple and see if it feels right.
		// Actually, standard TUI behavior usually keeps focus on tree during navigation.
		// Let's NOT restore focus for now, unless we are in a mode that demands it.
		// m.focus = state.PanelFocus

		// Access cached efficiency report immediately if in Monitor mode
		if m.viewMode == ViewModeMonitor {
			node := m.nodes[m.cursor]
			if node.CallData != nil && node.CallData.EfficiencyReport != nil {
				m.resourceReport = node.CallData.EfficiencyReport
				m.resourceError = ""
			} else {
				// If no cache, fall back to details to avoid loading state flicker or empty screen
				m.viewMode = ViewModeDetails
				m.resourceReport = nil
			}
		}
	} else {
		// Default state for new nodes
		m.viewMode = ViewModeDetails
		m.logCursor = 0
		m.resourceReport = nil
		m.resourceError = ""
	}
}

// changeSelectedNode handles selection change with state persistence
func (m *Model) changeSelectedNode(newCursor int) {
	if m.cursor < len(m.nodes) {
		currentNode := m.nodes[m.cursor]
		m.saveNodeState(currentNode.ID)
	}

	m.cursor = newCursor

	if m.cursor < len(m.nodes) {
		newNode := m.nodes[m.cursor]
		m.restoreNodeState(newNode.ID)
	}

	m.updateDetailsContent()
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
