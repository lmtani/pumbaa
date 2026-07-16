package common

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// FormatDuration formats a duration into a human-readable string.
func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh%dm", h, m)
}

// FormatDurationShort formats a duration without decimals (for dashboard).
func FormatDurationShort(d time.Duration) string {
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

// Truncate truncates a string to maxLen characters with ellipsis.
// It operates on runes, so multi-byte characters are never cut in half.
func Truncate(s string, maxLen int) string {
	if maxLen <= 3 {
		return s
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-3]) + "..."
}

// TruncateWidth truncates a string so its display width fits in maxWidth,
// appending an ellipsis when truncated. Unlike Truncate, it measures
// terminal cells (wide CJK runes count as 2), so table columns stay aligned.
func TruncateWidth(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= maxWidth {
		return s
	}
	target := maxWidth - 1 // reserve one cell for the ellipsis
	var b strings.Builder
	width := 0
	for _, r := range s {
		rw := lipgloss.Width(string(r))
		if width+rw > target {
			break
		}
		b.WriteRune(r)
		width += rw
	}
	return b.String() + "…"
}

// PadRight pads s with spaces up to display width w, truncating first if needed.
func PadRight(s string, w int) string {
	s = TruncateWidth(s, w)
	if d := w - lipgloss.Width(s); d > 0 {
		return s + strings.Repeat(" ", d)
	}
	return s
}

// PadLeft pads s with leading spaces up to display width w, truncating first if needed.
func PadLeft(s string, w int) string {
	s = TruncateWidth(s, w)
	if d := w - lipgloss.Width(s); d > 0 {
		return strings.Repeat(" ", d) + s
	}
	return s
}

// MinInt returns the minimum of two integers.
func MinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// MaxInt returns the maximum of two integers.
func MaxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// WrapText wraps text to fit within maxWidth characters.
func WrapText(text string, maxWidth int) string {
	if maxWidth <= 0 {
		maxWidth = 80
	}

	var result strings.Builder
	lines := strings.Split(text, "\n")

	for i, line := range lines {
		if i > 0 {
			result.WriteString("\n")
		}

		if len(line) <= maxWidth {
			result.WriteString(line)
			continue
		}

		// Wrap long lines
		for len(line) > maxWidth {
			// Try to find a good break point (space)
			breakPoint := maxWidth
			for j := maxWidth; j > maxWidth/2; j-- {
				if line[j] == ' ' {
					breakPoint = j
					break
				}
			}

			result.WriteString(line[:breakPoint])
			result.WriteString("\n")
			line = strings.TrimLeft(line[breakPoint:], " ")
		}
		if len(line) > 0 {
			result.WriteString(line)
		}
	}

	return result.String()
}

// ClipboardCopiedMsg is sent when clipboard copy completes.
type ClipboardCopiedMsg struct {
	Success bool
	Err     error
	Context string // What was copied (e.g., "Docker image", "log content")
}

// CopyToClipboard creates a tea.Cmd that copies text to the system clipboard.
func CopyToClipboard(text, context string) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd

		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("pbcopy")
		case "linux":
			// Try xclip first, then xsel, then wl-copy (for Wayland)
			if _, err := exec.LookPath("xclip"); err == nil {
				cmd = exec.Command("xclip", "-selection", "clipboard")
			} else if _, err := exec.LookPath("xsel"); err == nil {
				cmd = exec.Command("xsel", "--clipboard", "--input")
			} else if _, err := exec.LookPath("wl-copy"); err == nil {
				cmd = exec.Command("wl-copy")
			} else {
				return ClipboardCopiedMsg{Success: false, Err: fmt.Errorf("no clipboard tool found (install xclip, xsel, or wl-copy)"), Context: context}
			}
		case "windows":
			cmd = exec.Command("clip")
		default:
			return ClipboardCopiedMsg{Success: false, Err: fmt.Errorf("unsupported OS: %s", runtime.GOOS), Context: context}
		}

		stdin, err := cmd.StdinPipe()
		if err != nil {
			return ClipboardCopiedMsg{Success: false, Err: err, Context: context}
		}

		if err := cmd.Start(); err != nil {
			return ClipboardCopiedMsg{Success: false, Err: err, Context: context}
		}

		_, err = stdin.Write([]byte(text))
		if err != nil {
			return ClipboardCopiedMsg{Success: false, Err: err, Context: context}
		}
		_ = stdin.Close()

		if err := cmd.Wait(); err != nil {
			return ClipboardCopiedMsg{Success: false, Err: err, Context: context}
		}

		return ClipboardCopiedMsg{Success: true, Context: context}
	}
}
