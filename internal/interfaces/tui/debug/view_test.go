package debug

import (
	"strings"
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

// TestWorkflowPanelShowsTheLocalNote covers the run's description in the
// workflow panel, including the wrapping a long one needs.
func TestWorkflowPanelShowsTheLocalNote(t *testing.T) {
	m := testModel(t, 100, 30)
	updated, _ := m.Update(runNoteLoadedMsg{
		description: "reprocessing the January cohort after the reference panel was rebuilt, to check whether the ancestry step still disagrees with the clinical call",
	})
	m, ok := updated.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want Model", updated)
	}

	view := m.View()
	if !strings.Contains(view, "Note") {
		t.Error("workflow panel has no Note section")
	}
	if !strings.Contains(view, "reprocessing the January cohort") {
		t.Error("workflow panel does not show the note")
	}
	if got := lipgloss.Height(view); got != 30 {
		t.Errorf("view height %d, want 30 — a long note must wrap inside the panel, not grow it", got)
	}
	if got := lipgloss.Width(view); got > 100 {
		t.Errorf("view width %d, want <= 100", got)
	}
}

// TestWorkflowPanelWithoutANoteHasNoSection keeps the panel quiet for runs
// nobody wrote about, which is most of them.
func TestWorkflowPanelWithoutANoteHasNoSection(t *testing.T) {
	m := testModel(t, 100, 30)

	if strings.Contains(m.View(), "Note") {
		t.Error("Note section rendered for a run with no local record")
	}
}

func TestNoteWrapWidthHasAFloor(t *testing.T) {
	if got := noteWrapWidth(10); got != 20 {
		t.Errorf("noteWrapWidth(10) = %d, want the floor of 20", got)
	}
	if got := noteWrapWidth(60); got != 54 {
		t.Errorf("noteWrapWidth(60) = %d, want the panel width less its padding", got)
	}
}
