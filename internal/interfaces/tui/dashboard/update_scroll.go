package dashboard

import "github.com/lmtani/pumbaa/internal/interfaces/tui/common"

// ensureVisible ensures the cursor is within the visible area.
func (m *Model) ensureVisible() {
	visibleRows := m.getVisibleRows()
	if m.cursor < m.scrollY {
		m.scrollY = m.cursor
	} else if m.cursor >= m.scrollY+visibleRows {
		m.scrollY = m.cursor - visibleRows + 1
	}
}

// getVisibleRows calculates the number of workflow rows that fit inside the
// table panel: panel height minus the table header (2 lines including its
// bottom border) and the scroll indicator (2 lines).
func (m Model) getVisibleRows() int {
	return maxInt(1, common.ContentPanelHeight(m.height)-4)
}
