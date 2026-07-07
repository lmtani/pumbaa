package dashboard

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	workflowapp "github.com/lmtani/pumbaa/internal/application/workflow"
	"github.com/lmtani/pumbaa/internal/domain/workflow"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

// handleCompareKey implements the two-step compare flow: the first press on a
// workflow marks it as the base; the second press on a different workflow runs
// the comparison; pressing it again on the base clears the mark.
func (m *Model) handleCompareKey() tea.Cmd {
	if m.compareUC == nil {
		m.setStatusMessage("Compare not available")
		return getClearStatusCmd()
	}
	if len(m.workflows) == 0 || m.cursor >= len(m.workflows) {
		return nil
	}
	selected := m.workflows[m.cursor]

	// No base yet: mark this one.
	if m.compareBaseID == "" {
		m.compareBaseID = selected.ID
		m.compareBaseName = selected.Name
		m.setStatusMessage("Marked base: " + truncateID(selected.ID) + " — select another and press c to compare")
		return getClearStatusCmd()
	}

	// Same workflow: unmark.
	if m.compareBaseID == selected.ID {
		m.compareBaseID = ""
		m.compareBaseName = ""
		m.setStatusMessage("Compare base cleared")
		return getClearStatusCmd()
	}

	// Second selection: run the comparison.
	m.showDiff = true
	m.diffLoading = true
	m.diffError = ""
	m.diffResult = nil
	baseID := m.compareBaseID
	otherID := selected.ID
	return tea.Batch(m.spinner.Tick, m.runCompare(baseID, otherID))
}

// runCompare fetches and diffs the two runs off the UI thread.
func (m *Model) runCompare(baseID, otherID string) tea.Cmd {
	uc := m.compareUC
	return func() tea.Msg {
		diff, err := uc.Execute(context.Background(), workflowapp.CompareInput{
			WorkflowIDA:  baseID,
			WorkflowIDB:  otherID,
			ResolveCache: true,
		})
		if err != nil {
			return diffErrorMsg{err: err}
		}
		return diffLoadedMsg{diff: diff}
	}
}

// handleDiffModalKeys handles input while the diff modal is open.
func (m Model) handleDiffModalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "c":
		m.showDiff = false
		m.diffResult = nil
		m.diffError = ""
		// Keep the base marked so the user can compare against another run.
		return m, nil
	case "up", "k":
		m.diffViewport.ScrollUp(1)
	case "down", "j":
		m.diffViewport.ScrollDown(1)
	case "pgup":
		m.diffViewport.PageUp()
	case "pgdown":
		m.diffViewport.PageDown()
	case "g", "home":
		m.diffViewport.GotoTop()
	case "G", "end":
		m.diffViewport.GotoBottom()
	}
	return m, nil
}

// setDiffContent populates the diff viewport once the comparison completes.
func (m *Model) setDiffContent() {
	width := minInt(m.width-8, 110)
	if width < 40 {
		width = maxInt(20, m.width-4)
	}
	m.diffViewport = viewport.New(width-4, maxInt(6, m.height-10))
	m.diffViewport.SetContent(renderDiffBody(m.diffResult, width-6))
}

// renderDiffModal renders the comparison result (or its loading/error state).
func (m Model) renderDiffModal() string {
	width := minInt(m.width-8, 110)
	if width < 40 {
		width = maxInt(20, m.width-4)
	}

	title := common.TitleStyle.Render("Workflow Comparison")

	var body, footer string
	switch {
	case m.diffLoading:
		body = m.spinner.View() + " " + common.MutedStyle.Render("Fetching and comparing both runs...")
		footer = common.MutedStyle.Render("esc cancel")
	case m.diffError != "":
		body = common.ErrorStyle.Render("Comparison failed:") + "\n\n" +
			lipgloss.NewStyle().Foreground(common.ErrorSoftColor).Width(width-6).Render(m.diffError)
		footer = common.MutedStyle.Render("esc close")
	default:
		body = m.diffViewport.View()
		footer = common.MutedStyle.Render("↑↓ scroll · PgUp/PgDn page · esc close")
	}

	content := lipgloss.JoinVertical(lipgloss.Left, title, "", body, "", footer)
	modal := common.ModalStyle.Width(width).Render(content)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal,
		lipgloss.WithWhitespaceChars(" "))
}

// renderDiffBody renders a RunDiff as styled text (mirrors the CLI presenter).
func renderDiffBody(d *workflow.RunDiff, width int) string {
	if d == nil {
		return common.MutedStyle.Render("No comparison available")
	}

	var sb strings.Builder

	sb.WriteString(common.LabelStyle.Render("A: ") + labelOrDash(d.NameA) + " " +
		common.MutedStyle.Render(truncateID(d.IDA)) + " " + statusText(string(d.StatusA)) + "\n")
	sb.WriteString(common.LabelStyle.Render("B: ") + labelOrDash(d.NameB) + " " +
		common.MutedStyle.Render(truncateID(d.IDB)) + " " + statusText(string(d.StatusB)) + "\n")

	if d.NameMismatch {
		sb.WriteString(warnText("⚠ Workflow names differ — comparing runs of different workflows") + "\n")
	}
	if !d.HasDifferences() {
		sb.WriteString("\n" + common.SuccessStyle.Render("✓ No differences found") + "\n")
		return sb.String()
	}

	sb.WriteString(diffKeySection("Inputs", d.Inputs, width))
	sb.WriteString(diffKeySection("Options", d.Options, width))

	// Source
	sb.WriteString("\n" + common.TitleStyle.Render("Source") + "\n")
	if d.SourceChanged {
		sb.WriteString("  " + yellow("~") + fmt.Sprintf(" WDL source changed (%d → %d lines)\n", d.SourceLinesA, d.SourceLinesB))
	} else {
		sb.WriteString("  " + common.MutedStyle.Render("unchanged") + "\n")
	}

	// Tasks
	sb.WriteString("\n" + common.TitleStyle.Render(fmt.Sprintf("Tasks (%d of %d changed)", len(d.Tasks), d.TotalTasks)) + "\n")
	for _, td := range d.Tasks {
		sb.WriteString(diffTaskLines(td))
	}

	return sb.String()
}

func diffKeySection(title string, diffs []workflow.KeyDiff, width int) string {
	var sb strings.Builder
	if len(diffs) == 0 {
		sb.WriteString("\n" + common.TitleStyle.Render(title+" (no changes)") + "\n")
		return sb.String()
	}
	sb.WriteString("\n" + common.TitleStyle.Render(fmt.Sprintf("%s (%d changed)", title, len(diffs))) + "\n")
	for _, kd := range diffs {
		switch kd.Kind {
		case workflow.ChangeAdded:
			sb.WriteString("  " + green("+") + " " + kd.Key + "  " + valuePreview(kd.ValueB, width) + "\n")
		case workflow.ChangeRemoved:
			sb.WriteString("  " + red("-") + " " + kd.Key + "  " + valuePreview(kd.ValueA, width) + "\n")
		default:
			sb.WriteString("  " + yellow("~") + " " + kd.Key + "  " +
				valuePreview(kd.ValueA, width) + " → " + valuePreview(kd.ValueB, width) + "\n")
		}
	}
	return sb.String()
}

func diffTaskLines(td workflow.TaskDiff) string {
	var sb strings.Builder
	switch td.Kind {
	case workflow.ChangeAdded:
		sb.WriteString("  " + green("+") + " " + td.Name + "  " + statusText(td.StatusB) + common.MutedStyle.Render("  (only in B)") + "\n")
	case workflow.ChangeRemoved:
		sb.WriteString("  " + red("-") + " " + td.Name + "  " + statusText(td.StatusA) + common.MutedStyle.Render("  (only in A)") + "\n")
	default:
		sb.WriteString("  " + yellow("~") + " " + td.Name + "\n")
		if td.StatusChanged() {
			sb.WriteString("      " + common.MutedStyle.Render("status:   ") + statusText(td.StatusA) + " → " + statusText(td.StatusB) + "\n")
		}
		if td.DockerChanged() {
			sb.WriteString("      " + common.MutedStyle.Render("docker:   ") + labelOrDash(td.DockerA) + " → " + labelOrDash(td.DockerB) + "\n")
		}
		if td.ShardsChanged() {
			sb.WriteString("      " + common.MutedStyle.Render(fmt.Sprintf("shards:   %d → %d\n", td.ShardsA, td.ShardsB)))
		}
		if td.AttemptsA != td.AttemptsB {
			sb.WriteString("      " + common.MutedStyle.Render(fmt.Sprintf("attempts: %d → %d\n", td.AttemptsA, td.AttemptsB)))
		}
		if td.DurationChangedSignificantly() {
			verdict := durationVerdict(td.DurationRatio())
			sb.WriteString("      " + common.MutedStyle.Render("duration: ") +
				common.FormatDuration(td.DurationA) + " → " + common.FormatDuration(td.DurationB) + "  " + verdict + "\n")
		}
	}
	return sb.String()
}

// --- small styling helpers, local to the dashboard diff view ---

func statusText(s string) string {
	if s == "" {
		return common.MutedStyle.Render("-")
	}
	return common.StatusStyle(s).Render(s)
}

func labelOrDash(s string) string {
	if s == "" {
		return common.MutedStyle.Render("-")
	}
	return s
}

func valuePreview(v string, width int) string {
	if v == "" {
		return common.MutedStyle.Render("(absent)")
	}
	maxLen := 50
	if width > 60 {
		maxLen = width - 10
	}
	r := []rune(v)
	if len(r) > maxLen {
		return string(r[:maxLen-1]) + "…"
	}
	return v
}

func durationVerdict(ratio float64) string {
	if ratio > 1 {
		return lipgloss.NewStyle().Foreground(common.StatusFailed).Render(fmt.Sprintf("%.1f× slower", ratio))
	}
	if ratio > 0 {
		return common.SuccessStyle.Render(fmt.Sprintf("%.1f× faster", 1/ratio))
	}
	return common.MutedStyle.Render("changed")
}

func green(s string) string  { return common.SuccessStyle.Render(s) }
func red(s string) string    { return common.ErrorStyle.Render(s) }
func yellow(s string) string { return lipgloss.NewStyle().Foreground(common.WarningColor).Render(s) }
func warnText(s string) string {
	return lipgloss.NewStyle().Foreground(common.WarningColor).Render(s)
}
