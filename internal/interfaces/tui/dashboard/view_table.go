package dashboard

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/lmtani/pumbaa/internal/domain/workflow"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

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
	header := fmt.Sprintf("%-*s  %-*s  %-*s  %-*s  %-*s  %-*s",
		colWidths[0], "STATUS",
		colWidths[1], "ID",
		colWidths[2], "NAME",
		colWidths[3], "SUBMITTED",
		colWidths[4], "DURATION",
		colWidths[5], "LABELS",
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
		Height(common.ContentPanelHeight(m.height)).
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
	cells := []string{
		common.PadRight(statusText, colWidths[0]),
		common.PadRight(truncateID(wf.ID), colWidths[1]),
		common.PadRight(wf.Name, colWidths[2]),
		common.PadRight(submitted, colWidths[3]),
		common.PadLeft(duration, colWidths[4]),
	}

	// Labels get whatever width remains, so the row never overflows the panel
	base := strings.Join(cells, "  ") + "  "
	labels := formatLabelsPlain(wf.Labels, maxRowWidth-lipgloss.Width(base))

	if selected {
		row := common.TruncateWidth(base+labels, maxRowWidth)
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
		common.StatusStyle(string(wf.Status)).Render(cells[0]),
		common.MutedStyle.Render(cells[1]),
		common.ValueStyle.Render(cells[2]),
		common.MutedStyle.Render(cells[3]),
		common.MutedStyle.Render(cells[4]),
		common.MutedStyle.Render(labels),
	}
	return common.TruncateANSI(strings.Join(parts, "  "), maxRowWidth)
}

// getColumnWidths calculates the width of each table column based on available space.
func (m Model) getColumnWidths() []int {
	// STATUS(12) + ID(9) + SUBMITTED(15) + DURATION(8) = 44 fixed columns,
	// plus 5 separators of 2 cells = 54. NAME and LABELS share the rest.
	maxRowWidth := m.width - 6
	available := maxRowWidth - 54

	// Distribute remaining space: 30% NAME, 70% LABELS. The row renderer gives
	// LABELS whatever is left, so only NAME needs clamping here. The floor of
	// 5 keeps very narrow terminals from producing zero-width columns.
	nameWidth := maxInt(5, minInt(available*30/100, available-5))

	return []int{
		12,                             // STATUS (icon + space + "Succeeded" = 11, +1 for padding)
		9,                              // ID (8 chars + space)
		nameWidth,                      // NAME (flexible)
		15,                             // SUBMITTED (YY-MM-DD HH:MM)
		8,                              // DURATION
		maxInt(5, available-nameWidth), // LABELS (gets more space)
	}
}
