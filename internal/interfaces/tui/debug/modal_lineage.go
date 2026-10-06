package debug

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/lmtani/pumbaa/internal/domain/workflow"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

// lineageRow is one line of the cache lineage, shared by the details panel
// and the lineage modal (O) so both read the chain the same way.
type lineageRow struct {
	icon   string
	label  string // "this run", "1 run back", ...
	id     string
	name   string // workflow name
	when   time.Time
	detail string // "cache hit", "ran 6m23s"
	ran    bool   // the producing run
}

// lineageRows lays the chain out from this run back to the furthest run
// reached.
func (m Model) lineageRows(lineage workflow.CacheLineage) []lineageRow {
	rows := []lineageRow{{
		icon:   common.IconWorkflow,
		label:  "this run",
		id:     m.metadata.ID,
		name:   m.metadata.Name,
		when:   submittedAt(m.metadata),
		detail: "cache hit",
	}}
	for i, hop := range lineage.Hops {
		row := lineageRow{
			icon:   common.IconCached,
			label:  runsBack(i + 1),
			id:     hop.Workflow.ID,
			name:   hop.Workflow.Name,
			when:   submittedAt(hop.Workflow),
			detail: "cache hit",
		}
		if hop.Ran() {
			row.icon = statusStyle(string(hop.Call.Status))
			row.detail = "ran"
			if !hop.Call.Start.IsZero() && !hop.Call.End.IsZero() {
				row.detail += " " + formatDuration(hop.Call.End.Sub(hop.Call.Start))
			}
			row.ran = true
		}
		rows = append(rows, row)
	}
	return rows
}

func submittedAt(wf *workflow.Workflow) time.Time {
	if !wf.SubmittedAt.IsZero() {
		return wf.SubmittedAt
	}
	return wf.Start
}

func formatWhen(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return t.Format("02 Jan 15:04")
}

// lineageSummary is the one-line verdict: how far back the results come from.
func lineageSummary(lineage workflow.CacheLineage) string {
	if _, ok := lineage.Original(); ok {
		n := len(lineage.Hops)
		if n == 1 {
			return "produced by the previous run"
		}
		return fmt.Sprintf("produced %s (%d cached in between)", runsBack(n), n-1)
	}
	return fmt.Sprintf("chain broken after %s", runsBack(len(lineage.Hops)))
}

// renderCacheSection renders the Cache block of a call's details.
func (m Model) renderCacheSection(node *TreeNode) string {
	cd := node.CallData
	var sb strings.Builder
	sb.WriteString("\n" + titleStyle.Render("Cache") + "\n")

	src, ok := cacheSourceOf(node)
	if !ok {
		status := "Miss"
		if cd.CacheHit {
			status = "Hit"
		}
		sb.WriteString(labelStyle.Render("Status: ") + valueStyle.Render(status) + "\n")
		if cd.CacheResult != "" {
			sb.WriteString(labelStyle.Render("Result: ") + valueStyle.Render(cd.CacheResult) + "\n")
		}
		return sb.String()
	}

	entry := m.lineageFor(src)
	switch {
	case m.lineageUC == nil:
		sb.WriteString(labelStyle.Render("Status: ") + valueStyle.Render("Hit") + "\n")
		sb.WriteString(labelStyle.Render("Copied from: ") + valueStyle.Render(callLabel(src.CallName, src.ShardIndex)) + "\n")
		sb.WriteString("  " + mutedStyle.Render("in workflow "+src.WorkflowID) + "\n")

	case entry == nil || entry.pending:
		sb.WriteString(labelStyle.Render("Status: ") + valueStyle.Render("Hit") + "\n")
		sb.WriteString(mutedStyle.Render("Following the chain back from "+shortID(src.WorkflowID)+"…") + "\n")

	default:
		lineage := entry.lineage
		sb.WriteString(labelStyle.Render("Status: ") + valueStyle.Render("Hit") + mutedStyle.Render(" · "+lineageSummary(lineage)) + "\n")
		for _, row := range m.lineageRows(lineage) {
			line := fmt.Sprintf("%s %-11s %s  %s  %s", row.icon, row.label, shortID(row.id), formatWhen(row.when), row.detail)
			if row.ran {
				sb.WriteString(valueStyle.Render(line) + "\n")
			} else {
				sb.WriteString(mutedStyle.Render(line) + "\n")
			}
		}
		if lineage.Err != nil {
			sb.WriteString(errorStyle.Render(common.IconFailed+" "+common.Truncate(lineage.Err.Error(), m.detailsWidth-10)) + "\n")
		}
		target := "producing run"
		if _, ok := lineage.Original(); !ok {
			target = "furthest run"
		}
		sb.WriteString(common.KeyStyle.Render("o") + mutedStyle.Render(" open "+target+"  ") +
			common.KeyStyle.Render("O") + mutedStyle.Render(" choose a run") + "\n")
	}
	return sb.String()
}

// openLineageModal (O) lists every run of the selected cache hit's chain.
func (m Model) openLineageModal(node *TreeNode) (tea.Model, tea.Cmd) {
	src, ok := cacheSourceOf(node)
	if !ok {
		m.setStatusMessage("Not a cache hit: this task ran here")
		return m, getClearStatusCmd()
	}
	entry := m.lineageFor(src)
	if entry == nil || entry.pending {
		m.setStatusMessage("Still following the cache chain — try again in a moment")
		return m, tea.Batch(getClearStatusCmd(), m.resolveLineage(src))
	}
	m.lineageModalSource = src
	m.lineageModalCursor = len(entry.lineage.Hops) // the furthest run
	m.activeModal = ModalCacheLineage
	return m, nil
}

func (m Model) renderLineageModal() string {
	entry := m.lineageFor(m.lineageModalSource)
	if entry == nil {
		return ""
	}
	lineage := entry.lineage
	rows := m.lineageRows(lineage)

	modalWidth := minInt(96, m.width-8)
	modalHeight := minInt(len(rows)+10, m.height-4)
	title := titleStyle.Render("Cache lineage · " + callLabel(m.lineageModalSource.CallName, m.lineageModalSource.ShardIndex))

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
		case i == m.lineageModalCursor:
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
	entry := m.lineageFor(m.lineageModalSource)
	if entry == nil {
		m.activeModal = ModalNone
		return m, nil
	}
	last := len(entry.lineage.Hops)

	switch {
	case key.Matches(msg, m.keys.Escape):
		m.activeModal = ModalNone
	case key.Matches(msg, m.keys.Up):
		if m.lineageModalCursor > 0 {
			m.lineageModalCursor--
		}
	case key.Matches(msg, m.keys.Down):
		if m.lineageModalCursor < last {
			m.lineageModalCursor++
		}
	case key.Matches(msg, m.keys.Enter):
		m.activeModal = ModalNone
		if m.lineageModalCursor == 0 {
			return m, nil // this run: already here
		}
		return m.openHop(m.lineageModalSource, m.lineageModalCursor-1)
	}
	return m, nil
}
