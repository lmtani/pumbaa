package dashboard

import (
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/lmtani/pumbaa/internal/domain/workflow"
)

func testModel(width, height int) Model {
	return Model{
		width:  width,
		height: height,
		workflows: []workflow.Workflow{
			{ID: "11111111-2222-3333-4444-555555555555", Name: "DemuxFastqOraToUbam", Status: workflow.StatusRunning, SubmittedAt: time.Now()},
			{ID: "66666666-7777-8888-9999-000000000000", Name: "RunDeepVariant", Status: workflow.StatusSucceeded, SubmittedAt: time.Now()},
		},
		totalCount: 2,
	}
}

// TestViewFillsTerminalExactly guards the layout math: header bar, content
// panel and footer must add up to exactly the terminal height, with no line
// overflowing the terminal width.
func TestViewFillsTerminalExactly(t *testing.T) {
	sizes := []struct{ w, h int }{
		{80, 24},
		{120, 40},
		{60, 16},
	}

	for _, size := range sizes {
		m := testModel(size.w, size.h)
		view := m.View()

		if got := lipgloss.Height(view); got != size.h {
			t.Errorf("View() at %dx%d has height %d, want %d", size.w, size.h, got, size.h)
		}
		if got := lipgloss.Width(view); got > size.w {
			t.Errorf("View() at %dx%d has width %d, want <= %d", size.w, size.h, got, size.w)
		}
	}
}

// TestViewEmptyStateFillsTerminal covers the empty-workflows branch.
func TestViewEmptyStateFillsTerminal(t *testing.T) {
	m := testModel(80, 24)
	m.workflows = nil

	view := m.View()
	if got := lipgloss.Height(view); got != 24 {
		t.Errorf("empty View() has height %d, want 24", got)
	}
}
