package debug

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/lmtani/pumbaa/internal/domain/workflow"
)

func testModel(t *testing.T, width, height int) Model {
	t.Helper()

	wf := &workflow.Workflow{
		ID:     "11111111-2222-3333-4444-555555555555",
		Name:   "Tso500SomaticAnalysis",
		Status: workflow.StatusSucceeded,
	}
	m := NewModel(wf, nil, nil, nil, nil)

	updated, _ := m.Update(tea.WindowSizeMsg{Width: width, Height: height})
	resized, ok := updated.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want Model", updated)
	}
	return resized
}

// TestViewFillsTerminalExactly guards the layout math: header bar, panels and
// footer must add up to exactly the terminal height. A single extra line
// pushes the header bar off the top of the screen (regression: the details
// title used a style with MarginBottom, hiding the header with name/cost).
func TestViewFillsTerminalExactly(t *testing.T) {
	sizes := []struct{ w, h int }{
		{80, 24},
		{120, 40},
		{60, 16},
		{160, 64},
	}

	for _, size := range sizes {
		m := testModel(t, size.w, size.h)
		view := m.View()

		if got := lipgloss.Height(view); got != size.h {
			t.Errorf("View() at %dx%d has height %d, want %d", size.w, size.h, got, size.h)
		}
		if got := lipgloss.Width(view); got > size.w {
			t.Errorf("View() at %dx%d has width %d, want <= %d", size.w, size.h, got, size.w)
		}
	}
}
