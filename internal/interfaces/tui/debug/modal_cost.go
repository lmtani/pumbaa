package debug

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/lmtani/pumbaa/internal/domain/workflow"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

// openCostModal builds the per-task cost breakdown from the loaded metadata
// and opens it in a scrollable modal.
func (m Model) openCostModal() (tea.Model, tea.Cmd) {
	m.activeModal = ModalCost
	m.costViewport = viewport.New(m.width-10, m.height-8)
	m.costViewport.SetContent(m.buildCostContent())
	return m, nil
}

func (m Model) renderCostModal() string {
	title := titleStyle.Render("Cost by Task (most expensive first)")
	content := renderModalViewportContent(m.costViewport.View(), m.costViewport.Width, false, "")
	footer := m.modalFooterWithHints("↑↓ scroll", "PgUp/PgDn page", "esc close")
	return m.renderStandardModal(title, content, footer)
}

func (m Model) handleCostModalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	actions := viewportModalActions{
		onClose: func(m *Model) { m.activeModal = ModalNone },
	}
	cmd, handled := m.handleViewportModalKeys(msg, &m.costViewport, actions)
	if handled {
		return m, cmd
	}
	return m, nil
}

// buildCostContent renders the cost breakdown table. It shows the API total
// (authoritative) alongside the reconstructed per-task sum, so a gap from
// unexpanded subworkflows is visible rather than silently wrong.
func (m Model) buildCostContent() string {
	if m.metadata == nil {
		return mutedStyle.Render("No metadata available")
	}

	breakdown := m.metadata.CalculateCostBreakdown()
	if len(breakdown.Tasks) == 0 {
		return mutedStyle.Render("No per-task cost data available (tasks have no VM cost/timing yet)")
	}

	var sb strings.Builder

	// Header: authoritative API total + reconstructed coverage
	if m.totalCost > 0 {
		sb.WriteString(labelStyle.Render("Workflow total (API): "))
		sb.WriteString(costBadgeStyle.Render(fmt.Sprintf("$%.2f", m.totalCost)))
		sb.WriteString("\n")
	}
	sb.WriteString(mutedStyle.Render(fmt.Sprintf("Accounted here: $%.2f across %d task(s)",
		breakdown.TotalCost, len(breakdown.Tasks))))
	if !breakdown.FromActual {
		sb.WriteString(mutedStyle.Render("  (some values estimated from resources)"))
	}
	sb.WriteString("\n")
	if breakdown.SubworkflowsPending > 0 {
		sb.WriteString(infoNoteStyle.Render(fmt.Sprintf(
			"⚠ %d subworkflow(s) not expanded — their tasks are missing. Press f or open them to include.",
			breakdown.SubworkflowsPending)))
		sb.WriteString("\n")
	}
	sb.WriteString("\n")

	// Column widths
	maxNameLen := 20
	for _, t := range breakdown.Tasks {
		if len(t.Name) > maxNameLen {
			maxNameLen = len(t.Name)
		}
	}
	if maxNameLen > 44 {
		maxNameLen = 44
	}

	maxCost := breakdown.Tasks[0].TotalCost // sorted desc

	for _, t := range breakdown.Tasks {
		sb.WriteString(m.formatCostRow(t, maxNameLen, maxCost))
		sb.WriteString("\n")
	}

	// Optimization hint: the biggest non-preemptible task is the prime target
	if tip := costOptimizationHint(breakdown); tip != "" {
		sb.WriteString("\n")
		sb.WriteString(infoNoteStyle.Render(tip))
		sb.WriteString("\n")
	}

	return sb.String()
}

func (m Model) formatCostRow(t workflow.TaskCost, maxNameLen int, maxCost float64) string {
	name := common.PadRight(t.Name, maxNameLen)

	cost := fmt.Sprintf("$%7.2f", t.TotalCost)
	pct := fmt.Sprintf("%5.1f%%", t.Percent)

	// Cost bar proportional to the most expensive task
	barWidth := 20
	filled := 0
	if maxCost > 0 {
		filled = int(t.TotalCost / maxCost * float64(barWidth))
	}
	if filled > barWidth {
		filled = barWidth
	}
	bar := mutedStyle.Render("[") +
		valueStyle.Render(strings.Repeat("█", filled)) +
		mutedStyle.Render(strings.Repeat("░", barWidth-filled)+"]")

	// Preemptible marker: flag expensive non-preemptible tasks (optimization target)
	marker := "  "
	if !t.Preemptible {
		if t.Percent >= 15 {
			marker = warnMarkerStyle.Render("◆ ") // costly & on-demand: worth making preemptible
		} else {
			marker = mutedStyle.Render("· ")
		}
	}

	meta := mutedStyle.Render(fmt.Sprintf("%.1fh", t.VMHours))
	if t.ShardCount > 1 {
		meta += mutedStyle.Render(fmt.Sprintf(" ×%d", t.ShardCount))
	}
	if !t.Preemptible {
		meta += mutedStyle.Render(" on-demand")
	} else {
		meta += mutedStyle.Render(" preempt")
	}

	return fmt.Sprintf("%s%s  %s %s  %s  %s",
		marker, name,
		valueStyle.Render(cost), mutedStyle.Render(pct),
		bar, meta)
}

// costOptimizationHint points at the biggest on-demand task, the usual
// candidate for switching to preemptible VMs.
func costOptimizationHint(b *workflow.CostBreakdown) string {
	for _, t := range b.Tasks {
		if !t.Preemptible && t.Percent >= 15 {
			return fmt.Sprintf("◆ %s is %.0f%% of cost on on-demand VMs (%.1fh) — making it preemptible could cut cost substantially.",
				t.Name, t.Percent, t.VMHours)
		}
	}
	return ""
}

// warnMarkerStyle highlights expensive on-demand tasks.
var warnMarkerStyle = lipgloss.NewStyle().Foreground(common.StatusRunning).Bold(true)
