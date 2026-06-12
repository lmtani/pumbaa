package dashboard

import (
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/lmtani/pumbaa/internal/domain/workflow"
)

func testModel(width, height int) Model {
	m := NewModel()
	m.width = width
	m.height = height
	m.workflows = []workflow.Workflow{
		{ID: "11111111-2222-3333-4444-555555555555", Name: "DemuxFastqOraToUbam", Status: workflow.StatusRunning, SubmittedAt: time.Now()},
		{ID: "66666666-7777-8888-9999-000000000000", Name: "RunDeepVariant", Status: workflow.StatusSucceeded, SubmittedAt: time.Now(), Labels: map[string]string{"project": "genomics"}},
	}
	m.allWorkflows = m.workflows
	m.totalCount = 2
	return m
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

// TestViewWithFilterBarFillsTerminal ensures the inline filter bar steals one
// line from the content panel instead of growing the screen.
func TestViewWithFilterBarFillsTerminal(t *testing.T) {
	m := testModel(80, 24)
	m.showFilter = true
	m.filterType = "name"

	view := m.View()
	if got := lipgloss.Height(view); got != 24 {
		t.Errorf("View() with filter bar has height %d, want 24", got)
	}
}

func TestApplyLocalFilter(t *testing.T) {
	m := testModel(80, 24)

	m.filterType = "name"
	m.filterInput.SetValue("deep")
	m.applyLocalFilter()
	if len(m.workflows) != 1 || m.workflows[0].Name != "RunDeepVariant" {
		t.Errorf("name filter %q matched %d workflows, want RunDeepVariant only", "deep", len(m.workflows))
	}

	m.filterType = "label"
	m.filterInput.SetValue("project:gen")
	m.applyLocalFilter()
	if len(m.workflows) != 1 || m.workflows[0].Name != "RunDeepVariant" {
		t.Errorf("label filter matched %d workflows, want 1", len(m.workflows))
	}

	m.filterInput.SetValue("")
	m.applyLocalFilter()
	if len(m.workflows) != 2 {
		t.Errorf("empty filter shows %d workflows, want full list (2)", len(m.workflows))
	}
}
