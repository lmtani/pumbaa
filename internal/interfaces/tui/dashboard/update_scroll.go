package dashboard

// ensureVisible ensures the cursor is within the visible area.
func (m *Model) ensureVisible() {
	visibleRows := m.getVisibleRows()
	if m.cursor < m.scrollY {
		m.scrollY = m.cursor
	} else if m.cursor >= m.scrollY+visibleRows {
		m.scrollY = m.cursor - visibleRows + 1
	}
}

// getVisibleRows calculates the number of rows that can be displayed.
func (m Model) getVisibleRows() int {
	// Account for header, table header, footer, etc.
	return maxInt(1, m.height-12)
}
