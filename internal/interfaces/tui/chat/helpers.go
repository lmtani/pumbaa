package chat

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"google.golang.org/genai"
)

// extractText extracts text parts from a genai.Content.
func extractText(content *genai.Content) string {
	var texts []string
	for _, part := range content.Parts {
		if part.Text != "" {
			texts = append(texts, part.Text)
		}
	}
	return strings.Join(texts, "\n")
}

// renderMarkdown renders markdown content using glamour.
func renderMarkdown(content string, width int) string {
	if width <= 20 {
		width = 80
	}

	renderer, err := glamour.NewTermRenderer(
		glamour.WithStylePath("dark"),
		glamour.WithWordWrap(width),
		glamour.WithEmoji(),
	)
	if err != nil {
		return content
	}

	rendered, err := renderer.Render(content)
	if err != nil {
		return content
	}

	return strings.TrimSpace(rendered)
}

// copyToClipboard creates a tea.Cmd that copies text to the system clipboard.
func copyToClipboard(text string) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd

		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("pbcopy")
		case "linux":
			if _, err := exec.LookPath("xclip"); err == nil {
				cmd = exec.Command("xclip", "-selection", "clipboard")
			} else if _, err := exec.LookPath("xsel"); err == nil {
				cmd = exec.Command("xsel", "--clipboard", "--input")
			} else if _, err := exec.LookPath("wl-copy"); err == nil {
				cmd = exec.Command("wl-copy")
			} else {
				return clipboardCopiedMsg{success: false, err: fmt.Errorf("no clipboard tool found")}
			}
		default:
			return clipboardCopiedMsg{success: false, err: fmt.Errorf("unsupported OS: %s", runtime.GOOS)}
		}

		stdin, err := cmd.StdinPipe()
		if err != nil {
			return clipboardCopiedMsg{success: false, err: err}
		}

		if err := cmd.Start(); err != nil {
			return clipboardCopiedMsg{success: false, err: err}
		}

		_, err = stdin.Write([]byte(text))
		if err != nil {
			return clipboardCopiedMsg{success: false, err: err}
		}
		stdin.Close()

		if err := cmd.Wait(); err != nil {
			return clipboardCopiedMsg{success: false, err: err}
		}

		return clipboardCopiedMsg{success: true}
	}
}

// wrapText wraps text to the given width.
func wrapText(text string, width int) string {
	if width <= 0 {
		width = 80
	}

	var result strings.Builder
	lines := strings.Split(text, "\n")

	for i, line := range lines {
		if i > 0 {
			result.WriteString("\n")
		}

		// Handle lines that are too long
		for len(line) > width {
			// Find a good break point
			breakPoint := width
			for breakPoint > 0 && line[breakPoint] != ' ' {
				breakPoint--
			}
			if breakPoint == 0 {
				breakPoint = width // Force break if no space found
			}

			result.WriteString(line[:breakPoint])
			result.WriteString("\n")
			line = strings.TrimLeft(line[breakPoint:], " ")
		}
		result.WriteString(line)
	}

	return result.String()
}
