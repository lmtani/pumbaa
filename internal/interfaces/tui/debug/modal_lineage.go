package debug

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/lmtani/pumbaa/internal/domain/workflow"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

// lineageModalState is the lineage modal (O): which chain it lists and the
// selected row.
type lineageModalState struct {
	source workflow.CacheSource
	cursor int // index into lineageRows; row 0 is this run
}

// openLineageModal (O) lists every run of the selected cache hit's chain.
func (m Model) openLineageModal(node *TreeNode) (tea.Model, tea.Cmd) {
	src, ok := cacheSourceOf(node)
	if !ok {
		m.setStatusMessage("Not a cache hit: this task ran here")
		return m, getClearStatusCmd()
	}
	if !m.lineage.enabled() {
		m.setStatusMessage("Following cache hits needs a server connection")
		return m, getClearStatusCmd()
	}
	entry := m.lineage.get(src)
	if entry == nil || entry.pending {
		m.setStatusMessage("Still following the cache chain — try again in a moment")
		return m, tea.Batch(getClearStatusCmd(), m.lineage.request(src))
	}
	m.lineageModal.source = src
	m.lineageModal.cursor = len(m.lineageRows(entry.lineage)) - 1 // the furthest run
	m.activeModal = ModalCacheLineage
	return m, nil
}

func (m Model) renderLineageModal() string {
	entry := m.lineage.get(m.lineageModal.source)
	if entry == nil {
		return ""
	}
	lineage := entry.lineage
	rows := m.lineageRows(lineage)

	modalWidth := minInt(96, m.width-8)
	modalHeight := minInt(len(rows)+10, m.height-4)
	title := titleStyle.Render("Cache lineage · " + callLabel(m.lineageModal.source.CallName, m.lineageModal.source.ShardIndex))

	nameWidth := common.MaxInt(10, modalWidth-66)
	lines := []string{mutedStyle.Render("Results " + lineageSummary(lineage)), ""}
	for i, row := range rows {
		line := fmt.Sprintf("%s %-11s %s  %s  %s  %s",
			row.icon, row.label, shortID(row.id),
			common.PadRight(common.Truncate(row.name, nameWidth), nameWidth),
			formatWhen(row.when), row.detail)
		if row.ran {
			line += "  ← original"
		}
		switch {
		case i == m.lineageModal.cursor:
			lines = append(lines, "▶ "+selectedStyle.Render(line))
		case row.ran:
			lines = append(lines, "  "+valueStyle.Render(line))
		default:
			lines = append(lines, "  "+mutedStyle.Render(line))
		}
	}
	if lineage.Err != nil {
		lines = append(lines, "", errorStyle.Render(common.IconFailed+" "+common.Truncate(lineage.Err.Error(), modalWidth-10)))
	}

	footer := m.modalFooterWithHints("↑↓ navigate", "enter open", "esc close")
	return m.renderCenteredModal(modalWidth, modalHeight, title, strings.Join(lines, "\n"), footer)
}

func (m Model) handleLineageModalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	entry := m.lineage.get(m.lineageModal.source)
	if entry == nil {
		m.activeModal = ModalNone
		return m, nil
	}
	rows := m.lineageRows(entry.lineage)

	switch {
	case key.Matches(msg, m.keys.Escape):
		m.activeModal = ModalNone
	case key.Matches(msg, m.keys.Up):
		if m.lineageModal.cursor > 0 {
			m.lineageModal.cursor--
		}
	case key.Matches(msg, m.keys.Down):
		if m.lineageModal.cursor < len(rows)-1 {
			m.lineageModal.cursor++
		}
	case key.Matches(msg, m.keys.Enter):
		m.activeModal = ModalNone
		if hop := rows[m.lineageModal.cursor].hop; hop != thisRun {
			return m.openHop(m.lineageModal.source, hop)
		}
	}
	return m, nil
}
