package dashboard

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/lmtani/pumbaa/internal/domain/workflow"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
	"github.com/muesli/reflow/truncate"
)

// truncateID truncates a workflow ID to 8 characters for display.
func truncateID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}

// truncateToWidth truncates a string to fit within maxWidth visible characters.
// Uses muesli/reflow/truncate to properly handle ANSI escape codes.
func truncateToWidth(s string, maxWidth int) string {
	if lipgloss.Width(s) <= maxWidth {
		return s
	}
	// Use truncate.StringWithTail which properly handles ANSI escape sequences
	return truncate.StringWithTail(s, uint(maxWidth), "...")
}

// formatDuration formats a duration into a human-readable string (seconds, minutes, or hours).
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh%dm", h, m)
}

// containsStatus checks if a status slice contains a specific status.
func containsStatus(statuses []workflow.Status, status workflow.Status) bool {
	for _, s := range statuses {
		if s == status {
			return true
		}
	}
	return false
}

// minInt returns the minimum of two integers.
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// maxInt returns the maximum of two integers.
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// formatLabelsPlain formats workflow labels as plain text (no styling).
func formatLabelsPlain(labels map[string]string, maxWidth int) string {
	if len(labels) == 0 {
		return ""
	}

	var parts []string
	for k, v := range labels {
		// Skip cromwell internal labels
		if k == "cromwell-workflow-id" {
			continue
		}
		if v != "" {
			parts = append(parts, fmt.Sprintf("%s=%s", k, v))
		} else {
			parts = append(parts, k)
		}
	}

	if len(parts) == 0 {
		return ""
	}

	// Sort for consistent display
	sort.Strings(parts)

	return strings.Join(parts, ", ")
}

// formatLabels formats workflow labels for display, excluding cromwell-workflow-id.
func formatLabels(labels map[string]string, maxWidth int) string {
	result := formatLabelsPlain(labels, maxWidth)
	if result == "" {
		return ""
	}

	// Truncate if exceeds maxWidth
	if maxWidth > 3 && len(result) > maxWidth {
		result = result[:maxWidth-3] + "..."
	}

	return common.MutedStyle.Render(result)
}
