package debug

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderFailures renders workflow-level failures
func (m Model) renderFailures() string {
	var sb strings.Builder

	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF6B6B")).
		Bold(true)

	errorMsgStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF8E8E"))

	sb.WriteString(errorStyle.Render("⚠️  Workflow Failures") + "\n\n")

	for i, failure := range m.metadata.Failures {
		sb.WriteString(renderFailure(failure, 0, i+1, errorMsgStyle))
	}

	return sb.String()
}

// renderTaskFailures renders task-level failures
func (m Model) renderTaskFailures(failures []Failure) string {
	var sb strings.Builder

	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF6B6B")).
		Bold(true)

	errorMsgStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF8E8E"))

	sb.WriteString(errorStyle.Render("⚠️  Task Failures") + "\n\n")

	for i, failure := range failures {
		sb.WriteString(renderFailure(failure, 0, i+1, errorMsgStyle))
	}

	return sb.String()
}

// renderFailure recursively renders a failure and its causes
func renderFailure(f Failure, depth int, index int, style lipgloss.Style) string {
	var sb strings.Builder
	indent := strings.Repeat("  ", depth)

	// Main failure message
	if depth == 0 {
		sb.WriteString(fmt.Sprintf("%s%d. %s\n", indent, index, style.Render(f.Message)))
	} else {
		sb.WriteString(fmt.Sprintf("%s└─ %s\n", indent, style.Render(f.Message)))
	}

	// Render causes
	for _, cause := range f.CausedBy {
		sb.WriteString(renderFailure(cause, depth+1, 0, style))
	}

	return sb.String()
}
