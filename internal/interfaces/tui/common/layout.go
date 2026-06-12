package common

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/truncate"
)

// Vertical layout budget shared by all screens. Keeping the math here
// (instead of scattering height-N literals through the views) ensures the
// header, content panels and footer always add up to the terminal height.
const (
	HeaderBarHeight = 1 // single-line header bar
	FooterHeight    = 2 // help bar: top border + one line of hints
	PanelChrome     = 2 // top+bottom border of a main content panel
)

// ContentPanelHeight returns the inner height available to a screen's main
// content panel: terminal height minus header bar, footer and panel borders.
func ContentPanelHeight(termHeight int) int {
	return MaxInt(3, termHeight-HeaderBarHeight-FooterHeight-PanelChrome)
}

// HeaderBrandStyle renders the product chip at the left edge of the header bar.
var HeaderBrandStyle = lipgloss.NewStyle().
	Background(PrimaryColor).
	Foreground(OnPrimaryColor).
	Bold(true).
	Padding(0, 1)

// RenderHeaderBar lays out a single header line: left content anchored at the
// start, right content aligned to the end, flexible space in between. When
// there is not enough room for both, the right side is dropped.
func RenderHeaderBar(width int, left, right string) string {
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		if right != "" {
			return RenderHeaderBar(width, left, "")
		}
		return left
	}
	return left + strings.Repeat(" ", gap) + right
}

// TruncateANSI truncates a styled string to maxWidth display cells while
// preserving ANSI escape sequences (unlike TruncateWidth, which is for plain
// text only).
func TruncateANSI(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	return truncate.String(s, uint(maxWidth))
}

// FitParts joins styled segments with the given separator, keeping only as
// many as fit in maxWidth display cells. Measuring with lipgloss.Width makes
// it safe for ANSI-styled segments, so footers never wrap into extra lines.
func FitParts(maxWidth int, separator string, parts []string) string {
	out := ""
	for _, p := range parts {
		if p == "" {
			continue
		}
		candidate := out
		if candidate != "" {
			candidate += separator
		}
		candidate += p
		if lipgloss.Width(candidate) > maxWidth {
			break
		}
		out = candidate
	}
	return out
}
