package dashboard

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/lmtani/pumbaa/internal/application/ports"
	"github.com/lmtani/pumbaa/internal/domain/workflow"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

// noteMarkerStyle keeps the memory marker legible without competing with the
// status column for attention.
var noteMarkerStyle = lipgloss.NewStyle().Foreground(common.PrimaryColor)

// renderTable renders the workflow table with all rows and scrolling info.
func (m Model) renderTable() string {
	var b strings.Builder

	// Table header
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(common.TextColor).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(common.BorderColor)

	colWidths := m.getColumnWidths()
	header := fmt.Sprintf("%-*s %-*s  %-*s  %-*s  %-*s  %-*s  %-*s",
		colWidths[0], "",
		colWidths[1], "STATUS",
		colWidths[2], "ID",
		colWidths[3], "NAME",
		colWidths[4], "SUBMITTED",
		colWidths[5], "DURATION",
		colWidths[6], "LABELS",
	)
	header = common.TruncateWidth(header, m.width-6)
	b.WriteString(headerStyle.Render(header) + "\n")

	// Table rows
	visibleRows := m.getVisibleRows()
	startIdx := m.scrollY
	endIdx := minInt(startIdx+visibleRows, len(m.workflows))

	for i := startIdx; i < endIdx; i++ {
		wf := m.workflows[i]
		row := m.renderWorkflowRow(wf, colWidths, i == m.cursor)
		b.WriteString(row + "\n")
	}

	// Scrollbar indicator
	if len(m.workflows) > visibleRows {
		scrollInfo := common.MutedStyle.Render(
			fmt.Sprintf("\n  Showing %d-%d of %d (↑↓ to scroll)", startIdx+1, endIdx, len(m.workflows)),
		)
		b.WriteString(scrollInfo)
	}

	return common.PanelStyle.
		Width(m.width - 2).
		Height(m.contentHeight()).
		Render(b.String())
}

// renderWorkflowRow renders a single workflow row with status, ID, name, and labels.
func (m Model) renderWorkflowRow(wf workflow.Workflow, colWidths []int, selected bool) string {
	maxRowWidth := m.width - 6

	// Submitted time (compact format: YY-MM-DD HH:MM)
	submitted := wf.SubmittedAt.Format("06-01-02 15:04")

	// Duration
	duration := "-"
	if !wf.Start.IsZero() {
		endTime := wf.End
		if endTime.IsZero() {
			endTime = time.Now()
		}
		duration = formatDuration(endTime.Sub(wf.Start))
	}

	// Build each cell as plain text, truncated and padded to its column width.
	// Padding is display-width aware, so multi-byte names never break alignment.
	statusText := common.StatusIcon(string(wf.Status)) + " " + string(wf.Status)
	note, remembered := m.notes[wf.ID]
	cells := []string{
		common.PadRight(memoryMarker(note, remembered), colWidths[0]),
		common.PadRight(statusText, colWidths[1]),
		common.PadRight(truncateID(wf.ID), colWidths[2]),
		common.PadRight(wf.Name, colWidths[3]),
		common.PadRight(submitted, colWidths[4]),
		common.PadLeft(duration, colWidths[5]),
	}

	// The trailing column carries the labels and, after them, whatever of the
	// local note still fits: the note is why the row is worth recognising, so
	// it is shown rather than hidden behind a keystroke.
	base := cells[0] + " " + strings.Join(cells[1:], "  ") + "  "
	trailing := formatLabelsPlain(wf.Labels, maxRowWidth-lipgloss.Width(base))
	if note.Description != "" {
		trailing = appendNote(trailing, note.Description, maxRowWidth-lipgloss.Width(base))
	}

	if selected {
		row := common.TruncateWidth(base+trailing, maxRowWidth)
		// Pad to full width so the highlight covers the entire line
		if d := maxRowWidth - lipgloss.Width(row); d > 0 {
			row += strings.Repeat(" ", d)
		}
		return lipgloss.NewStyle().
			Background(common.HighlightColor).
			Foreground(common.TextColor).
			Render(row)
	}

	// Visual hierarchy: status colored, NAME bright, metadata muted
	parts := []string{
		common.StatusStyle(string(wf.Status)).Render(cells[1]),
		common.MutedStyle.Render(cells[2]),
		common.ValueStyle.Render(cells[3]),
		common.MutedStyle.Render(cells[4]),
		common.MutedStyle.Render(cells[5]),
		common.MutedStyle.Render(trailing),
	}
	row := noteMarkerStyle.Render(cells[0]) + " " + strings.Join(parts, "  ")
	return common.TruncateANSI(row, maxRowWidth)
}

// memoryMarker tells at a glance which rows this machine remembers: a pen for
// the ones carrying a note, a dot for the ones submitted from here without
// one, nothing for the rest.
func memoryMarker(note ports.RunRecord, remembered bool) string {
	switch {
	case note.Description != "":
		return "✎"
	case remembered:
		return "•"
	default:
		return " "
	}
}

// appendNote adds the note after the labels, using whatever room is left. A
// note too long for the row is dropped rather than shown as a stub: the
// marker already says it exists. The pen is repeated here only when labels
// come first, where it is what separates a tag from a sentence.
func appendNote(trailing, note string, available int) string {
	separator := ""
	if trailing != "" {
		separator = "  ✎ "
	}
	room := available - lipgloss.Width(trailing) - lipgloss.Width(separator)
	if room < 8 {
		return trailing
	}
	return trailing + separator + common.Truncate(note, room)
}

// getColumnWidths calculates the width of each table column based on available space.
func (m Model) getColumnWidths() []int {
	// MEMORY(1) + STATUS(12) + ID(9) + SUBMITTED(15) + DURATION(8) = 45 fixed
	// columns, plus the memory separator of 1 cell and 5 separators of 2 cells
	// = 56. NAME and the trailing column share the rest.
	maxRowWidth := m.width - 6
	available := maxRowWidth - 56

	// Distribute remaining space: 30% NAME, 70% LABELS. The row renderer gives
	// LABELS whatever is left, so only NAME needs clamping here. The floor of
	// 5 keeps very narrow terminals from producing zero-width columns.
	nameWidth := maxInt(5, minInt(available*30/100, available-5))

	return []int{
		1,                              // MEMORY (local history marker)
		12,                             // STATUS (icon + space + "Succeeded" = 11, +1 for padding)
		9,                              // ID (8 chars + space)
		nameWidth,                      // NAME (flexible)
		15,                             // SUBMITTED (YY-MM-DD HH:MM)
		8,                              // DURATION
		maxInt(5, available-nameWidth), // LABELS and note (gets more space)
	}
}
