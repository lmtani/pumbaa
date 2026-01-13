package debug

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/lmtani/pumbaa/internal/interfaces/tui/debug/tree"
)

func (m *Model) updateSearchFilter() {
	query := strings.TrimSpace(m.searchQuery)
	m.searchQuery = query

	if query == "" {
		m.searchForcedExpanded = nil
		m.nodes = tree.GetVisibleNodes(m.tree)
		m.updateSearchMatches()
		m.ensureCursorInRange()
		m.updateDetailsContent()
		return
	}

	forcedExpanded := make(map[*TreeNode]bool)
	filtered, _ := collectFilteredNodes(m.tree, query, forcedExpanded)
	m.nodes = filtered
	m.searchForcedExpanded = forcedExpanded
	m.updateSearchMatches()
	m.ensureCursorInRange()
	m.updateDetailsContent()
}

func (m Model) handleSearchKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.searchActive = false
		return m, nil
	case tea.KeyEnter:
		m.searchActive = false
		m.updateSearchFilter()
		return m, nil
	case tea.KeyCtrlX:
		m.searchQuery = ""
		m.updateSearchFilter()
		return m, nil
	case tea.KeyBackspace, tea.KeyCtrlH:
		m.searchQuery = removeLastRune(m.searchQuery)
		m.updateSearchFilter()
		return m, nil
	case tea.KeyRunes:
		m.searchQuery += string(msg.Runes)
		m.updateSearchFilter()
		return m, nil
	}

	return m, nil
}

func (m *Model) updateSearchMatches() {
	m.searchMatches = nil
	if m.searchQuery == "" {
		m.searchMatchCursor = 0
		return
	}

	for i, node := range m.nodes {
		if nodeMatchesQuery(node, m.searchQuery) {
			m.searchMatches = append(m.searchMatches, i)
		}
	}

	if len(m.searchMatches) == 0 {
		m.searchMatchCursor = 0
		return
	}

	for i, idx := range m.searchMatches {
		if idx == m.cursor {
			m.searchMatchCursor = i
			return
		}
	}
	m.searchMatchCursor = 0
	m.cursor = m.searchMatches[0]
}

func (m *Model) ensureCursorInRange() {
	if len(m.nodes) == 0 {
		m.cursor = 0
		return
	}
	if m.cursor >= len(m.nodes) {
		m.cursor = len(m.nodes) - 1
	}
}

func (m *Model) jumpToSearchMatch(next bool) {
	if len(m.searchMatches) == 0 {
		m.setStatusMessage("No matches")
		return
	}

	if next {
		m.searchMatchCursor++
		if m.searchMatchCursor >= len(m.searchMatches) {
			m.searchMatchCursor = 0
		}
	} else {
		m.searchMatchCursor--
		if m.searchMatchCursor < 0 {
			m.searchMatchCursor = len(m.searchMatches) - 1
		}
	}

	m.cursor = m.searchMatches[m.searchMatchCursor]
	m.focus = FocusTree
	m.updateDetailsContent()
}

func nodeMatchesQuery(node *TreeNode, query string) bool {
	q := strings.ToLower(query)
	if strings.Contains(strings.ToLower(node.Name), q) {
		return true
	}
	return strings.Contains(strings.ToLower(node.Status), q)
}

func collectFilteredNodes(node *TreeNode, query string, forcedExpanded map[*TreeNode]bool) ([]*TreeNode, bool) {
	matchesSelf := nodeMatchesQuery(node, query)
	type childResult struct {
		nodes   []*TreeNode
		matched bool
	}

	childResults := make([]childResult, 0, len(node.Children))
	hasChildMatch := false
	for _, child := range node.Children {
		childNodes, childMatched := collectFilteredNodes(child, query, forcedExpanded)
		childResults = append(childResults, childResult{nodes: childNodes, matched: childMatched})
		if childMatched {
			hasChildMatch = true
		}
	}

	if !matchesSelf && !hasChildMatch {
		return nil, false
	}

	if hasChildMatch {
		forcedExpanded[node] = true
	}

	result := []*TreeNode{node}
	for _, res := range childResults {
		if res.matched {
			result = append(result, res.nodes...)
		}
	}

	return result, true
}

func removeLastRune(value string) string {
	runes := []rune(value)
	if len(runes) == 0 {
		return value
	}
	return string(runes[:len(runes)-1])
}
