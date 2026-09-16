package dashboard

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

// noteCharLimit is generous on purpose: this is the one place a run gets
// described in sentences rather than tags.
const noteCharLimit = 500

// openNoteModal starts editing the local note of the selected run, prefilled
// with whatever is already remembered about it.
func (m Model) openNoteModal() (tea.Model, tea.Cmd) {
	if m.historyUC == nil || len(m.workflows) == 0 || m.cursor >= len(m.workflows) {
		return m, nil
	}
	wf := m.workflows[m.cursor]
	return m.openNoteModalFor(wf.ID, wf.Name)
}

// openNoteModalFor edits the note of a specific run, which is how the history
// modal reaches runs the dashboard itself cannot list.
func (m Model) openNoteModalFor(workflowID, name string) (tea.Model, tea.Cmd) {
	if m.historyUC == nil {
		return m, nil
	}

	input := textinput.New()
	input.Placeholder = "what this run is for..."
	input.CharLimit = noteCharLimit
	input.Width = minInt(60, maxInt(20, m.width-16))
	input.SetValue(m.noteFor(workflowID))
	input.CursorEnd()
	input.Focus()

	m.showNoteModal = true
	m.noteWorkflowID = workflowID
	m.noteWorkflowName = name
	m.noteInput = input
	m.noteMessage = ""
	return m, textinput.Blink
}

// noteFor returns the note already written about a run, looking in whichever
// listing knows about it.
func (m Model) noteFor(workflowID string) string {
	if rec, ok := m.notes[workflowID]; ok {
		return rec.Description
	}
	for _, entry := range m.historyEntries {
		if entry.Record.WorkflowID == workflowID {
			return entry.Record.Description
		}
	}
	return ""
}

// handleNoteModalKeys processes keyboard input in the note modal.
func (m Model) handleNoteModalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.noteSaving {
		return m, nil
	}

	switch msg.String() {
	case "esc":
		m.closeNoteModal()
		return m, nil

	case "enter":
		description := strings.TrimSpace(m.noteInput.Value())
		m.noteSaving = true
		return m, tea.Batch(m.spinner.Tick, m.saveNote(m.noteWorkflowID, description))
	}

	var cmd tea.Cmd
	m.noteInput, cmd = m.noteInput.Update(msg)
	return m, cmd
}

func (m *Model) closeNoteModal() {
	m.showNoteModal = false
	m.noteWorkflowID = ""
	m.noteWorkflowName = ""
	m.noteMessage = ""
	m.noteSaving = false
}

// renderNoteModal draws the note editor.
func (m Model) renderNoteModal() string {
	var body strings.Builder

	body.WriteString(common.MutedStyle.Render(truncateID(m.noteWorkflowID)+"  ") +
		common.ValueStyle.Render(m.noteWorkflowName) + "\n\n")
	body.WriteString(m.noteInput.View() + "\n\n")

	switch {
	case m.noteSaving:
		body.WriteString(m.spinner.View() + common.MutedStyle.Render(" Saving..."))
	case m.noteMessage != "":
		body.WriteString(common.MutedStyle.Render(m.noteMessage))
	default:
		body.WriteString(common.MutedStyle.Render("Kept on this machine only — Cromwell never sees it."))
	}
	body.WriteString("\n\n")
	body.WriteString(common.KeyStyle.Render("enter") + common.DescStyle.Render(" save") + "   " +
		common.KeyStyle.Render("esc") + common.DescStyle.Render(" cancel"))

	modal := common.ModalStyle.
		Width(minInt(72, maxInt(40, m.width-8))).
		Render(lipgloss.JoinVertical(lipgloss.Left,
			common.HeaderTitleStyle.Render("Note"),
			"",
			body.String(),
		))

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		modal,
		lipgloss.WithWhitespaceChars(" "),
	)
}
