package common

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"short string unchanged", "abc", 10, "abc"},
		{"exact length unchanged", "abcdefghij", 10, "abcdefghij"},
		{"long string truncated", "abcdefghijk", 10, "abcdefg..."},
		{"maxLen too small returns input", "abcdef", 3, "abcdef"},
		{"multi-byte runes not split", "ação genômica de variantes", 10, "ação ge..."},
		{"emoji not split", "🧬🧬🧬🧬🧬🧬🧬🧬🧬🧬🧬", 10, "🧬🧬🧬🧬🧬🧬🧬..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Truncate(tt.input, tt.maxLen)
			if got != tt.expected {
				t.Errorf("Truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.expected)
			}
		})
	}
}

func TestTruncateWidth(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxWidth int
		expected string
	}{
		{"fits unchanged", "abc", 5, "abc"},
		{"exact width unchanged", "abcde", 5, "abcde"},
		{"truncated with ellipsis", "abcdefgh", 5, "abcd…"},
		{"zero width returns empty", "abc", 0, ""},
		{"multi-byte runes not split", "ação genômica", 6, "ação …"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TruncateWidth(tt.input, tt.maxWidth)
			if got != tt.expected {
				t.Errorf("TruncateWidth(%q, %d) = %q, want %q", tt.input, tt.maxWidth, got, tt.expected)
			}
			if w := lipgloss.Width(got); w > tt.maxWidth {
				t.Errorf("TruncateWidth(%q, %d) has display width %d, want <= %d", tt.input, tt.maxWidth, w, tt.maxWidth)
			}
		})
	}
}

func TestPadRightAndPadLeft(t *testing.T) {
	// Wide runes (CJK) occupy two terminal cells; padded cells must always
	// land on the exact target display width so table columns stay aligned.
	inputs := []string{"abc", "ação", "数据流分析", "a-very-long-workflow-name"}
	const width = 10

	for _, in := range inputs {
		right := PadRight(in, width)
		if w := lipgloss.Width(right); w != width {
			t.Errorf("PadRight(%q, %d) has display width %d, want %d", in, width, w, width)
		}

		left := PadLeft(in, width)
		if w := lipgloss.Width(left); w != width {
			t.Errorf("PadLeft(%q, %d) has display width %d, want %d", in, width, w, width)
		}
		if !strings.HasPrefix(left, " ") && lipgloss.Width(in) < width {
			t.Errorf("PadLeft(%q, %d) = %q, expected leading spaces", in, width, left)
		}
	}
}
