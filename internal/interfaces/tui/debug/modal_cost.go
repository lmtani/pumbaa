package debug

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/lmtani/pumbaa/internal/domain/workflow"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

// costBreakdownLoadedMsg carries the complete cost breakdown after the fully
// expanded metadata was fetched.
type costBreakdownLoadedMsg struct {
	breakdown *workflow.CostBreakdown
}

// costBreakdownErrorMsg reports a failure to load the expanded metadata for
// the cost breakdown.
type costBreakdownErrorMsg struct {
	err error
}

// openCostModal opens the per-task cost breakdown. It shows the breakdown
// from the already-loaded tree immediately, and — so subworkflow costs are
// included by default — fetches the fully expanded metadata in the background
// (one call) when some subworkflows haven't been loaded, then swaps in the
// complete breakdown. The complete breakdown is cached until the metadata
// changes (watch refresh).
func (m Model) openCostModal() (tea.Model, tea.Cmd) {
	m.activeModal = ModalCost
	m.costViewport = viewport.New(m.width-10, m.height-8)

	var cmd tea.Cmd
	display := m.displayCostBreakdown()
	if m.costBreakdown == nil && display.SubworkflowsPending > 0 && m.fetcher != nil && !m.costLoading {
		m.costLoading = true
		m.costError = ""
		cmd = m.fetchExpandedCostBreakdown()
	}

	m.costViewport.SetContent(m.buildCostContent())
	return m, cmd
}

// displayCostBreakdown returns the complete breakdown when it has been loaded,
// otherwise the partial one computed from the currently loaded tree.
func (m Model) displayCostBreakdown() *workflow.CostBreakdown {
	if m.costBreakdown != nil {
		return m.costBreakdown
	}
	if m.metadata == nil {
		return &workflow.CostBreakdown{}
	}
	return m.metadata.CalculateCostBreakdown()
}

// fetchExpandedCostBreakdown fetches the fully expanded metadata and computes
// the complete cost breakdown off the UI thread.
func (m Model) fetchExpandedCostBreakdown() tea.Cmd {
	fetcher := m.fetcher
	workflowID := m.metadata.ID
	return func() tea.Msg {
		ctx := context.Background()
		data, err := fetcher.GetRawMetadataWithOptions(ctx, workflowID, true)
		if err != nil {
			return costBreakdownErrorMsg{err: err}
		}
		wf, err := fetcher.ParseMetadata(data)
		if err != nil {
			return costBreakdownErrorMsg{err: err}
		}
		return costBreakdownLoadedMsg{breakdown: wf.CalculateCostBreakdown()}
	}
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
// (authoritative) alongside the reconstructed per-task sum, and reports
// whether subworkflow costs are still loading or missing.
func (m Model) buildCostContent() string {
	if m.metadata == nil {
		return mutedStyle.Render("No metadata available")
	}

	breakdown := m.displayCostBreakdown()
	if len(breakdown.Tasks) == 0 {
		if m.costLoading {
			return mutedStyle.Render("⏳ Loading subworkflow costs...")
		}
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
	switch {
	case m.costLoading:
		sb.WriteString(infoNoteStyle.Render("⏳ Loading subworkflow costs — totals will update shortly..."))
		sb.WriteString("\n")
	case m.costError != "":
		sb.WriteString(infoNoteStyle.Render("⚠ Could not load subworkflow costs: " + common.Truncate(m.costError, 60)))
		sb.WriteString("\n")
	case breakdown.SubworkflowsPending > 0:
		sb.WriteString(infoNoteStyle.Render(fmt.Sprintf(
			"⚠ %d subworkflow(s) not included (no server connection to expand).",
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
