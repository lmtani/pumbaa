package chat

import (
	"strings"
	"testing"
	"time"

	"google.golang.org/genai"
)

func TestExtractText(t *testing.T) {
	content := &genai.Content{
		Role: "model",
		Parts: []*genai.Part{
			genai.NewPartFromText("hello"),
			genai.NewPartFromText("world"),
		},
	}
	// Multiple text parts are joined with newlines.
	if got := extractText(content); got != "hello\nworld" {
		t.Errorf("extractText = %q, want %q", got, "hello\nworld")
	}

	empty := &genai.Content{Role: "model"}
	if got := extractText(empty); got != "" {
		t.Errorf("extractText on empty content = %q, want empty", got)
	}
}

func TestWrapText(t *testing.T) {
	if got := wrapText("short", 10); got != "short" {
		t.Errorf("short line should be untouched, got %q", got)
	}

	got := wrapText("aaa bbb ccc", 7)
	for _, line := range strings.Split(got, "\n") {
		if len(line) > 7 {
			t.Errorf("line %q exceeds width 7", line)
		}
	}

	// A word longer than the width is force-broken instead of looping forever.
	got = wrapText(strings.Repeat("x", 25), 10)
	if lines := strings.Split(got, "\n"); len(lines) != 3 {
		t.Errorf("expected forced break into 3 lines, got %d: %q", len(lines), got)
	}

	// Zero width falls back to the default instead of panicking.
	if got := wrapText("abc", 0); got != "abc" {
		t.Errorf("zero width should fall back to default, got %q", got)
	}
}

func TestFormatAge(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{30 * time.Second, "just now"},
		{5 * time.Minute, "5m ago"},
		{3 * time.Hour, "3h ago"},
		{49 * time.Hour, "2d ago"},
	}
	for _, c := range cases {
		if got := formatAge(c.d); got != c.want {
			t.Errorf("formatAge(%v) = %q, want %q", c.d, got, c.want)
		}
	}
}

func TestFormatTokenCount(t *testing.T) {
	cases := []struct {
		count int
		want  string
	}{
		{0, "0"},
		{999, "999"},
		{1500, "1.5K"},
		{2300000, "2.3M"},
	}
	for _, c := range cases {
		if got := formatTokenCount(c.count); got != c.want {
			t.Errorf("formatTokenCount(%d) = %q, want %q", c.count, got, c.want)
		}
	}
}
