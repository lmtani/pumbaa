package debug

import (
	"strings"

	"github.com/muesli/reflow/ansi"
)

// renderLogModal renders the log modal.
func (m Model) renderLogModal() string {
	// Modal title with scroll indicator
	titleText := "📄 " + m.logModalTitle
	if m.logModalHScrollOffset > 0 {
		titleText += " ◀"
	}
	title := titleStyle.Render(titleText)

	// Modal content - truncate each line to viewport width to prevent lipgloss wrap
	viewportContent := m.logModalViewport.View()
	content := renderModalViewportContent(viewportContent, m.logModalViewport.Width, m.logModalLoading, m.logModalError)

	// Footer with instructions (including horizontal scroll)
	footer := m.logModalFooter()

	return m.renderStandardModal(title, content, footer)
}

// logModalFooter generates the footer for log modals with horizontal scroll hint
func (m Model) logModalFooter() string {
	return m.modalFooterWithHints("↑↓ scroll", "←→ pan", "y copy", "esc close")
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
