package debug

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/ansi"
)

// renderLogModal renders the log modal.
func (m Model) renderLogModal() string {
	modalWidth := m.width - 6
	modalHeight := m.height - 4

	// Modal title with scroll indicator
	titleText := "📄 " + m.logModalTitle
	if m.logModalHScrollOffset > 0 {
		titleText += " ◀"
	}
	title := titleStyle.Render(titleText)

	// Modal content - truncate each line to viewport width to prevent lipgloss wrap
	var content string
	if m.logModalError != "" {
		content = errorStyle.Render("Error: " + m.logModalError)
	} else if m.logModalLoading {
		content = mutedStyle.Render("Loading...")
	} else {
		// Get viewport content and truncate lines to prevent wrap
		viewportContent := m.logModalViewport.View()
		content = truncateLinesToWidth(viewportContent, m.logModalViewport.Width)
	}

	// Footer with instructions (including horizontal scroll)
	footer := m.logModalFooter()

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

// logModalFooter generates the footer for log modals with horizontal scroll hint
func (m Model) logModalFooter() string {
	baseFooter := "↑↓ scroll • ←→ pan • y copy • esc close"
	if m.statusMessage != "" {
		return mutedStyle.Render(baseFooter) + "  " + temporaryStatusStyle.Render(m.statusMessage)
	}
	return mutedStyle.Render(baseFooter)
}

// truncateLinesToWidth truncates each line to the specified visible width while preserving ANSI codes
func truncateLinesToWidth(content string, maxWidth int) string {
	return processLinesWithANSI(content, 0, maxWidth)
}

// applyHorizontalScroll applies horizontal scroll offset to content while preserving ANSI codes
func applyHorizontalScroll(content string, offset, viewportWidth int) string {
	if offset == 0 {
		return content
	}
	return processLinesWithANSI(content, offset, viewportWidth)
}

// processLinesWithANSI processes content line by line, extracting visible characters
// from startOffset to startOffset+maxWidth while preserving ANSI escape codes
func processLinesWithANSI(content string, startOffset, maxWidth int) string {
	lines := strings.Split(content, "\n")
	result := make([]string, len(lines))

	for i, line := range lines {
		result[i] = sliceLineWithANSI(line, startOffset, maxWidth)
	}

	return strings.Join(result, "\n")
}

// sliceLineWithANSI extracts visible characters from startOffset to startOffset+maxWidth
// while preserving all ANSI escape codes
func sliceLineWithANSI(line string, startOffset, maxWidth int) string {
	if maxWidth <= 0 && startOffset == 0 {
		return line
	}

	// Check if line is too short for the offset
	if startOffset > 0 {
		visibleWidth := ansi.PrintableRuneWidth(line)
		if startOffset >= visibleWidth {
			return ""
		}
	}

	var sb strings.Builder
	printableCount := 0
	inAnsi := false
	runes := []rune(line)
	endOffset := startOffset + maxWidth

	for j := 0; j < len(runes); j++ {
		r := runes[j]

		// Check for ANSI escape sequence start
		if r == '\x1b' && j+1 < len(runes) && runes[j+1] == '[' {
			inAnsi = true
			sb.WriteRune(r) // Always preserve ANSI codes
			continue
		}

		if inAnsi {
			sb.WriteRune(r)
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inAnsi = false
			}
			continue
		}

		// This is a printable character
		if printableCount >= endOffset && maxWidth > 0 {
			break // Reached the end of visible window
		}
		if printableCount >= startOffset {
			sb.WriteRune(r)
		}
		printableCount++
	}

	return sb.String()
}
