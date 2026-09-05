package dashboard

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	workflowapp "github.com/lmtani/pumbaa/internal/application/workflow"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

// historyPageSize bounds both the query and the modal: a history is read from
// the recent end, and a screenful at a time is how it gets read.
const historyPageSize = 100

// historyVisibleRows is how many entries the modal shows at once.
const historyVisibleRows = 12

// openHistoryModal shows what this machine remembers, including the runs the
// server has forgotten — the ones the dashboard itself can never list.
func (m Model) openHistoryModal() (tea.Model, tea.Cmd) {
	if m.historyUC == nil {
		return m, nil
	}
	m.showHistoryModal = true
	m.historyLoading = true
	m.historyError = ""
	m.historyCursor = 0
	m.historyScroll = 0
	return m, tea.Batch(m.spinner.Tick, m.fetchHistory())
}

// fetchHistory loads the local history of the active host, refreshing statuses
// so runs that disappeared from the server can be marked as such.
func (m Model) fetchHistory() tea.Cmd {
	uc := m.historyUC
	return func() tea.Msg {
		out, err := uc.List(context.Background(), workflowapp.ListRunHistoryInput{
			Limit:   historyPageSize,
			Refresh: true,
		})
		if err != nil {
			return historyLoadedMsg{err: err}
		}
		return historyLoadedMsg{entries: out.Entries, refreshErr: out.RefreshError}
	}
}

// handleHistoryModalKeys processes keyboard input in the history modal.
func (m Model) handleHistoryModalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.historyLoading {
		if msg.String() == "esc" {
			m.closeHistoryModal()
		}
		return m, nil
	}

	switch msg.String() {
	case "esc":
		m.closeHistoryModal()
		return m, nil

	case "up", "k":
		if m.historyCursor > 0 {
			m.historyCursor--
			m.ensureHistoryVisible()
		}

	case "down", "j":
		if m.historyCursor < len(m.historyEntries)-1 {
			m.historyCursor++
			m.ensureHistoryVisible()
		}

	case "pgup":
		m.historyCursor = maxInt(0, m.historyCursor-historyVisibleRows)
		m.ensureHistoryVisible()

	case "pgdown":
		m.historyCursor = minInt(len(m.historyEntries)-1, m.historyCursor+historyVisibleRows)
		m.ensureHistoryVisible()

	case "home":
		m.historyCursor, m.historyScroll = 0, 0

	case "end":
		m.historyCursor = maxInt(0, len(m.historyEntries)-1)
		m.ensureHistoryVisible()

	case "r":
		m.historyLoading = true
		return m, tea.Batch(m.spinner.Tick, m.fetchHistory())

	case "n":
		entry, ok := m.selectedHistoryEntry()
		if !ok {
			return m, nil
		}
		// The note editor takes over the screen; coming back to the history
		// is what the reader expects after writing in it.
		m.showHistoryModal = false
		m.noteReturnToHistory = true
		return m.openNoteModalFor(entry.Record.WorkflowID, entry.Record.Name)

	case "enter":
		entry, ok := m.selectedHistoryEntry()
		if !ok {
			return m, nil
		}
		if entry.Forgotten {
			m.historyError = "The server no longer has this run; there is nothing to debug."
			return m, nil
		}
		m.closeHistoryModal()
		m.loadingDebug = true
		m.loadingDebugID = entry.Record.WorkflowID
		return m, tea.Batch(m.spinner.Tick, m.fetchDebugMetadata(entry.Record.WorkflowID))
	}

	return m, nil
}

func (m *Model) closeHistoryModal() {
	m.showHistoryModal = false
	m.historyError = ""
}

func (m Model) selectedHistoryEntry() (workflowapp.RunHistoryEntry, bool) {
	if m.historyCursor < 0 || m.historyCursor >= len(m.historyEntries) {
		return workflowapp.RunHistoryEntry{}, false
	}
	return m.historyEntries[m.historyCursor], true
}

func (m *Model) ensureHistoryVisible() {
	if m.historyCursor < m.historyScroll {
		m.historyScroll = m.historyCursor
	}
	if m.historyCursor >= m.historyScroll+historyVisibleRows {
		m.historyScroll = m.historyCursor - historyVisibleRows + 1
	}
}

// renderHistoryModal draws the local run history.
func (m Model) renderHistoryModal() string {
	width := minInt(120, maxInt(50, m.width-4))
	var body strings.Builder

	switch {
	case m.historyLoading:
		body.WriteString(m.spinner.View() + common.MutedStyle.Render(" Reading the local history..."))
	case len(m.historyEntries) == 0:
		body.WriteString(common.MutedStyle.Render("Nothing remembered for this host yet.\n\n") +
			common.DescStyle.Render("Runs are remembered when submitted through pumbaa, and ") +
			common.KeyStyle.Render("n") +
			common.DescStyle.Render(" writes a note about any run."))
	default:
		body.WriteString(m.renderHistoryRows(width - 6))
	}

	if m.historyError != "" {
		body.WriteString("\n\n" + common.MutedStyle.Render("⚠ "+m.historyError))
	}

	body.WriteString("\n\n")
	body.WriteString(common.KeyStyle.Render("↑↓") + common.DescStyle.Render(" navigate") + "   " +
		common.KeyStyle.Render("enter") + common.DescStyle.Render(" debug") + "   " +
		common.KeyStyle.Render("n") + common.DescStyle.Render(" note") + "   " +
		common.KeyStyle.Render("r") + common.DescStyle.Render(" reload") + "   " +
		common.KeyStyle.Render("esc") + common.DescStyle.Render(" close"))

	modal := common.ModalStyle.
		Width(width).
		Render(lipgloss.JoinVertical(lipgloss.Left,
			common.HeaderTitleStyle.Render("Local history"),
			"",
			body.String(),
		))

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal,
		lipgloss.WithWhitespaceChars(" "))
}

// renderHistoryRows lays the entries out, giving the description whatever the
// fixed columns leave: it is the column being read.
func (m Model) renderHistoryRows(width int) string {
	const (
		statusWidth    = 12
		idWidth        = 9
		submittedWidth = 15
		nameWidth      = 22
	)
	descriptionWidth := maxInt(10, width-statusWidth-idWidth-submittedWidth-nameWidth-8)

	var b strings.Builder
	end := minInt(m.historyScroll+historyVisibleRows, len(m.historyEntries))
	for i := m.historyScroll; i < end; i++ {
		entry := m.historyEntries[i]
		status := entry.Status
		if entry.Forgotten {
			status += "*"
		}

		cells := []string{
			common.PadRight(common.StatusIcon(entry.Status)+" "+status, statusWidth),
			common.PadRight(truncateID(entry.Record.WorkflowID), idWidth),
			common.PadRight(common.Truncate(entry.Record.Name, nameWidth), nameWidth),
			common.PadRight(entry.Record.SubmittedAt.Local().Format("06-01-02 15:04"), submittedWidth),
			common.Truncate(entry.Record.Description, descriptionWidth),
		}

		if i == m.historyCursor {
			row := common.TruncateWidth(strings.Join(cells, "  "), width)
			if pad := width - lipgloss.Width(row); pad > 0 {
				row += strings.Repeat(" ", pad)
			}
			b.WriteString(lipgloss.NewStyle().
				Background(common.HighlightColor).
				Foreground(common.TextColor).
				Render(row))
		} else {
			b.WriteString(common.TruncateANSI(strings.Join([]string{
				common.StatusStyle(entry.Status).Render(cells[0]),
				common.MutedStyle.Render(cells[1]),
				common.ValueStyle.Render(cells[2]),
				common.MutedStyle.Render(cells[3]),
				common.DescStyle.Render(cells[4]),
			}, "  "), width))
		}
		b.WriteString("\n")
	}

	if len(m.historyEntries) > historyVisibleRows {
		b.WriteString(common.MutedStyle.Render(
			fmt.Sprintf("\n  %d-%d of %d", m.historyScroll+1, end, len(m.historyEntries))))
	}
	if m.historyHasForgotten() {
		b.WriteString(common.MutedStyle.Render("\n  * the server no longer knows this run"))
	}
	return b.String()
}

func (m Model) historyHasForgotten() bool {
	for _, entry := range m.historyEntries {
		if entry.Forgotten {
			return true
		}
	}
	return false
}
