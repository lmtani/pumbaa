package debug

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// taskTimelineEntry represents a task with its timing information for the timeline
type taskTimelineEntry struct {
	Name     string
	Status   string
	Start    time.Time
	End      time.Time
	Duration time.Duration
}

// renderGlobalTimelineModal renders the global timeline modal showing all tasks sorted by duration
func (m Model) renderGlobalTimelineModal() string {
	modalWidth := m.width - 6
	modalHeight := m.height - 4

	title := titleStyle.Render("⏱  Tasks by Duration (longest first): " + m.globalTimelineTitle)

	content := m.globalTimelineViewport.View()

	footer := m.timelineModalFooter()

	modalContent := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		content,
		"",
		footer,
	)

	modal := modalStyle.
		Width(modalWidth).
		Height(modalHeight).
		Render(modalContent)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		modal,
	)
}

// timelineModalFooter generates the footer for the timeline modal
func (m Model) timelineModalFooter() string {
	return mutedStyle.Render("↑↓/PgUp/PgDn scroll • esc close")
}

// buildGlobalTimelineContentForMetadata builds timeline content for a specific workflow/subworkflow
func (m Model) buildGlobalTimelineContentForMetadata(metadata *WorkflowMetadata) string {
	entries := collectTaskTimelineEntriesFromMetadata(metadata)

	if len(entries) == 0 {
		return mutedStyle.Render("No task timing information available")
	}

	// Sort by duration (longest first)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Duration > entries[j].Duration
	})

	// Find the workflow time range for progress bar visualization
	var minStart, maxEnd time.Time
	for i, e := range entries {
		if i == 0 || e.Start.Before(minStart) {
			minStart = e.Start
		}
		if i == 0 || e.End.After(maxEnd) {
			maxEnd = e.End
		}
	}
	totalDuration := maxEnd.Sub(minStart)

	var sb strings.Builder

	// Calculate column widths
	maxNameLen := 30
	for _, e := range entries {
		if len(e.Name) > maxNameLen {
			maxNameLen = len(e.Name)
		}
	}
	if maxNameLen > 50 {
		maxNameLen = 50
	}

	// Summary header
	workflowDur := formatDurationCompact(totalDuration)
	sb.WriteString(mutedStyle.Render(fmt.Sprintf("Workflow duration: %s | Tasks: %d", workflowDur, len(entries))))
	sb.WriteString("\n\n")

	// Build each entry (no header row - the data is self-explanatory)
	for _, e := range entries {
		line := m.formatTimelineEntryStatic(e, maxNameLen, minStart, totalDuration)
		sb.WriteString(line + "\n")
	}

	return sb.String()
}

// formatTimelineEntryStatic formats a single timeline entry (static function)
func (m Model) formatTimelineEntryStatic(e taskTimelineEntry, maxNameLen int, minStart time.Time, totalDuration time.Duration) string {
	// Status icon
	icon := StatusIcon(e.Status)
	style := StatusStyle(e.Status)

	// Remove global title prefix from name to reduce space
	name := strings.TrimPrefix(e.Name, m.globalTimelineTitle+".")
	// Name (truncated if needed, left-aligned)
	if len(name) > maxNameLen {
		name = name[:maxNameLen-3] + "..."
	}
	// Pad name to maxNameLen
	for len(name) < maxNameLen {
		name = name + " "
	}

	// Duration (right-aligned, 8 chars)
	durStr := formatDurationCompact(e.Duration)
	for len(durStr) < 8 {
		durStr = " " + durStr
	}

	// Time range (fixed format HH:MM:SS→HH:MM:SS = 17 chars)
	startStr := e.Start.Format("15:04:05")
	endStr := e.End.Format("15:04:05")
	timeRange := startStr + "→" + endStr

	// Visual timeline bar (30 chars wide)
	barWidth := 30
	bar := buildTimelineBarStatic(e.Start, e.End, minStart, totalDuration, barWidth)

	// Build the line with styled icon at the beginning
	return style.Render(icon) + " " + name + "  " + durStr + "  " + timeRange + "  " + bar
}

// collectTaskTimelineEntriesFromMetadata collects task entries from a specific metadata
func collectTaskTimelineEntriesFromMetadata(metadata *WorkflowMetadata) []taskTimelineEntry {
	var entries []taskTimelineEntry

	if metadata == nil {
		return entries
	}

	for callName, calls := range metadata.Calls {
		for _, call := range calls {
			// Skip subworkflows (they don't have direct timing)
			if call.SubWorkflowMetadata != nil {
				continue
			}

			start := call.Start
			end := call.End

			// Skip if no valid times
			if start.IsZero() || end.IsZero() {
				continue
			}

			duration := end.Sub(start)

			// Build the display name
			name := callName
			if call.ShardIndex >= 0 {
				name = fmt.Sprintf("%s[%d]", callName, call.ShardIndex)
			}
			if call.Attempt > 1 {
				name = fmt.Sprintf("%s (attempt %d)", name, call.Attempt)
			}

			entries = append(entries, taskTimelineEntry{
				Name:     name,
				Status:   string(call.Status),
				Start:    start,
				End:      end,
				Duration: duration,
			})
		}
	}

	return entries
}

// buildTimelineBarStatic creates a visual bar (static function)
func buildTimelineBarStatic(start, end, minStart time.Time, totalDuration time.Duration, width int) string {
	if totalDuration == 0 {
		return strings.Repeat("─", width)
	}

	startOffset := start.Sub(minStart)
	endOffset := end.Sub(minStart)

	startPos := int(float64(startOffset) / float64(totalDuration) * float64(width))
	endPos := int(float64(endOffset) / float64(totalDuration) * float64(width))

	// Ensure at least 1 char width
	if endPos <= startPos {
		endPos = startPos + 1
	}
	if endPos > width {
		endPos = width
	}

	// Build the bar: spaces + filled + spaces
	bar := strings.Repeat("░", startPos) +
		strings.Repeat("█", endPos-startPos) +
		strings.Repeat("░", width-endPos)

	return mutedStyle.Render("[") + valueStyle.Render(bar) + mutedStyle.Render("]")
}

// formatDurationCompact formats duration in a compact human-readable form
func formatDurationCompact(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		secs := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm%ds", mins, secs)
	}
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh%dm", hours, mins)
}
