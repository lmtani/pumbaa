package dashboard

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/lmtani/pumbaa/internal/application/ports"
	workflowapp "github.com/lmtani/pumbaa/internal/application/workflow"
)

func historyModel(width, height int) Model {
	m := testModel(width, height)
	m.historyUC = &workflowapp.RunHistoryUseCase{}
	m.showHistoryModal = true
	submitted := time.Date(2026, 7, 12, 9, 30, 0, 0, time.UTC)
	m.historyEntries = []workflowapp.RunHistoryEntry{
		{
			Record: ports.RunRecord{WorkflowID: "11111111-aaaa", Name: "DemuxFastqOraToUbam", Description: "the rerun with the fixed reference", SubmittedAt: submitted},
			Status: "Succeeded",
		},
		{
			Record:    ports.RunRecord{WorkflowID: "33333333-cccc", Name: "SomalierAncestry", Description: "the server restarted after this", SubmittedAt: submitted},
			Status:    "Succeeded",
			Forgotten: true,
		},
	}
	return m
}

func pressKey(name string) tea.KeyMsg {
	if len(name) == 1 {
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(name)}
	}
	switch name {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	default:
		panic("unmapped key: " + name)
	}
}

func TestHistoryModalShowsRunsTheServerForgot(t *testing.T) {
	m := historyModel(120, 26)

	view := m.View()
	if !strings.Contains(view, "the server restarted after this") {
		t.Error("modal does not list the run the dashboard itself cannot show")
	}
	if !strings.Contains(view, "the server no longer knows this run") {
		t.Error("modal does not explain the * marker")
	}
	if got := lipgloss.Height(view); got != 26 {
		t.Errorf("modal view has height %d, want 26", got)
	}
	if got := lipgloss.Width(view); got > 120 {
		t.Errorf("modal view has width %d, want <= 120", got)
	}
}

func TestHistoryModalRefusesToDebugAForgottenRun(t *testing.T) {
	m := historyModel(120, 26)
	m.historyCursor = 1

	updated, _ := m.handleHistoryModalKeys(pressKey("enter"))
	after, ok := updated.(Model)
	if !ok {
		t.Fatalf("got %T, want Model", updated)
	}
	if after.loadingDebug {
		t.Error("started loading metadata for a run the server no longer has")
	}
	if after.historyError == "" {
		t.Error("no explanation given for refusing to open the run")
	}
	if !after.showHistoryModal {
		t.Error("modal closed on a refused action")
	}
}

func TestHistoryModalOpensDebugForALiveRun(t *testing.T) {
	m := historyModel(120, 26)

	updated, cmd := m.handleHistoryModalKeys(pressKey("enter"))
	after := updated.(Model)
	if !after.loadingDebug || after.loadingDebugID != "11111111-aaaa" {
		t.Errorf("loadingDebug=%v id=%q, want the selected run being opened", after.loadingDebug, after.loadingDebugID)
	}
	if after.showHistoryModal {
		t.Error("modal stayed open while navigating away")
	}
	if cmd == nil {
		t.Error("no command issued to fetch the metadata")
	}
}

func TestHistoryModalNoteReturnsToTheHistory(t *testing.T) {
	m := historyModel(120, 26)

	updated, _ := m.handleHistoryModalKeys(pressKey("n"))
	after := updated.(Model)
	if !after.showNoteModal || after.showHistoryModal {
		t.Errorf("showNote=%v showHistory=%v, want the note editor to take over", after.showNoteModal, after.showHistoryModal)
	}
	if !after.noteReturnToHistory {
		t.Error("the way back to the history was not remembered")
	}
	if after.noteInput.Value() != "the rerun with the fixed reference" {
		t.Errorf("input = %q, want the existing note of the selected entry", after.noteInput.Value())
	}

	// Saving from there must land back on the history rather than the table.
	saved, _ := after.Update(noteSavedMsg{workflowID: "11111111-aaaa", description: "edited"})
	back := saved.(Model)
	if !back.showHistoryModal || back.showNoteModal {
		t.Errorf("showHistory=%v showNote=%v, want the history back after saving", back.showHistoryModal, back.showNoteModal)
	}
}

func TestHistoryModalNavigationStaysInBounds(t *testing.T) {
	m := historyModel(120, 26)

	updated, _ := m.handleHistoryModalKeys(pressKey("down"))
	after := updated.(Model)
	updated, _ = after.handleHistoryModalKeys(pressKey("down"))
	after = updated.(Model)

	if after.historyCursor != len(m.historyEntries)-1 {
		t.Errorf("cursor = %d, want it clamped to the last entry", after.historyCursor)
	}
}
