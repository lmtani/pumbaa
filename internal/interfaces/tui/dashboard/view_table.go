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
		Height(m.height - 8).
		Render(b.String())
}

// renderWorkflowRow renders a single workflow row with status, ID, name, and labels.
func (m Model) renderWorkflowRow(wf workflow.Workflow, colWidths []int, selected bool) string {
	maxRowWidth := m.width - 6

	// Status icon and text
	statusIcon := common.StatusIcon(string(wf.Status))
	statusText := statusIcon + " " + string(wf.Status)
	// Pad status to fixed width
	statusPadding := colWidths[0] - lipgloss.Width(statusText)
	if statusPadding > 0 {
		statusText = statusText + strings.Repeat(" ", statusPadding)
	}

	// ID (truncated)
	id := truncateID(wf.ID)
	if len(id) > colWidths[1] {
		id = id[:colWidths[1]-3] + "..."
	}

	// Name
	name := wf.Name
	if len(name) > colWidths[2] {
		name = name[:colWidths[2]-3] + "..."
	}

	// Submitted time (compact format: YY-MM-DD HH:MM)
	submitted := wf.SubmittedAt.Format("06-01-02 15:04")

	// Duration
	duration := "-"
	if !wf.Start.IsZero() {
		endTime := wf.End
		if endTime.IsZero() {
			endTime = time.Now()
		}
		dur := endTime.Sub(wf.Start)
		duration = formatDuration(dur)
	}

	// Labels (plain text, no styling yet)
	labelsText := formatLabelsPlain(wf.Labels, colWidths[5])

	// Build row without ANSI codes for proper truncation
	row := fmt.Sprintf("%s  %-*s  %-*s  %-*s  %*s  %s",
		statusText,
		colWidths[1], id,
		colWidths[2], name,
		colWidths[3], submitted,
		colWidths[4], duration,
		labelsText,
	)

	// Truncate if needed (now safe since no ANSI codes)
	if len(row) > maxRowWidth {
		if maxRowWidth > 3 {
			row = row[:maxRowWidth-3] + "..."
		} else {
			row = row[:maxRowWidth]
		}
	}

	// Pad to full width for consistent highlighting
	if len(row) < maxRowWidth {
		row = row + strings.Repeat(" ", maxRowWidth-len(row))
	}

	if selected {
		return lipgloss.NewStyle().
			Background(common.HighlightColor).
			Foreground(common.TextColor).
			Render(row)
	}

	// Apply status color only to the status portion for non-selected rows
	statusStyle := common.StatusStyle(string(wf.Status))
	coloredStatus := statusStyle.Render(statusText)
	// Replace plain status with colored version
	rest := row[len(statusText):]
	return coloredStatus + common.MutedStyle.Render(rest)
}

// getColumnWidths calculates the width of each table column based on available space.
func (m Model) getColumnWidths() []int {
	// STATUS, ID, NAME, SUBMITTED, DURATION, LABELS
	// Fixed widths: STATUS(12) + ID(9) + SUBMITTED(15) + DURATION(8) + separators(12) = 56
	fixedWidth := 56
	available := m.width - fixedWidth

	// Distribute remaining space: 30% NAME, 70% LABELS
	nameWidth := maxInt(10, available*30/100)
	labelsWidth := maxInt(10, available-nameWidth)

	return []int{
		12,          // STATUS (icon + space + "Succeeded" = 11, +1 for padding)
		9,           // ID (8 chars + space)
		nameWidth,   // NAME (flexible)
		15,          // SUBMITTED (YY-MM-DD HH:MM)
		8,           // DURATION
		labelsWidth, // LABELS (gets more space)
	}
}
