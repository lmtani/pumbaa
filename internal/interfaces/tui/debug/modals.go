package debug

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// modalFooter generates the footer for modals, including copy feedback if present
func (m Model) modalFooter() string {
	baseFooter := "↑↓/PgUp/PgDn scroll • y copy • esc close"
	if m.statusMessage != "" {
		return mutedStyle.Render(baseFooter) + "  " + temporaryStatusStyle.Render(m.statusMessage)
	}
	return mutedStyle.Render(baseFooter)
}

// renderLogModal renders the log modal.
func (m Model) renderLogModal() string {
	modalWidth := m.width - 6
	modalHeight := m.height - 4

	// Modal title
	title := titleStyle.Render("📄 " + m.logModalTitle)

	// Modal content
	var content string
	if m.logModalError != "" {
		content = errorStyle.Render("Error: " + m.logModalError)
	} else if m.logModalLoading {
		content = mutedStyle.Render("Loading...")
	} else {
		content = m.logModalViewport.View()
	}

	// Footer with instructions
	footer := m.modalFooter()

	// Build modal box
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

	// Center the modal
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		modal,
	)
}

// renderInputsModal renders the inputs modal.
func (m Model) renderInputsModal() string {
	modalWidth := m.width - 6
	modalHeight := m.height - 4

	title := titleStyle.Render("📥 Workflow Inputs: " + m.metadata.Name)

	content := m.inputsModalViewport.View()

	footer := m.modalFooter()

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

// renderOutputsModal renders the outputs modal.
func (m Model) renderOutputsModal() string {
	modalWidth := m.width - 6
	modalHeight := m.height - 4

	title := titleStyle.Render("📤 Workflow Outputs: " + m.metadata.Name)

	content := m.outputsModalViewport.View()

	footer := m.modalFooter()

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

// renderOptionsModal renders the options modal.
func (m Model) renderOptionsModal() string {
	modalWidth := m.width - 6
	modalHeight := m.height - 4

	title := titleStyle.Render("⚙️  Workflow Options: " + m.metadata.Name)

	content := m.optionsModalViewport.View()

	footer := m.modalFooter()

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

// renderCallInputsModal renders the call inputs modal.
func (m Model) renderCallInputsModal() string {
	modalWidth := m.width - 6
	modalHeight := m.height - 4

	// Get current node for title
	nodeName := "Unknown"
	if m.cursor < len(m.nodes) {
		nodeName = m.nodes[m.cursor].Name
	}

	title := titleStyle.Render("📥 Call Inputs: " + nodeName)

	content := m.callInputsViewport.View()

	footer := m.modalFooter()

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

// renderCallOutputsModal renders the call outputs modal.
func (m Model) renderCallOutputsModal() string {
	modalWidth := m.width - 6
	modalHeight := m.height - 4

	// Get current node for title
	nodeName := "Unknown"
	if m.cursor < len(m.nodes) {
		nodeName = m.nodes[m.cursor].Name
	}

	title := titleStyle.Render("📤 Call Outputs: " + nodeName)

	content := m.callOutputsViewport.View()

	footer := m.modalFooter()

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

// renderCallCommandModal renders the call command modal.
func (m Model) renderCallCommandModal() string {
	modalWidth := m.width - 6
	modalHeight := m.height - 4

	// Get current node for title
	nodeName := "Unknown"
	if m.cursor < len(m.nodes) {
		nodeName = m.nodes[m.cursor].Name
	}

	title := titleStyle.Render("📜 Call Command: " + nodeName)

	content := m.callCommandViewport.View()

	footer := m.modalFooter()

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

// formatInputsForModal formats inputs for display in the modal.
func (m Model) formatInputsForModal(node *TreeNode) string {
	if node.CallData == nil || len(node.CallData.Inputs) == 0 {
		return mutedStyle.Render("No inputs available")
	}

	var sb strings.Builder

	// Sort keys for consistent display
	keys := make([]string, 0, len(node.CallData.Inputs))
	for k := range node.CallData.Inputs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := node.CallData.Inputs[k]
		sb.WriteString(modalLabelStyle.Render(k) + "\n")
		sb.WriteString(formatValueForModal(v, m.width-16) + "\n\n")
	}

	return sb.String()
}

// formatWorkflowInputsForModal formats workflow inputs for display in the modal.
func (m Model) formatWorkflowInputsForModal() string {
	if len(m.metadata.Inputs) == 0 {
		return mutedStyle.Render("No inputs available")
	}

	var sb strings.Builder

	// Sort keys for consistent display
	keys := make([]string, 0, len(m.metadata.Inputs))
	for k := range m.metadata.Inputs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := m.metadata.Inputs[k]
		// Skip null values
		if v == nil {
			continue
		}
		sb.WriteString(modalLabelStyle.Render(k) + "\n")
		sb.WriteString(formatValueForModal(v, m.width-16) + "\n\n")
	}

	return sb.String()
}

// formatWorkflowOutputsForModal formats workflow outputs for display in the modal.
func (m Model) formatWorkflowOutputsForModal() string {
	if len(m.metadata.Outputs) == 0 {
		return mutedStyle.Render("No outputs available")
	}

	var sb strings.Builder

	// Sort keys for consistent display
	keys := make([]string, 0, len(m.metadata.Outputs))
	for k := range m.metadata.Outputs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := m.metadata.Outputs[k]
		// Skip null values
		if v == nil {
			continue
		}
		sb.WriteString(modalLabelStyle.Render(k) + "\n")
		sb.WriteString(formatValueForModal(v, m.width-16) + "\n\n")
	}

	return sb.String()
}

// formatOptionsForModal formats workflow options for display in the modal.
func (m Model) formatOptionsForModal() string {
	if m.metadata.SubmittedOptions == "" {
		return mutedStyle.Render("No options available")
	}

	// Parse the JSON options
	var options map[string]interface{}
	if err := json.Unmarshal([]byte(m.metadata.SubmittedOptions), &options); err != nil {
		// If it's not valid JSON, just return the raw string formatted
		return modalValueStyle.Render(m.metadata.SubmittedOptions)
	}

	var sb strings.Builder

	// Sort keys for consistent display
	keys := make([]string, 0, len(options))
	for k := range options {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := options[k]
		// Skip null values
		if v == nil {
			continue
		}
		sb.WriteString(modalLabelStyle.Render(k) + "\n")
		sb.WriteString(formatValueForModal(v, m.width-16) + "\n\n")
	}

	return sb.String()
}

// formatCallInputsForModal formats call inputs for display in the modal.
func (m Model) formatCallInputsForModal(node *TreeNode) string {
	if node.CallData == nil || len(node.CallData.Inputs) == 0 {
		return mutedStyle.Render("No inputs available")
	}

	var sb strings.Builder

	// Sort keys for consistent display
	keys := make([]string, 0, len(node.CallData.Inputs))
	for k := range node.CallData.Inputs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := node.CallData.Inputs[k]
		// Skip null values
		if v == nil {
			continue
		}
		sb.WriteString(modalLabelStyle.Render(k) + "\n")
		sb.WriteString(formatValueForModal(v, m.width-16) + "\n\n")
	}

	return sb.String()
}

// formatCallOutputsForModal formats call outputs for display in the modal.
func (m Model) formatCallOutputsForModal(node *TreeNode) string {
	if node.CallData == nil || len(node.CallData.Outputs) == 0 {
		return mutedStyle.Render("No outputs available")
	}

	var sb strings.Builder

	// Sort keys for consistent display
	keys := make([]string, 0, len(node.CallData.Outputs))
	for k := range node.CallData.Outputs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := node.CallData.Outputs[k]
		// Skip null values
		if v == nil {
			continue
		}
		sb.WriteString(modalLabelStyle.Render(k) + "\n")
		sb.WriteString(formatValueForModal(v, m.width-16) + "\n\n")
	}

	return sb.String()
}

// formatCallCommandForModal formats call command for display in the modal.
func (m Model) formatCallCommandForModal(node *TreeNode) string {
	if node.CallData == nil || node.CallData.CommandLine == "" {
		return mutedStyle.Render("No command available")
	}

	// Wrap text to fit the modal width
	wrapped := wrapText(node.CallData.CommandLine, m.width-20)
	return commandStyle.Render(wrapped)
}

// formatValueForModal formats a value for display in modals with appropriate colors.
func formatValueForModal(v interface{}, maxWidth int) string {
	return formatValueWithStyles(v, maxWidth, modalValueStyle, modalPathStyle, mutedStyle)
}

// formatValueWithStyles formats a value using the provided styles.
func formatValueWithStyles(v interface{}, maxWidth int, valStyle, pthStyle, mutStyle lipgloss.Style) string {
	if maxWidth < 20 {
		maxWidth = 80
	}

	switch val := v.(type) {
	case nil:
		return mutStyle.Render("  null")
	case bool:
		return valStyle.Render(fmt.Sprintf("  %v", val))
	case float64:
		// Check if it's an integer
		if val == float64(int64(val)) {
			return valStyle.Render(fmt.Sprintf("  %d", int64(val)))
		}
		return valStyle.Render(fmt.Sprintf("  %g", val))
	case string:
		wrappedVal := val
		if len(val) > maxWidth-4 {
			wrappedVal = wrapText(val, maxWidth-4)
		}
		// Handle GCS paths with special styling
		if strings.HasPrefix(val, "gs://") {
			return pthStyle.Render("  " + wrappedVal)
		}
		// Handle local paths
		if strings.HasPrefix(val, "/") {
			return pthStyle.Render("  " + wrappedVal)
		}
		return valStyle.Render("  " + wrappedVal)
	case []interface{}:
		if len(val) == 0 {
			return mutStyle.Render("  []")
		}
		var sb strings.Builder
		for i, item := range val {
			prefix := "  - "
			itemStr := formatValueWithStyles(item, maxWidth-4, valStyle, pthStyle, mutStyle)
			// Remove leading spaces from nested formatValue
			itemStr = strings.TrimPrefix(itemStr, "  ")
			sb.WriteString(prefix + itemStr)
			if i < len(val)-1 {
				sb.WriteString("\n")
			}
		}
		return sb.String()
	case map[string]interface{}:
		// Pretty print maps with indentation
		jsonBytes, err := json.MarshalIndent(val, "  ", "  ")
		if err != nil {
			return mutStyle.Render("  [complex object]")
		}
		return valStyle.Render("  " + string(jsonBytes))
	default:
		// Fallback to JSON for unknown types
		jsonBytes, err := json.MarshalIndent(val, "  ", "  ")
		if err != nil {
			return valStyle.Render(fmt.Sprintf("  %v", val))
		}
		return valStyle.Render("  " + string(jsonBytes))
	}
}

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

// buildGlobalTimelineContent builds the content for the global timeline modal (uses root metadata)
func (m Model) buildGlobalTimelineContent() string {
	return m.buildGlobalTimelineContentForMetadata(m.metadata)
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
		line := formatTimelineEntryStatic(e, maxNameLen, minStart, totalDuration)
		sb.WriteString(line + "\n")
	}

	return sb.String()
}

// collectTaskTimelineEntries collects all task entries with timing from the workflow
func (m Model) collectTaskTimelineEntries() []taskTimelineEntry {
	return collectTaskTimelineEntriesFromMetadata(m.metadata)
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
				Status:   call.ExecutionStatus,
				Start:    start,
				End:      end,
				Duration: duration,
			})
		}
	}

	return entries
}

// formatTimelineEntry formats a single timeline entry with a visual timeline bar
func (m Model) formatTimelineEntry(e taskTimelineEntry, maxNameLen int, minStart time.Time, totalDuration time.Duration) string {
	return formatTimelineEntryStatic(e, maxNameLen, minStart, totalDuration)
}

// formatTimelineEntryStatic formats a single timeline entry (static function)
func formatTimelineEntryStatic(e taskTimelineEntry, maxNameLen int, minStart time.Time, totalDuration time.Duration) string {
	// Status icon
	icon := StatusIcon(e.Status)
	style := StatusStyle(e.Status)

	// Name (truncated if needed, left-aligned)
	name := e.Name
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

// buildTimelineBar creates a visual bar showing when the task ran relative to workflow
func (m Model) buildTimelineBar(start, end, minStart time.Time, totalDuration time.Duration, width int) string {
	return buildTimelineBarStatic(start, end, minStart, totalDuration, width)
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
